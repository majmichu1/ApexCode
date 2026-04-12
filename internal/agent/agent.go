package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/apexcode/apexcode/internal/config"
	"github.com/apexcode/apexcode/internal/memory"
	"github.com/apexcode/apexcode/internal/providers"
	"github.com/apexcode/apexcode/internal/tools"
)

// Agent represents the AI coding agent
// Implements the proper while(true) loop pattern from opencode
type Agent struct {
	config         *config.Config
	provider       providers.Provider
	memory         *memory.MemPalace
	toolList       []tools.Tool
	toolMap        map[string]tools.Tool
	messages       []providers.Message
	maxTurns       int
	currentTurn    int
	workDir        string
	streamCallback func(string)
	compactCallback func(string)
	// Context management
	maxInputTokens int
	maxOutputTokens int
	tokenUsage     *TokenUsage
	// Compaction
	needsCompaction bool
	compactBoundary int
	// File tracking
	chatFiles     map[string]bool // Files currently in chat
	readonlyFiles map[string]bool // Read-only files in context
	// Permission system
	permissionRules []*PermissionRule
}

// TokenUsage tracks token consumption
type TokenUsage struct {
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
	CostUSD          float64
}

// PermissionRule represents an allow/deny/ask rule
type PermissionRule struct {
	Permission string // "bash", "edit", "read", etc.
	Pattern    string // glob pattern
	Action     string // "allow", "deny", "ask"
}

// New creates a new agent
func New(cfg *config.Config) *Agent {
	mem, err := memory.NewMemPalace(fmt.Sprintf("%s/.apexcode/memory", cfg.WorkDir))
	if err != nil {
		fmt.Printf("Warning: failed to initialize memory palace: %v\n", err)
	}

	toolList := []tools.Tool{
		tools.NewBashTool(cfg.WorkDir, 120*time.Second),
		tools.NewFileReadTool(cfg.WorkDir),
		tools.NewFileWriteTool(cfg.WorkDir),
		tools.NewFileEditTool(cfg.WorkDir),
		tools.NewGrepTool(cfg.WorkDir),
		tools.NewGlobTool(cfg.WorkDir),
		tools.NewWebFetchTool(),
	}

	toolMap := make(map[string]tools.Tool)
	for _, t := range toolList {
		toolMap[t.Name()] = t
	}

	return &Agent{
		config:        cfg,
		memory:        mem,
		toolList:      toolList,
		toolMap:       toolMap,
		maxTurns:      cfg.MaxTurns,
		workDir:       cfg.WorkDir,
		messages:      make([]providers.Message, 0),
		chatFiles:     make(map[string]bool),
		readonlyFiles: make(map[string]bool),
		permissionRules: []*PermissionRule{
			{Permission: "read", Pattern: "*", Action: "allow"},
			{Permission: "bash", Pattern: "git *", Action: "allow"},
			{Permission: "bash", Pattern: "ls *", Action: "allow"},
			{Permission: "bash", Pattern: "cat *", Action: "allow"},
			{Permission: "edit", Pattern: "*", Action: "ask"},
			{Permission: "write", Pattern: "*", Action: "ask"},
			{Permission: "bash", Pattern: "*", Action: "ask"},
		},
		maxInputTokens:  128000, // Default for GPT-4o
		maxOutputTokens: 4096,
		tokenUsage:      &TokenUsage{},
	}
}

// SetProvider sets the LLM provider
func (a *Agent) SetProvider(name string) error {
	pc, ok := a.config.Providers[name]
	if !ok {
		return fmt.Errorf("unknown provider: %s", name)
	}

	providerCfg := providers.ProviderConfig{
		APIKey:        pc.APIKey,
		BaseURL:       pc.BaseURL,
		Model:         pc.Model,
		Streaming:     pc.Streaming,
		ContextWindow: pc.ContextWindow,
		Temperature:   pc.Temperature,
		MaxTokens:     pc.MaxTokens,
	}

	p, err := providers.NewProvider(name, providerCfg)
	if err != nil {
		return fmt.Errorf("creating provider: %w", err)
	}

	a.provider = p
	a.config.Provider = name
	return nil
}

