package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	openai "github.com/sashabaranov/go-openai"
)

// OpenAIProvider implements the Provider interface for OpenAI-compatible APIs
type OpenAIProvider struct {
	client    *openai.Client
	model     string
	baseURL   string
	temperature float64
	maxTokens int
}

// NewOpenAIProvider creates a new OpenAI provider
func NewOpenAIProvider(config ProviderConfig) (*OpenAIProvider, error) {
	if config.APIKey == "" && config.BaseURL == "" {
		return nil, fmt.Errorf("OpenAI provider requires api_key or base_url")
	}

	c := openai.DefaultConfig(config.APIKey)
	if config.BaseURL != "" {
		c.BaseURL = config.BaseURL
	}

	client := openai.NewClientWithConfig(c)

	return &OpenAIProvider{
		client:    client,
		model:     config.Model,
		baseURL:   config.BaseURL,
		temperature: config.Temperature,
		maxTokens: config.MaxTokens,
	}, nil
}

// NewLMStudioProvider creates a provider for LM Studio (OpenAI-compatible)
func NewLMStudioProvider(config ProviderConfig) (*OpenAIProvider, error) {
	if config.BaseURL == "" {
		config.BaseURL = "http://localhost:1234/v1"
	}
	return NewOpenAIProvider(config)
}

// NewOllamaProvider creates a provider for Ollama (OpenAI-compatible)
func NewOllamaProvider(config ProviderConfig) (*OpenAIProvider, error) {
	if config.BaseURL == "" {
		config.BaseURL = "http://localhost:11434/v1"
	}
	if config.Model == "" {
		config.Model = "llama3"
	}
	return NewOpenAIProvider(config)
}

// NewGroqProvider creates a provider for Groq (OpenAI-compatible)
func NewGroqProvider(config ProviderConfig) (*OpenAIProvider, error) {
	c := openai.DefaultConfig(config.APIKey)
	c.BaseURL = "https://api.groq.com/openai/v1"
	
	client := openai.NewClientWithConfig(c)

	return &OpenAIProvider{
		client:    client,
		model:     config.Model,
		baseURL:   c.BaseURL,
		temperature: config.Temperature,
		maxTokens: config.MaxTokens,
	}, nil
}

// Chat sends a chat completion request
func (p *OpenAIProvider) Chat(ctx context.Context, messages []Message, tools []ToolDefinition) (*ChatResponse, error) {
	// Convert messages
	var openaiMessages []openai.ChatCompletionMessage
	for _, m := range messages {
		openaiMessages = append(openaiMessages, openai.ChatCompletionMessage{
			Role:    m.Role,
			Content: m.Content,
		})
	}

	// Convert tools
	var openaiTools []openai.Tool
	for _, t := range tools {
		paramsJSON := []byte("{}")
		if t.Parameters != nil {
			paramsJSON, _ = json.Marshal(t.Parameters)
		}

		var params map[string]interface{}
		json.Unmarshal(paramsJSON, &params)

		openaiTools = append(openaiTools, openai.Tool{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        t.Name,
				Description: t.Description,
				Parameters:  params,
			},
		})
	}

	req := openai.ChatCompletionRequest{
		Model:       p.model,
		Messages:    openaiMessages,
		Tools:       openaiTools,
		Temperature: float32(p.temperature),
		MaxTokens:   p.maxTokens,
	}

	resp, err := p.client.CreateChatCompletion(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("OpenAI chat completion: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("no choices returned")
	}

	choice := resp.Choices[0]
	result := &ChatResponse{
		Content:      choice.Message.Content,
		FinishReason: string(choice.FinishReason),
	}

	// Extract tool calls
	for _, tc := range choice.Message.ToolCalls {
		result.ToolCalls = append(result.ToolCalls, ToolCall{
			ID:        tc.ID,
			Name:      tc.Function.Name,
			Arguments: tc.Function.Arguments,
		})
	}

	// Usage
	if resp.Usage.TotalTokens > 0 {
		result.Usage = &Usage{
			PromptTokens:     resp.Usage.PromptTokens,
			CompletionTokens: resp.Usage.CompletionTokens,
			TotalTokens:      resp.Usage.TotalTokens,
		}
	}

	return result, nil
}

// ChatStream starts a streaming response
func (p *OpenAIProvider) ChatStream(ctx context.Context, messages []Message, tools []ToolDefinition) (Stream, error) {
	var openaiMessages []openai.ChatCompletionMessage
	for _, m := range messages {
		openaiMessages = append(openaiMessages, openai.ChatCompletionMessage{
			Role:    m.Role,
			Content: m.Content,
		})
	}

	req := openai.ChatCompletionRequest{
		Model:       p.model,
		Messages:    openaiMessages,
		Stream:      true,
		Temperature: float32(p.temperature),
		MaxTokens:   p.maxTokens,
	}

	stream, err := p.client.CreateChatCompletionStream(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("OpenAI chat stream: %w", err)
	}

	return &OpenAIStream{stream: stream}, nil
}

// Name returns the provider name
func (p *OpenAIProvider) Name() string {
	if p.baseURL != "" {
		return "openai-compatible"
	}
	return "openai"
}

// Model returns the model being used
func (p *OpenAIProvider) Model() string {
	return p.model
}

// OpenAIStream implements Stream for OpenAI responses
type OpenAIStream struct {
	stream *openai.ChatCompletionStream
}

func (s *OpenAIStream) Recv() (string, error) {
	resp, err := s.stream.Recv()
	if err != nil {
		if err == io.EOF {
			return "", io.EOF
		}
		return "", fmt.Errorf("stream recv: %w", err)
	}

	if len(resp.Choices) > 0 {
		return resp.Choices[0].Delta.Content, nil
	}
	return "", nil
}

func (s *OpenAIStream) Close() error {
	s.stream.Close()
	return nil
}
