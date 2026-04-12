package tools

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"syscall"
	"time"
)

// BashTool executes bash commands with streaming output, process groups, and timeout
// Based on opencode's bash.ts with web-tree-sitter AST parsing for permissions
type BashTool struct {
	workDir       string
	defaultTimeout time.Duration
	streamCallback func(string) // For streaming output
}

// NewBashTool creates a new bash tool
func NewBashTool(workDir string, timeout time.Duration) *BashTool {
	return &BashTool{
		workDir:       workDir,
		defaultTimeout: timeout,
	}
}

// SetStreamCallback sets a callback for streaming output
func (t *BashTool) SetStreamCallback(fn func(string)) {
	t.streamCallback = fn
}

func (t *BashTool) Name() string {
	return "bash"
}

func (t *BashTool) Description() string {
	return "Execute a bash command in the terminal. Preferred over chaining multiple file tools for complex operations."
}

func (t *BashTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"command": map[string]interface{}{
				"type":        "string",
				"description": "The bash command to execute",
			},
			"timeout": map[string]interface{}{
				"type":        "integer",
				"description": "Command timeout in milliseconds (default: 120000 = 2 min)",
			},
			"workdir": map[string]interface{}{
				"type":        "string",
				"description": "Working directory (default: project root)",
			},
			"description": map[string]interface{}{
				"type":        "string",
				"description": "A 5-10 word description of what the command does",
			},
		},
		"required": []string{"command"},
	}
}

func (t *BashTool) Execute(ctx context.Context, args map[string]interface{}) (string, error) {
	command, ok := args["command"].(string)
	if !ok {
		return "", fmt.Errorf("command argument is required and must be a string")
	}

	// Determine working directory
	workDir := t.workDir
	if wd, ok := args["workdir"].(string); ok && wd != "" {
		workDir = wd
	}

	// Determine timeout
	timeout := t.defaultTimeout
	if t, ok := args["timeout"].(float64); ok && t > 0 {
		timeout = time.Duration(t) * time.Millisecond
	}

	// Create command with context
	cmd := exec.CommandContext(ctx, "bash", "-c", command)
	cmd.Dir = workDir
	
	// Process group management (like opencode's detached: true)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true, // Put in new process group
	}

	// Create pipes for stdout and stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", fmt.Errorf("creating stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return "", fmt.Errorf("creating stderr pipe: %w", err)
	}

	// Start command
	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("starting command: %w", err)
	}

	// Stream output
	var outputBuilder strings.Builder
	done := make(chan struct{})

	go func() {
		defer close(done)
		
		// Read stdout with streaming
		buf := make([]byte, 4096)
		for {
			n, err := stdout.Read(buf)
			if n > 0 {
				text := string(buf[:n])
				outputBuilder.WriteString(text)
				
				// Stream to callback
				if t.streamCallback != nil {
					t.streamCallback(text)
				}
			}
			if err == io.EOF {
				break
			}
			if err != nil {
				break
			}
		}

		// Read stderr
		buf = make([]byte, 4096)
		for {
			n, err := stderr.Read(buf)
			if n > 0 {
				outputBuilder.WriteString(string(buf[:n]))
			}
			if err == io.EOF {
				break
			}
			if err != nil {
				break
			}
		}
	}()

	// Wait for completion with timeout
	select {
	case <-done:
		// Command finished reading
	case <-time.After(timeout):
		// Timeout - kill process group
		syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		return outputBuilder.String(), fmt.Errorf("command timed out after %v", timeout)
	case <-ctx.Done():
		// Context cancelled - kill process group
		syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		return outputBuilder.String(), fmt.Errorf("command cancelled")
	}

	// Wait for command to exit
	err = cmd.Wait()
	output := outputBuilder.String()

	// Truncate output if too large (30KB limit like opencode)
	maxOutput := 30000
	if len(output) > maxOutput {
		output = output[:maxOutput] + "\n... (output truncated, exceeded 30KB limit)"
	}

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			// Return output even on error (useful for debugging)
			return fmt.Sprintf("Exit code: %d\n%s", exitErr.ExitCode(), output), nil
		}
		return output, fmt.Errorf("command execution failed: %w", err)
	}

	return output, nil
}

// KillProcessGroup kills a process group by PID
func KillProcessGroup(pid int) {
	syscall.Kill(-pid, syscall.SIGKILL)
}
