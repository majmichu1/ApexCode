package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// AnthropicProvider implements the Provider interface for Anthropic Claude
// Uses direct HTTP API instead of external package
type AnthropicProvider struct {
	apiKey      string
	model       string
	temperature float64
	maxTokens   int
	baseURL     string
	client      *http.Client
}

// NewAnthropicProvider creates a new Anthropic provider
func NewAnthropicProvider(config ProviderConfig) (*AnthropicProvider, error) {
	if config.APIKey == "" {
		return nil, fmt.Errorf("Anthropic provider requires api_key")
	}

	model := config.Model
	if model == "" {
		model = "claude-sonnet-4-20250514"
	}

	return &AnthropicProvider{
		apiKey:      config.APIKey,
		model:       model,
		temperature: config.Temperature,
		maxTokens:   config.MaxTokens,
		baseURL:     "https://api.anthropic.com/v1",
		client:      &http.Client{},
	}, nil
}

// Chat sends a chat completion request
func (p *AnthropicProvider) Chat(ctx context.Context, messages []Message, tools []ToolDefinition) (*ChatResponse, error) {
	// Build request
	reqBody := map[string]interface{}{
		"model":       p.model,
		"max_tokens":  p.maxTokens,
		"temperature": p.temperature,
	}

	// Convert messages (Anthropic uses different format)
	var anthropicMessages []map[string]interface{}
	var systemPrompt string

	for _, m := range messages {
		switch m.Role {
		case "system":
			systemPrompt = m.Content
		case "user", "assistant":
			anthropicMessages = append(anthropicMessages, map[string]interface{}{
				"role":    m.Role,
				"content": m.Content,
			})
		}
	}

	if systemPrompt != "" {
		reqBody["system"] = systemPrompt
	}
	reqBody["messages"] = anthropicMessages

	// Add tools
	if len(tools) > 0 {
		var anthropicTools []map[string]interface{}
		for _, t := range tools {
			tool := map[string]interface{}{
				"name":        t.Name,
				"description": t.Description,
				"input_schema": t.Parameters,
			}
			anthropicTools = append(anthropicTools, tool)
		}
		reqBody["tools"] = anthropicTools
	}

	reqJSON, _ := json.Marshal(reqBody)

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/messages", bytes.NewReader(reqJSON))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", p.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	// Send request
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("sending request: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var result struct {
		ID           string `json:"id"`
		Type         string `json:"type"`
		Role         string `json:"role"`
		Content      []struct {
			Type       string          `json:"type"`
			Text       string          `json:"text"`
			ID         string          `json:"id"`
			Name       string          `json:"name"`
			Input      json.RawMessage `json:"input"`
		} `json:"content"`
		Model         string `json:"model"`
		StopReason    string `json:"stop_reason"`
		StopSequence  string `json:"stop_sequence"`
		Usage         struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
		} `json:"usage"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}

	chatResp := &ChatResponse{
		FinishReason: result.StopReason,
		Usage: &Usage{
			PromptTokens:     result.Usage.InputTokens,
			CompletionTokens: result.Usage.OutputTokens,
			TotalTokens:      result.Usage.InputTokens + result.Usage.OutputTokens,
		},
	}

	// Extract content and tool calls
	for _, block := range result.Content {
		switch block.Type {
		case "text":
			chatResp.Content += block.Text
		case "tool_use":
			chatResp.ToolCalls = append(chatResp.ToolCalls, ToolCall{
				ID:        block.ID,
				Name:      block.Name,
				Arguments: string(block.Input),
			})
		}
	}

	return chatResp, nil
}

// ChatStream starts a streaming response
func (p *AnthropicProvider) ChatStream(ctx context.Context, messages []Message, tools []ToolDefinition) (Stream, error) {
	return nil, fmt.Errorf("streaming not yet implemented for Anthropic provider")
}

// Name returns the provider name
func (p *AnthropicProvider) Name() string {
	return "anthropic"
}

// Model returns the model being used
func (p *AnthropicProvider) Model() string {
	return p.model
}
