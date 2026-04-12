package tools

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// Tool defines a tool interface
type Tool interface {
	Name() string
	Description() string
	Parameters() map[string]interface{}
	Execute(ctx context.Context, args map[string]interface{}) (string, error)
}

// GrepTool searches for patterns in files using ripgrep
type GrepTool struct {
	workDir string
}

func NewGrepTool(workDir string) *GrepTool {
	return &GrepTool{workDir: workDir}
}

func (t *GrepTool) Name() string {
	return "grep"
}

func (t *GrepTool) Description() string {
	return "Search for a regex pattern in files using ripgrep"
}

func (t *GrepTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"pattern": map[string]interface{}{
				"type":        "string",
				"description": "The regex pattern to search for",
			},
			"path": map[string]interface{}{
				"type":        "string",
				"description": "File or directory to search in (default: project root)",
			},
			"glob": map[string]interface{}{
				"type":        "string",
				"description": "Glob pattern to filter files (e.g., '*.go')",
			},
		},
		"required": []string{"pattern"},
	}
}

func (t *GrepTool) Execute(ctx context.Context, args map[string]interface{}) (string, error) {
	pattern, ok := args["pattern"].(string)
	if !ok {
		return "", fmt.Errorf("pattern argument is required")
	}

	searchPath := t.workDir
	if p, ok := args["path"].(string); ok {
		searchPath = p
	}

	cmdArgs := []string{"rg", "--color=never", "--no-heading", "--line-number"}

	if glob, ok := args["glob"].(string); ok {
		cmdArgs = append(cmdArgs, "--glob", glob)
	}

	cmdArgs = append(cmdArgs, pattern, searchPath)

	cmd := exec.CommandContext(ctx, cmdArgs[0], cmdArgs[1:]...)
	output, err := cmd.CombinedOutput()

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			return "No matches found", nil
		}
		return string(output), nil
	}

	return string(output), nil
}

// GlobTool finds files matching a glob pattern
type GlobTool struct {
	workDir string
}

func NewGlobTool(workDir string) *GlobTool {
	return &GlobTool{workDir: workDir}
}

func (t *GlobTool) Name() string {
	return "glob"
}

func (t *GlobTool) Description() string {
	return "Find files matching a glob pattern"
}

func (t *GlobTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"pattern": map[string]interface{}{
				"type":        "string",
				"description": "The glob pattern to match (e.g., '**/*.go')",
			},
		},
		"required": []string{"pattern"},
	}
}

func (t *GlobTool) Execute(ctx context.Context, args map[string]interface{}) (string, error) {
	pattern, ok := args["pattern"].(string)
	if !ok {
		return "", fmt.Errorf("pattern argument is required")
	}

	matches, err := exec.CommandContext(ctx, "rg", "--files", "--glob", pattern, t.workDir).CombinedOutput()
	if err != nil {
		// Fallback to filepath.Glob
		return t.fallbackGlob(pattern)
	}

	if len(matches) == 0 {
		return "No files found", nil
	}

	return string(matches), nil
}

func (t *GlobTool) fallbackGlob(pattern string) (string, error) {
	matches, err := matchGlob(t.workDir, pattern)
	if err != nil {
		return "", err
	}
	if len(matches) == 0 {
		return "No files found", nil
	}
	return strings.Join(matches, "\n"), nil
}

func matchGlob(root, pattern string) ([]string, error) {
	// Simple glob fallback
	cmd := exec.Command("find", root, "-name", pattern)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, err
	}
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	var result []string
	for _, l := range lines {
		if l != "" {
			result = append(result, l)
		}
	}
	return result, nil
}

// WebFetchTool fetches content from a URL
type WebFetchTool struct{}

func NewWebFetchTool() *WebFetchTool {
	return &WebFetchTool{}
}

func (t *WebFetchTool) Name() string {
	return "web_fetch"
}

func (t *WebFetchTool) Description() string {
	return "Fetch content from a URL"
}

func (t *WebFetchTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"url": map[string]interface{}{
				"type":        "string",
				"description": "The URL to fetch content from",
			},
		},
		"required": []string{"url"},
	}
}

func (t *WebFetchTool) Execute(ctx context.Context, args map[string]interface{}) (string, error) {
	url, ok := args["url"].(string)
	if !ok {
		return "", fmt.Errorf("url argument is required")
	}

	cmd := exec.CommandContext(ctx, "curl", "-s", "-L", "--max-time", "30", url)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("fetching URL: %w", err)
	}

	return string(output), nil
}
