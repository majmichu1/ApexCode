package providers

import (
	"context"
	"fmt"
)

// Message represents a chat message
type Message struct {
	Role    string `json:"role"`    // "system", "user", "assistant"
	Content string `json:"content"`
}

// ToolCall represents a tool call from the model
type ToolCall struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Arguments string `json:"arguments"` // JSON string
}

// ToolResult represents the result of a tool execution
type ToolResult struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Result  string `json:"result"`
	IsError bool   `json:"is_error"`
}

// ChatResponse represents a response from the LLM
type ChatResponse struct {
	// Content from the model
	Content string
	
	// Tool calls requested by the model
	ToolCalls []ToolCall
	
	// Whether the response is complete
	FinishReason string
	
	// Usage statistics
	Usage *Usage
}

// Usage represents token usage
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// Provider defines the interface for LLM providers
type Provider interface {
	// Chat sends a chat completion request and returns the response
	Chat(ctx context.Context, messages []Message, tools []ToolDefinition) (*ChatResponse, error)
	
	// ChatStream starts a streaming chat response
	ChatStream(ctx context.Context, messages []Message, tools []ToolDefinition) (Stream, error)
	
	// Name returns the provider name
	Name() string
	
	// Model returns the model being used
	Model() string
}

// Stream represents a streaming response
type Stream interface {
	// Recv returns the next chunk of content
	Recv() (string, error)
	
	// Close closes the stream
	Close() error
}

// ToolDefinition defines a tool available to the model
type ToolDefinition struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Parameters  interface{} `json:"parameters"` // JSON Schema
}

// NewProvider creates a new provider instance based on configuration
func NewProvider(providerType string, config ProviderConfig) (Provider, error) {
	switch providerType {
	case "openai":
		return NewOpenAIProvider(config)
	case "anthropic":
		return NewAnthropicProvider(config)
	case "google":
		return NewGoogleProvider(config)
	case "lmstudio":
		return NewLMStudioProvider(config)
	case "ollama":
		return NewOllamaProvider(config)
	case "groq":
		return NewGroqProvider(config)
	default:
		return nil, fmt.Errorf("unknown provider: %s", providerType)
	}
}

// ProviderConfig holds the configuration for creating a provider
type ProviderConfig struct {
	APIKey        string
	BaseURL       string
	Model         string
	Streaming     bool
	ContextWindow int
	Temperature   float64
	MaxTokens     int
}
