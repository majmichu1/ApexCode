package providers

import (
	"context"
	"fmt"
	"strings"
	"sync"
)

// Router handles model selection and failover
type Router struct {
	providers   map[string]Provider
	active      string
	priority    []string // Ordered list of providers for fallback
	mu          sync.RWMutex
}

// TaskComplexity represents the estimated complexity of a task
type TaskComplexity int

const (
	// SimpleTask is for basic operations (use fast/cheap model)
	SimpleTask TaskComplexity = iota
	// MediumTask for moderate complexity
	MediumTask
	// ComplexTask for difficult multi-step tasks (use most capable model)
	ComplexTask
)

// NewRouter creates a new model router
func NewRouter() *Router {
	return &Router{
		providers: make(map[string]Provider),
		priority:  make([]string, 0),
	}
}

// RegisterProvider adds a provider to the router
func (r *Router) RegisterProvider(name string, provider Provider, isFallback bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.providers[name] = provider
	r.priority = append(r.priority, name)

	if isFallback && r.active == "" {
		r.active = name
	}
}

// SetActive sets the active provider
func (r *Router) SetActive(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.providers[name]; !exists {
		return fmt.Errorf("provider %s not registered", name)
	}

	r.active = name
	return nil
}

// AutoSelectModel selects the best provider based on task complexity
func (r *Router) AutoSelectModel(task string) Provider {
	r.mu.RLock()
	defer r.mu.RUnlock()

	complexity := estimateComplexity(task)

	// Select provider based on complexity
	switch complexity {
	case SimpleTask:
		// Use first available (likely cheapest/fastest)
		if len(r.priority) > 0 {
			name := r.priority[0]
			return r.providers[name]
		}
	case MediumTask:
		// Use middle provider
		if len(r.priority) > 1 {
			name := r.priority[1]
			return r.providers[name]
		}
	case ComplexTask:
		// Use most capable provider
		if len(r.priority) > 0 {
			name := r.priority[len(r.priority)-1]
			return r.providers[name]
		}
	}

	// Fallback to active
	if r.active != "" {
		return r.providers[r.active]
	}

	// Last resort: first provider
	if len(r.priority) > 0 {
		return r.providers[r.priority[0]]
	}

	return nil
}

// ChatWithFailover attempts chat with automatic failover
func (r *Router) ChatWithFailover(ctx context.Context, task string, messages []Message, tools []ToolDefinition) (*ChatResponse, error) {
	r.mu.RLock()
	providers := make([]Provider, 0, len(r.providers))
	for _, name := range r.priority {
		providers = append(providers, r.providers[name])
	}
	r.mu.RUnlock()

	if len(providers) == 0 {
		return nil, fmt.Errorf("no providers available")
	}

	// Try each provider in order
	var lastErr error
	for _, provider := range providers {
		resp, err := provider.Chat(ctx, messages, tools)
		if err == nil {
			// Update active provider to successful one
			r.mu.Lock()
			r.active = provider.Name()
			r.mu.Unlock()
			return resp, nil
		}

		lastErr = err
		// Log the failure and try next provider
		fmt.Printf("⚠️  Provider %s failed: %v, trying next...\n", provider.Name(), err)
	}

	return nil, fmt.Errorf("all providers failed. Last error: %w", lastErr)
}

// GetStatus returns router status
func (r *Router) GetStatus() string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var status strings.Builder
	status.WriteString("Model Router Status:\n\n")
	status.WriteString(fmt.Sprintf("Active provider: %s\n\n", r.active))
	status.WriteString("Registered providers:\n")
	
	for i, name := range r.priority {
		marker := " "
		if name == r.active {
			marker = "▶"
		}
		status.WriteString(fmt.Sprintf("  %d. %s %s\n", i+1, marker, name))
	}

	return status.String()
}

// estimateComplexity estimates task complexity from the task description
func estimateComplexity(task string) TaskComplexity {
	taskLower := strings.ToLower(task)

	// Keywords indicating complex tasks
	complexKeywords := []string{
		"refactor", "migrate", "architecture", "design",
		"implement from scratch", "rewrite", "multi-step",
		"integration", "database schema", "api",
	}

	// Keywords indicating medium tasks
	mediumKeywords := []string{
		"add feature", "create", "build", "fix bug",
		"debug", "error", "implement",
	}

	// Check for complex keywords
	for _, kw := range complexKeywords {
		if strings.Contains(taskLower, kw) {
			return ComplexTask
		}
	}

	// Check for medium keywords
	for _, kw := range mediumKeywords {
		if strings.Contains(taskLower, kw) {
			return MediumTask
		}
	}

	// Default to simple
	return SimpleTask
}
