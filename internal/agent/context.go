package agent

import (
	"fmt"
	"strings"
)

// ChatChunks represents the structured message assembly
// Based on aider's ChatChunks: system + examples + readonly_files + repo + done + chat_files + cur + reminder
type ChatChunks struct {
	System      []string // System prompt parts
	Examples    []string // Few-shot examples
	ReadOnlyFiles []string // Read-only file contents
	RepoMap     string   // Repository map
	Done        []string // Summarized old messages
	ChatFiles   []string // Files currently in chat
	Current     []string // Current conversation
	Reminder    string   // System reminder
}

// Build assembles the message structure
func (c *ChatChunks) Build() []MessagePart {
	var parts []MessagePart

	if len(c.System) > 0 {
		parts = append(parts, MessagePart{
			Role:    "system",
			Content: strings.Join(c.System, "\n"),
		})
	}

	if len(c.Examples) > 0 {
		parts = append(parts, MessagePart{
			Role:    "user",
			Content: "## Examples\n" + strings.Join(c.Examples, "\n"),
		})
	}

	if len(c.ReadOnlyFiles) > 0 {
		parts = append(parts, MessagePart{
			Role:    "user",
			Content: "## Read-Only Files\n" + strings.Join(c.ReadOnlyFiles, "\n"),
		})
	}

	if c.RepoMap != "" {
		parts = append(parts, MessagePart{
			Role:    "user",
			Content: "## Repository Map\n" + c.RepoMap,
		})
	}

	if len(c.Done) > 0 {
		parts = append(parts, MessagePart{
			Role:    "system",
			Content: "## Previous Conversation (Summarized)\n" + strings.Join(c.Done, "\n"),
		})
	}

	if len(c.ChatFiles) > 0 {
		parts = append(parts, MessagePart{
			Role:    "user",
			Content: "## Files in Chat\n" + strings.Join(c.ChatFiles, "\n"),
		})
	}

	if len(c.Current) > 0 {
		for _, content := range c.Current {
			parts = append(parts, MessagePart{
				Role:    "user",
				Content: content,
			})
		}
	}

	if c.Reminder != "" {
		parts = append(parts, MessagePart{
			Role:    "system",
			Content: c.Reminder,
		})
	}

	return parts
}

// MessagePart represents a part of the message
type MessagePart struct {
	Role    string
	Content string
}

// HistorySummarizer handles chat history summarization
// Based on aider's history.py
type HistorySummarizer struct {
	maxTokens     int
	summarizeFunc func(messages []string) (string, error)
}

// NewHistorySummarizer creates a new summarizer
func NewHistorySummarizer(maxTokens int, summarizeFunc func(messages []string) (string, error)) *HistorySummarizer {
	return &HistorySummarizer{
		maxTokens:     maxTokens,
		summarizeFunc: summarizeFunc,
	}
}

// Summarize summarizes old messages to fit within token budget
func (h *HistorySummarizer) Summarize(messages []string) ([]string, error) {
	if len(messages) == 0 {
		return messages, nil
	}

	// Estimate token count (rough: 1.3 tokens per word)
	totalTokens := h.estimateTokens(messages)
	if totalTokens <= h.maxTokens {
		return messages, nil
	}

	// Split into head (summarizable) and tail (recent, keep intact)
	// Ensure split point ends with assistant message
	splitIdx := h.findSplitPoint(messages)
	if splitIdx <= 0 {
		return messages, nil
	}

	head := messages[:splitIdx]
	tail := messages[splitIdx:]

	// Summarize head
	summary, err := h.summarizeFunc(head)
	if err != nil {
		return messages, err
	}

	// Check if combined result still too big
	result := append([]string{summary}, tail...)
	resultTokens := h.estimateTokens(result)
	if resultTokens > h.maxTokens {
		// Recursively summarize
		return h.Summarize(result)
	}

	return result, nil
}

// findSplitPoint finds a good place to split messages (after an assistant message)
func (h *HistorySummarizer) findSplitPoint(messages []string) int {
	// Go backwards from middle
	mid := len(messages) / 2
	for i := mid; i > 0; i-- {
		// In real implementation, would check message roles
		// For now, just split at a reasonable point
		if h.estimateTokens(messages[:i]) < h.maxTokens/2 {
			return i
		}
	}
	return 1
}

// estimateTokens estimates token count for messages
func (h *HistorySummarizer) estimateTokens(messages []string) int {
	total := 0
	for _, msg := range messages {
		// Rough estimate: 1.3 tokens per word, plus overhead
		words := len(strings.Fields(msg))
		total += int(float64(words) * 1.3)
	}
	return total
}

// PromptCacher handles prompt caching for API efficiency
// Based on aider's add_cache_control_headers
type PromptCacher struct {
	lastCacheTime int64 // Unix timestamp of last cache refresh
}

// NewPromptCacher creates a new prompt cacheer
func NewPromptCacher() *PromptCacher {
	return &PromptCacher{}
}

// AddCacheControl marks messages for caching
// In production, would add cache_control headers for Anthropic API
func (p *PromptCacher) AddCacheControl(messages []string) []string {
	// Mark the last few messages for caching
	// This is handled by the API client in production
	return messages
}

// CheckWarm checks if cache needs warming
func (p *PromptCacher) CheckWarm() bool {
	// In production, would ping API every 5 minutes to keep cache warm
	return false
}

// TokenBudget tracks token usage and provides warnings
type TokenBudget struct {
	MaxInput  int
	MaxOutput int
	Used      int
	Exhausted bool
}

// NewTokenBudget creates a new token budget tracker
func NewTokenBudget(maxInput, maxOutput int) *TokenBudget {
	return &TokenBudget{
		MaxInput:  maxInput,
		MaxOutput: maxOutput,
	}
}

// Check checks if we're approaching the limit
func (t *TokenBudget) Check() error {
	available := t.MaxInput - t.MaxOutput
	if t.Used >= available {
		t.Exhausted = true
		return fmt.Errorf("token budget exhausted: used %d of %d available", t.Used, available)
	}
	
	if t.Used >= int(float64(available)*0.9) {
		return fmt.Errorf("approaching token budget limit: %d%% used", int(float64(t.Used)/float64(available)*100))
	}
	
	return nil
}

// GetBreakdown returns a token usage breakdown
func (t *TokenBudget) GetBreakdown() string {
	available := t.MaxInput - t.MaxOutput
	return fmt.Sprintf(
		"Token Usage:\n"+
		"  Used: %d tokens\n"+
		"  Available: %d tokens\n"+
		"  Max input: %d tokens\n"+
		"  Max output: %d tokens\n"+
		"  Remaining: %d tokens",
		t.Used, available, t.MaxInput, t.MaxOutput, available-t.Used,
	)
}