// SetStreamCallback sets streaming callback
func (a *Agent) SetStreamCallback(fn func(string)) {
	a.streamCallback = fn
}

// SetCompactCallback sets compaction callback
func (a *Agent) SetCompactCallback(fn func(string)) {
	a.compactCallback = fn
}

// Run is the main agent loop - based on opencode's runLoop pattern
func (a *Agent) Run(ctx context.Context, request string) (string, error) {
	if a.provider == nil {
		if err := a.SetProvider(a.config.Provider); err != nil {
			return "", err
		}
	}

	// Load memory context
	a.loadMemoryContext(request)

	// Add user message
	a.messages = append(a.messages, providers.Message{
		Role:    "user",
		Content: request,
	})

	var finalResponse string

	// THE MAIN LOOP - while(true) pattern from opencode
	for {
		// Exit condition 1: Check max turns
		if a.currentTurn >= a.maxTurns {
			return finalResponse, fmt.Errorf("reached maximum turns (%d)", a.maxTurns)
		}
		a.currentTurn++

		// Exit condition 2: Check token budget
		if a.tokenUsage.TotalTokens >= a.maxInputTokens-a.maxOutputTokens {
			if a.compactCallback != nil {
				a.compactCallback("Context approaching limit, triggering compaction...")
			}
			if err := a.compact(); err != nil {
				return finalResponse, fmt.Errorf("compaction failed: %w", err)
			}
		}

		// Find last user message and assistant boundaries
		lastUserIdx := -1
		lastAssistantIdx := -1
		for i := len(a.messages) - 1; i >= 0; i-- {
			if a.messages[i].Role == "user" && lastUserIdx == -1 {
				lastUserIdx = i
			}
			if a.messages[i].Role == "assistant" && lastAssistantIdx == -1 {
				lastAssistantIdx = i
			}
			if lastUserIdx >= 0 && lastAssistantIdx >= 0 {
				break
			}
		}

		// Exit condition 3: Assistant finished with no tool calls pending
		if lastAssistantIdx > lastUserIdx {
			hasPendingTools := a.hasPendingToolCalls()
			if !hasPendingTools {
				// Model returned text only, no tool calls - we're done
				finalResponse = a.messages[lastAssistantIdx].Content
				break
			}
		}

		// Build tool definitions
		toolDefs := a.buildToolDefinitions()

		// Build system prompt
		systemPrompt := a.buildSystemPrompt()

		// Add system message
		allMessages := append([]providers.Message{{Role: "system", Content: systemPrompt}}, a.messages...)

		// Call LLM
		resp, err := a.provider.Chat(ctx, allMessages, toolDefs)
		if err != nil {
			return finalResponse, fmt.Errorf("LLM chat: %w", err)
		}

		// Update token usage
		if resp.Usage != nil {
			a.tokenUsage.PromptTokens += resp.Usage.PromptTokens
			a.tokenUsage.CompletionTokens += resp.Usage.CompletionTokens
			a.tokenUsage.TotalTokens += resp.Usage.TotalTokens
		}

		// Stream content
		if a.streamCallback != nil && resp.Content != "" {
			a.streamCallback(resp.Content)
		}

		// Add assistant message
		a.messages = append(a.messages, providers.Message{
			Role:    "assistant",
			Content: resp.Content,
		})

		// Check if there are tool calls
		if len(resp.ToolCalls) > 0 {
			// Execute tools (potentially in parallel like opencode)
			var wg sync.WaitGroup
			results := make([]struct{
				ID      string
				Name    string
				Result  string
				IsError bool
			}, len(resp.ToolCalls))

			for i, tc := range resp.ToolCalls {
				wg.Add(1)
				go func(idx int, toolCall providers.ToolCall) {
					defer wg.Done()
					
					// Check permissions
					if err := a.checkPermission(toolCall.Name); err != nil {
						results[idx] = struct{
							ID      string
							Name    string
							Result  string
							IsError bool
						}{
							ID:      toolCall.ID,
							Name:    toolCall.Name,
							Result:  fmt.Sprintf("Permission denied: %v", err),
							IsError: true,
						}
						return
					}

					results[idx] = a.executeTool(ctx, toolCall)
				}(i, tc)
			}

			wg.Wait()

			// Add tool results
			for _, r := range results {
				resultContent := r.Result
				if r.IsError {
					resultContent = fmt.Sprintf("Error: %s", r.Result)
				}
				a.messages = append(a.messages, providers.Message{
					Role:    "user",
					Content: fmt.Sprintf("[Tool %s result]: %s", r.Name, resultContent),
				})
			}

			// Detect doom loop (same tool called 3x with same args)
			if a.detectDoomLoop(resp.ToolCalls) {
				return finalResponse, fmt.Errorf("doom loop detected: same tool called 3+ times with same arguments")
			}

			// Continue loop for next turn
			finalResponse = resp.Content
			continue
		}

		// No tool calls, we're done
		finalResponse = resp.Content
		break
	}

	// Save memory after completion
	a.saveMemory(request, finalResponse)

	return finalResponse, nil
}

