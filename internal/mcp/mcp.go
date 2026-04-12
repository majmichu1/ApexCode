package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
)

// MCPServer represents an MCP server connection
type MCPServer struct {
	Name    string
	URL     string
	Client  *MCPClient
	Tools   []MCPTool
	Connected bool
}

// MCPClient handles MCP JSON-RPC communication
type MCPClient struct {
	serverURL string
	clientID  string
	mu        sync.Mutex
}

// MCPTool represents a tool provided by an MCP server
type MCPTool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Schema      map[string]interface{} `json:"schema"`
	Server      string                 `json:"server"`
}

// MCPManager manages multiple MCP server connections
type MCPManager struct {
	servers map[string]*MCPServer
	tools   []MCPTool
	mu      sync.RWMutex
}

// NewMCPManager creates a new MCP manager
func NewMCPManager() *MCPManager {
	return &MCPManager{
		servers: make(map[string]*MCPServer),
		tools:   make([]MCPTool, 0),
	}
}

// ConnectServer connects to an MCP server
func (m *MCPManager) ConnectServer(ctx context.Context, name, url string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.servers[name]; exists {
		return fmt.Errorf("server %s already connected", name)
	}

	server := &MCPServer{
		Name: name,
		URL:  url,
		Client: &MCPClient{
			serverURL: url,
			clientID:  name,
		},
	}

	// Initialize connection
	// In production, this would establish WebSocket or HTTP connection
	server.Connected = true
	m.servers[name] = server

	// Discover tools
	tools, err := m.discoverTools(ctx, server)
	if err != nil {
		return fmt.Errorf("discovering tools: %w", err)
	}

	server.Tools = tools
	m.tools = append(m.tools, tools...)

	return nil
}

// discoverTools gets available tools from an MCP server
func (m *MCPManager) discoverTools(ctx context.Context, server *MCPServer) ([]MCPTool, error) {
	// In production, this would call the MCP tools listing API
	// For now, return mock tools
	return []MCPTool{
		{
			Name:        fmt.Sprintf("%s.search", server.Name),
			Description: fmt.Sprintf("Search using %s", server.Name),
			Server:      server.Name,
			Schema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"query": map[string]interface{}{
						"type": "string",
					},
				},
			},
		},
	}, nil
}

// ExecuteTool executes a tool call
func (m *MCPManager) ExecuteTool(ctx context.Context, toolName string, args map[string]interface{}) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Find the tool
	var targetTool *MCPTool
	for _, tool := range m.tools {
		if tool.Name == toolName {
			targetTool = &tool
			break
		}
	}

	if targetTool == nil {
		return "", fmt.Errorf("tool %s not found", toolName)
	}

	// Find the server
	server, exists := m.servers[targetTool.Server]
	if !exists {
		return "", fmt.Errorf("server %s not connected", targetTool.Server)
	}

	// Execute via MCP
	return server.Client.CallTool(ctx, toolName, args)
}

// CallTool makes a JSON-RPC call to the MCP server
func (c *MCPClient) CallTool(ctx context.Context, toolName string, args map[string]interface{}) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Build JSON-RPC request
	request := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/call",
		"params": map[string]interface{}{
			"name":      toolName,
			"arguments": args,
		},
	}

	requestJSON, _ := json.Marshal(request)

	// In production, send HTTP/WebSocket request to server.URL
	// For now, return placeholder
	return fmt.Sprintf("MCP call to %s: %s", c.serverURL, string(requestJSON)), nil
}

// GetTools returns all available MCP tools
func (m *MCPManager) GetTools() []MCPTool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]MCPTool, len(m.tools))
	copy(result, m.tools)
	return result
}

// DisconnectServer disconnects from an MCP server
func (m *MCPManager) DisconnectServer(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	server, exists := m.servers[name]
	if !exists {
		return fmt.Errorf("server %s not found", name)
	}

	server.Connected = false
	delete(m.servers, name)

	// Remove tools from this server
	var filtered []MCPTool
	for _, tool := range m.tools {
		if tool.Server != name {
			filtered = append(filtered, tool)
		}
	}
	m.tools = filtered

	return nil
}

// GetStatus returns MCP connection status
func (m *MCPManager) GetStatus() string {
	var b strings.Builder
	b.WriteString("## MCP Servers\n\n")

	for name, server := range m.servers {
		status := "✅ Connected"
		if !server.Connected {
			status = "❌ Disconnected"
		}
		b.WriteString(fmt.Sprintf("- %s: %s (%d tools)\n", name, status, len(server.Tools)))
	}

	return b.String()
}
