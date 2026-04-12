package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// GoogleProvider implements the Provider interface for Google Gemini
type GoogleProvider struct {
	model       string
	apiKey      string
	temperature float64
}

// NewGoogleProvider creates a new Google provider
func NewGoogleProvider(config ProviderConfig) (*GoogleProvider, error) {
	if config.APIKey == "" {
		return nil, fmt.Errorf("Google provider requires api_key")
	}

	model := config.Model
	if model == "" {
		model = "gemini-2.5-pro"
	}

	return &GoogleProvider{
		model:       model,
		apiKey:      config.APIKey,
		temperature: config.Temperature,
	}, nil
}

// Chat sends a chat completion request
func (p *GoogleProvider) Chat(ctx context.Context, messages []Message, tools []ToolDefinition) (*ChatResponse, error) {
	// Build contents string from messages
	var contents strings.Builder
	for _, m := range messages {
		contents.WriteString(fmt.Sprintf("[%s]: %s\n", m.Role, m.Content))
	}

	// Build tool definitions
	var toolsJSON []map[string]interface{}
	for _, t := range tools {
		paramsJSON := []byte("{}")
		if t.Parameters != nil {
			paramsJSON, _ = json.Marshal(t.Parameters)
		}
		
		var params map[string]interface{}
		json.Unmarshal(paramsJSON, &params)
		
		toolsJSON = append(toolsJSON, map[string]interface{}{
			"name":        t.Name,
			"description": t.Description,
			"parameters":  params,
		})
	}

	// Note: Full Gemini API integration would use the official SDK
	// This is a placeholder for the actual API call
	return nil, fmt.Errorf("Google Gemini provider requires additional setup - please see documentation")
}

// ChatStream starts a streaming response
func (p *GoogleProvider) ChatStream(ctx context.Context, messages []Message, tools []ToolDefinition) (Stream, error) {
	return nil, fmt.Errorf("streaming not yet implemented for Google provider")
}

// Name returns the provider name
func (p *GoogleProvider) Name() string {
	return "google"
}

// Model returns the model being used
func (p *GoogleProvider) Model() string {
	return p.model
}