// hasPendingToolCalls checks if there are unresolved tool calls
func (a *Agent) hasPendingToolCalls() bool {
	// Count tool_use vs tool_result messages
	toolUseCount := 0
	toolResultCount := 0
	
	for _, msg := range a.messages {
		if strings.Contains(msg.Content, "[Tool") && strings.Contains(msg.Content, "result]") {
			toolResultCount++
		}
	}
	
	// Simplified check - in real implementation would track tool call IDs
	return toolUseCount > toolResultCount
}

// buildToolDefinitions creates tool definitions for the model
func (a *Agent) buildToolDefinitions() []providers.ToolDefinition {
	defs := make([]providers.ToolDefinition, len(a.toolList))
	for i, t := range a.toolList {
		defs[i] = providers.ToolDefinition{
			Name:        t.Name(),
			Description: t.Description(),
			Parameters:  t.Parameters(),
		}
	}
	return defs
}

// buildSystemPrompt assembles the system prompt
func (a *Agent) buildSystemPrompt() string {
	var parts []string

	parts = append(parts, `You are ApexCode, an expert AI coding assistant.
You help users write, understand, debug, and modify code.

## Capabilities
- You can read and write files, execute bash commands, search code with grep/glob
- You should prefer bash for multi-step operations over chaining file tools
- Always read files before editing to ensure accurate string matching

## Rules
- Be concise and direct. Avoid filler phrases.
- When making changes, explain what you did and why.
- If unsure about something, say so clearly.
- Never fabricate file contents or command outputs.
- Always verify your work when possible.`)

	// Add repo map if available
	// (Would be injected from repomap generator)

	// Add chat files context
	if len(a.chatFiles) > 0 {
		parts = append(parts, "\n## Files in Chat\n")
		for f := range a.chatFiles {
			parts = append(parts, fmt.Sprintf("- %s", f))
		}
	}

	return strings.Join(parts, "\n")
}

// loadMemoryContext loads context from MemPalace
func (a *Agent) loadMemoryContext(request string) {
	if a.memory == nil {
		return
	}

	a.memory.SetCurrentWing("code")
	
	// L1 retrieval: top 15 important drawers
	context, err := a.memory.Retrieve(request, 1, "", "")
	if err == nil && len(context) > 0 {
		a.messages = append(a.messages, providers.Message{
			Role:    "system",
			Content: "## Relevant Context from Memory\n" + strings.Join(context, "\n"),
		})
	}
}

// saveMemory saves important information after completion
func (a *Agent) saveMemory(request, response string) {
	if a.memory == nil {
		return
	}

	// Save request/response as a drawer
	_, _ = a.memory.StoreContent(
		fmt.Sprintf("Request: %s\n\nResponse: %s", request, response),
		"sessions",
		"interactions",
		"chat",
		"user",
	)
}

