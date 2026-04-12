package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Config represents the main configuration structure
type Config struct {
	// Active provider to use
	Provider string `json:"provider"`
	
	// Provider configurations
	Providers map[string]ProviderConfig `json:"providers"`
	
	// Default model to use
	DefaultModel string `json:"default_model,omitempty"`
	
	// Maximum agent turns (0 = unlimited)
	MaxTurns int `json:"max_turns,omitempty"`
	
	// Enable parallel tool calls
	ParallelTools bool `json:"parallel_tools,omitempty"`
	
	// Enable git safety mode
	GitSafety bool `json:"git_safety,omitempty"`
	
	// Enable auto-commit
	AutoCommit bool `json:"auto_commit,omitempty"`
	
	// Theme for TUI
	Theme string `json:"theme,omitempty"`
	
	// Working directory
	WorkDir string `json:"-"`
	
	// Path to APEX.md
	ApexMDPath string `json:"-"`
}

// ProviderConfig holds configuration for a specific LLM provider
type ProviderConfig struct {
	// API key for the provider
	APIKey string `json:"api_key,omitempty"`
	
	// Base URL (for local models like LM Studio, Ollama)
	BaseURL string `json:"base_url,omitempty"`
	
	// Model name to use
	Model string `json:"model,omitempty"`
	
	// Enable streaming
	Streaming bool `json:"streaming,omitempty"`
	
	// Context window size
	ContextWindow int `json:"context_window,omitempty"`
	
	// Temperature
	Temperature float64 `json:"temperature,omitempty"`
	
	// Max tokens
	MaxTokens int `json:"max_tokens,omitempty"`
}

// DefaultConfig returns a configuration with sensible defaults
func DefaultConfig() *Config {
	return &Config{
		Provider:      "openai",
		DefaultModel:  "gpt-4o",
		MaxTurns:      100,
		ParallelTools: true,
		GitSafety:     true,
		AutoCommit:    false,
		Theme:         "default",
		Providers: map[string]ProviderConfig{
			"openai": {
				Model:      "gpt-4o",
				Streaming:  true,
				Temperature: 0.7,
				MaxTokens:   4096,
			},
			"anthropic": {
				Model:      "claude-sonnet-4-20250514",
				Streaming:  true,
				Temperature: 0.7,
				MaxTokens:   4096,
			},
			"google": {
				Model:      "gemini-2.5-pro",
				Streaming:  true,
				Temperature: 0.7,
			},
			"lmstudio": {
				BaseURL:    "http://localhost:1234/v1",
				Model:      "local-model",
				Streaming:  true,
				Temperature: 0.7,
			},
			"ollama": {
				BaseURL:    "http://localhost:11434/v1",
				Model:      "llama3",
				Streaming:  true,
			},
		},
	}
}

// Load reads configuration from file and environment
func Load() (*Config, error) {
	configPath := getConfigPath()
	
	// Start with defaults
	cfg := DefaultConfig()
	
	// Load from file if exists
	if data, err := os.ReadFile(configPath); err == nil {
		if err := json.Unmarshal(data, cfg); err != nil {
			return nil, fmt.Errorf("parsing config file: %w", err)
		}
	}
	
	// Override with environment variables
	if apiKey := os.Getenv("OPENAI_API_KEY"); apiKey != "" {
		p := cfg.Providers["openai"]
		p.APIKey = apiKey
		cfg.Providers["openai"] = p
		cfg.Provider = "openai"
	}
	
	if apiKey := os.Getenv("ANTHROPIC_API_KEY"); apiKey != "" {
		p := cfg.Providers["anthropic"]
		p.APIKey = apiKey
		cfg.Providers["anthropic"] = p
		cfg.Provider = "anthropic"
	}
	
	if apiKey := os.Getenv("GOOGLE_API_KEY"); apiKey != "" {
		p := cfg.Providers["google"]
		p.APIKey = apiKey
		cfg.Providers["google"] = p
		cfg.Provider = "google"
	}
	
	if apiKey := os.Getenv("GROQ_API_KEY"); apiKey != "" {
		cfg.Providers["groq"] = ProviderConfig{
			APIKey:     apiKey,
			Model:      "llama-3.1-70b",
			Streaming:  true,
		}
	}
	
	// Set working directory
	wd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("getting working directory: %w", err)
	}
	cfg.WorkDir = wd
	
	// Find APEX.md
	cfg.ApexMDPath = findApexMD(wd)
	
	return cfg, nil
}

// Save writes configuration to file
func (c *Config) Save() error {
	configPath := getConfigPath()
	
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}
	
	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}
	
	return os.WriteFile(configPath, data, 0644)
}

// GetProviderConfig returns the configuration for the active provider
func (c *Config) GetProviderConfig() ProviderConfig {
	return c.Providers[c.Provider]
}

// getConfigPath returns the path to the config file
func getConfigPath() string {
	configDir := os.Getenv("APEX_CONFIG_DIR")
	if configDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			home = "."
		}
		configDir = filepath.Join(home, ".config", "apexcode")
	}
	return filepath.Join(configDir, "config.json")
}

// findApexMD searches for APEX.md in the directory tree
func findApexMD(dir string) string {
	for {
		path := filepath.Join(dir, "APEX.md")
		if _, err := os.Stat(path); err == nil {
			return path
		}
		
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return ""
}

// InitProject creates an APEX.md file in the current directory
func InitProject() error {
	apexPath := filepath.Join(".", "APEX.md")
	
	if _, err := os.Stat(apexPath); err == nil {
		return fmt.Errorf("APEX.md already exists")
	}
	
	content := `# APEX.md - Project Context

## Project Overview
<!-- Describe your project here -->

## Architecture
<!-- Key architectural decisions -->

## Coding Standards
<!-- Language-specific conventions and style guides -->

## Important Notes
<!-- Anything the AI should know -->
`
	return os.WriteFile(apexPath, []byte(content), 0644)
}
