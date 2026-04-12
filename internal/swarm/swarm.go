package swarm

import (
	"context"
	"fmt"
	"sync"

	"github.com/apexcode/apexcode/internal/agent"
	"github.com/apexcode/apexcode/internal/config"
)

// SwarmManager handles multi-agent orchestration
type SwarmManager struct {
	parent    *agent.Agent
	children  []*SwarmAgent
	config    *config.Config
	results   []SwarmResult
	mu        sync.Mutex
}

// SwarmAgent represents a child agent in the swarm
type SwarmAgent struct {
	ID        string
	Agent     *agent.Agent
	Task      string
	Status    string // "pending", "running", "completed", "failed"
	Result    string
	Error     error
	ToolLimit []string // Restricted toolset
}

// SwarmResult holds the result of a swarm task
type SwarmResult struct {
	AgentID string
	Task    string
	Result  string
	Error   error
}

// NewSwarmManager creates a new swarm manager
func NewSwarmManager(cfg *config.Config, parent *agent.Agent) *SwarmManager {
	return &SwarmManager{
		parent: parent,
		config: cfg,
		children: make([]*SwarmAgent, 0),
	}
}

// SpawnAgent creates a new child agent with restricted capabilities
func (sm *SwarmManager) SpawnAgent(task string, allowedTools []string) *SwarmAgent {
	// Create child agent
	child := &SwarmAgent{
		ID:        fmt.Sprintf("agent_%d", len(sm.children)+1),
		Agent:     agent.New(sm.config),
		Task:      task,
		Status:    "pending",
		ToolLimit: allowedTools,
	}

	sm.children = append(sm.children, child)
	return child
}

// ExecuteSwarm runs all spawned agents in parallel
func (sm *SwarmManager) ExecuteSwarm(ctx context.Context) ([]SwarmResult, error) {
	var wg sync.WaitGroup
	sm.results = make([]SwarmResult, 0, len(sm.children))

	for _, child := range sm.children {
		wg.Add(1)
		go func(c *SwarmAgent) {
			defer wg.Done()
			sm.executeAgent(ctx, c)
		}(child)
	}

	wg.Wait()
	return sm.results, nil
}

// executeAgent runs a single child agent
func (sm *SwarmManager) executeAgent(ctx context.Context, agent *SwarmAgent) {
	agent.Status = "running"

	// Set provider
	if err := agent.Agent.SetProvider(sm.config.Provider); err != nil {
		agent.Status = "failed"
		agent.Error = err
		sm.mu.Lock()
		sm.results = append(sm.results, SwarmResult{
			AgentID: agent.ID,
			Task:    agent.Task,
			Error:   err,
		})
		sm.mu.Unlock()
		return
	}

	// Run agent
	result, err := agent.Agent.Run(ctx, agent.Task)
	if err != nil {
		agent.Status = "failed"
		agent.Error = err
		sm.mu.Lock()
		sm.results = append(sm.results, SwarmResult{
			AgentID: agent.ID,
			Task:    agent.Task,
			Error:   err,
		})
		sm.mu.Unlock()
		return
	}

	agent.Status = "completed"
	agent.Result = result

	sm.mu.Lock()
	sm.results = append(sm.results, SwarmResult{
		AgentID: agent.ID,
		Task:    agent.Task,
		Result:  result,
	})
	sm.mu.Unlock()
}

// MergeResults combines results from all child agents
func (sm *SwarmManager) MergeResults() string {
	var merged string
	
	for _, result := range sm.results {
		if result.Error != nil {
			merged += fmt.Sprintf("\n## Agent %s (FAILED)\nError: %v\n", result.AgentID, result.Error)
		} else {
			merged += fmt.Sprintf("\n## Agent %s\n%s\n", result.AgentID, result.Result)
		}
	}

	return merged
}

// GetStatus returns the status of all agents
func (sm *SwarmManager) GetStatus() string {
	var status string
	for _, child := range sm.children {
		emoji := "⏳"
		if child.Status == "completed" {
			emoji = "✅"
		} else if child.Status == "failed" {
			emoji = "❌"
		} else if child.Status == "running" {
			emoji = "🔄"
		}
		status += fmt.Sprintf("%s %s: %s\n", emoji, child.ID, child.Task)
	}
	return status
}

// Clear resets the swarm
func (sm *SwarmManager) Clear() {
	sm.children = make([]*SwarmAgent, 0)
	sm.results = make([]SwarmResult, 0)
}