// compact performs context window compaction
// Based on opencode's compaction system
func (a *Agent) compact() error {
	if len(a.messages) < 10 {
		return nil // Too few messages to compact
	}

	// Find compaction boundary
	boundary := a.compactBoundary
	if boundary == 0 {
		// Keep last 20% of messages
		boundary = len(a.messages) * 80 / 100
	}

	// Create compaction summary
	summaryMsg := providers.Message{
		Role: "system",
		Content: fmt.Sprintf("[Compaction: summarized %d previous messages into this summary]", boundary),
	}

	// Replace old messages with summary
	a.messages = append([]providers.Message{summaryMsg}, a.messages[boundary:]...)
	a.compactBoundary = 1 // Mark that we compacted

	return nil
}

// executeTool executes a single tool call
func (a *Agent) executeTool(ctx context.Context, tc providers.ToolCall) struct {
	ID      string
	Name    string
	Result  string
	IsError bool
} {
	tool, exists := a.toolMap[tc.Name]
	if !exists {
		return struct{ID, Name, Result string; IsError bool}{tc.ID, tc.Name, fmt.Sprintf("Unknown tool: %s", tc.Name), true}
	}

	var args map[string]interface{}
	if err := json.Unmarshal([]byte(tc.Arguments), &args); err != nil {
		return struct{ID, Name, Result string; IsError bool}{tc.ID, tc.Name, fmt.Sprintf("Invalid arguments: %v", err), true}
	}

	result, err := tool.Execute(ctx, args)
	if err != nil {
		return struct{ID, Name, Result string; IsError bool}{tc.ID, tc.Name, err.Error(), true}
	}
	return struct{ID, Name, Result string; IsError bool}{tc.ID, tc.Name, result, false}
}

// detectDoomLoop checks if the same tool was called 3+ times with same arguments
// From opencode's processor.ts doom loop detection
func (a *Agent) detectDoomLoop(calls []providers.ToolCall) bool {
	if len(calls) < 3 {
		return false
	}

	// Check last 3 calls
	for i := 0; i < len(calls)-2; i++ {
		if calls[i].Name == calls[i+1].Name && calls[i].Name == calls[i+2].Name &&
			calls[i].Arguments == calls[i+1].Arguments && calls[i].Arguments == calls[i+2].Arguments {
			return true
		}
	}

	return false
}

// checkPermission checks if a tool execution is allowed
func (a *Agent) checkPermission(toolName string) error {
	for _, rule := range a.permissionRules {
		if rule.Permission == toolName || rule.Pattern == "*" {
			switch rule.Action {
			case "deny":
				return fmt.Errorf("tool %s is denied by permission rules", toolName)
			case "ask":
				// In real implementation, this would prompt user
				// For now, allow with warning
				fmt.Printf("⚠️  Permission required for tool: %s (auto-approved)\n", toolName)
				return nil
			case "allow":
				return nil
			}
		}
	}
	return nil
}

// AddPermissionRule adds a permission rule
func (a *Agent) AddPermissionRule(permission, pattern, action string) {
	a.permissionRules = append(a.permissionRules, &PermissionRule{
		Permission: permission,
		Pattern:    pattern,
		Action:     action,
	})
}

// GetMessages returns conversation history
func (a *Agent) GetMessages() []providers.Message {
	return a.messages
}

// ClearMessages clears conversation
func (a *Agent) ClearMessages() {
	a.messages = make([]providers.Message, 0)
	a.currentTurn = 0
	a.tokenUsage = &TokenUsage{}
}

// GetStats returns statistics
func (a *Agent) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"provider":       a.config.Provider,
		"model":          a.provider.Model(),
		"turn_count":     a.currentTurn,
		"message_count":  len(a.messages),
		"max_turns":      a.maxTurns,
		"token_usage":    a.tokenUsage,
		"chat_files":     len(a.chatFiles),
		"readonly_files": len(a.readonlyFiles),
	}
}
