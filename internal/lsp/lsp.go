package lsp

import (
	"fmt"
	"os/exec"
	"strings"
	"sync"
)

// Bridge manages LSP server connections
type Bridge struct {
	servers map[string]*LSPServer
	mu      sync.RWMutex
}

// LSPServer represents a connected language server
type LSPServer struct {
	Language string
	Cmd      *exec.Cmd
	Running  bool
}

// Diagnostic represents a code diagnostic (error, warning)
type Diagnostic struct {
	Severity string // "error", "warning", "info"
	Message  string
	Line     int
	Column   int
	File     string
}

// Completion represents a code completion
type Completion struct {
	Label      string
	Kind       string // "function", "variable", "type", etc.
	Detail     string
	InsertText string
}

// NewBridge creates a new LSP bridge
func NewBridge() *Bridge {
	return &Bridge{
		servers: make(map[string]*LSPServer),
	}
}

// StartServer launches an LSP server for a language
func (b *Bridge) StartServer(language string, command string, args []string) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if _, exists := b.servers[language]; exists {
		return fmt.Errorf("LSP server for %s already running", language)
	}

	cmd := exec.Command(command, args...)
	
	server := &LSPServer{
		Language: language,
		Cmd:      cmd,
		Running:  true,
	}

	b.servers[language] = server
	return nil
}

// StopServer stops an LSP server
func (b *Bridge) StopServer(language string) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	server, exists := b.servers[language]
	if !exists {
		return fmt.Errorf("LSP server for %s not found", language)
	}

	if server.Cmd.Process != nil {
		server.Cmd.Process.Kill()
	}
	server.Running = false
	delete(b.servers, language)

	return nil
}

// GetCompletions requests completions from the LSP server
func (b *Bridge) GetCompletions(file string, line int, column int) ([]Completion, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	// Find server for this file
	language := detectLanguage(file)
	server, exists := b.servers[language]
	if !exists || !server.Running {
		return nil, fmt.Errorf("no LSP server for %s", language)
	}

	// In production, send LSP textDocument/completion request
	// For now, return placeholder
	return []Completion{
		{
			Label:      "exampleFunction",
			Kind:       "function",
			Detail:     "Example function",
			InsertText: "exampleFunction()",
		},
	}, nil
}

// GetDiagnostics gets current diagnostics from the LSP server
func (b *Bridge) GetDiagnostics(file string) ([]Diagnostic, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	// In production, query LSP server for diagnostics
	// For now, run linter if available
	return runLinter(file)
}

// GetSymbolDefinition finds the definition of a symbol
func (b *Bridge) GetSymbolDefinition(file string, symbol string) (string, error) {
	// Use grep as a simple fallback for finding definitions
	cmd := exec.Command("grep", "-n", fmt.Sprintf("func.*%s", symbol), file)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("symbol not found: %s", symbol)
	}

	return string(output), nil
}

// GetStatus returns LSP connection status
func (b *Bridge) GetStatus() string {
	var bld strings.Builder
	bld.WriteString("## LSP Servers\n\n")

	b.mu.RLock()
	defer b.mu.RUnlock()

	for lang, server := range b.servers {
		status := "✅ Running"
		if !server.Running {
			status = "❌ Stopped"
		}
		bld.WriteString(fmt.Sprintf("- %s: %s\n", lang, status))
	}

	if len(b.servers) == 0 {
		bld.WriteString("No LSP servers running. Use `--lsp` flag to enable.\n")
	}

	return bld.String()
}

// detectLanguage determines the language from a file path
func detectLanguage(file string) string {
	if strings.HasSuffix(file, ".go") {
		return "go"
	} else if strings.HasSuffix(file, ".py") {
		return "python"
	} else if strings.HasSuffix(file, ".js") || strings.HasSuffix(file, ".jsx") {
		return "javascript"
	} else if strings.HasSuffix(file, ".ts") || strings.HasSuffix(file, ".tsx") {
		return "typescript"
	} else if strings.HasSuffix(file, ".java") {
		return "java"
	} else if strings.HasSuffix(file, ".rs") {
		return "rust"
	}
	return "unknown"
}

// runLinter runs a linter on the file
func runLinter(file string) ([]Diagnostic, error) {
	var diagnostics []Diagnostic

	// Determine language
	lang := detectLanguage(file)

	var cmd *exec.Cmd
	switch lang {
	case "go":
		cmd = exec.Command("go", "vet", file)
	case "python":
		cmd = exec.Command("pylint", file)
	case "javascript", "typescript":
		cmd = exec.Command("eslint", file)
	default:
		return diagnostics, nil
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		// Linter found issues, parse output
		lines := strings.Split(string(output), "\n")
		for _, line := range lines {
			if line == "" {
				continue
			}
			diagnostics = append(diagnostics, Diagnostic{
				Severity: "warning",
				Message:  line,
				File:     file,
			})
		}
	}

	return diagnostics, nil
}
