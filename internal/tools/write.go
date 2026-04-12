package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// FileWriteTool writes content to files with diff generation and file time assertion
// Based on opencode's write.ts
type FileWriteTool struct {
	workDir     string
	fileModTime map[string]time.Time // Track when files were last read
}

func NewFileWriteTool(workDir string) *FileWriteTool {
	return &FileWriteTool{
		workDir:     workDir,
		fileModTime: make(map[string]time.Time),
	}
}

func (t *FileWriteTool) Name() string {
	return "write_file"
}

func (t *FileWriteTool) Description() string {
	return "Write content to a file, creating it if it doesn't exist. Generates a diff if file exists."
}

func (t *FileWriteTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"file_path": map[string]interface{}{
				"type":        "string",
				"description": "Absolute path to the file to write",
			},
			"content": map[string]interface{}{
				"type":        "string",
				"description": "Content to write to the file",
			},
		},
		"required": []string{"file_path", "content"},
	}
}

func (t *FileWriteTool) Execute(ctx context.Context, args map[string]interface{}) (string, error) {
	filePath, ok := args["file_path"].(string)
	if !ok {
		return "", fmt.Errorf("file_path argument is required")
	}

	content, ok := args["content"].(string)
	if !ok {
		return "", fmt.Errorf("content argument is required and must be a string")
	}

	// Resolve path
	if !filepath.IsAbs(filePath) {
		filePath = filepath.Join(t.workDir, filePath)
	}

	// Security: ensure path is within work directory
	rel, err := filepath.Rel(t.workDir, filePath)
	if err != nil || strings.HasPrefix(rel, "..") {
		return "", fmt.Errorf("access denied: path outside working directory")
	}

	// Check if file exists and read old content for diff
	var oldContent string
	var fileExisted bool
	
	if _, err := os.Stat(filePath); err == nil {
		// File exists - check if it was modified externally
		info, err := os.Stat(filePath)
		if err != nil {
			return "", fmt.Errorf("stating file: %w", err)
		}

		// File time assertion - check if file was modified since we last read it
		if lastRead, tracked := t.fileModTime[filePath]; tracked {
			if info.ModTime().After(lastRead) {
				return "", fmt.Errorf(
					"file was modified externally since last read at %s. Please read the file again.",
					lastRead.Format(time.RFC3339),
				)
			}
		}

		// Read old content for diff
		oldData, err := os.ReadFile(filePath)
		if err == nil {
			oldContent = string(oldData)
			fileExisted = true
		}
	}

	// Create directory if needed
	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		return "", fmt.Errorf("creating directory: %w", err)
	}

	// Write file
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		return "", fmt.Errorf("writing file: %w", err)
	}

	// Track modification time
	t.fileModTime[filePath] = time.Now()

	// Generate diff if file existed
	var diffOutput string
	if fileExisted && oldContent != content {
		diffOutput = generateUnifiedDiff(oldContent, content, rel)
	}

	// Build result
	var result strings.Builder
	if fileExisted {
		result.WriteString(fmt.Sprintf("Successfully updated %s\n", rel))
		if diffOutput != "" {
			result.WriteString(fmt.Sprintf("\nDiff:\n%s", diffOutput))
		}
	} else {
		result.WriteString(fmt.Sprintf("Successfully created %s (%d bytes)\n", rel, len(content)))
	}

	return result.String(), nil
}

// SetFileModTime records when a file was last read
func (t *FileWriteTool) SetFileModTime(filePath string, modTime time.Time) {
	if !filepath.IsAbs(filePath) {
		filePath = filepath.Join(t.workDir, filePath)
	}
	t.fileModTime[filePath] = modTime
}

// generateUnifiedDiff creates a unified diff between two strings
func generateUnifiedDiff(oldContent, newContent, filename string) string {
	oldLines := strings.Split(oldContent, "\n")
	newLines := strings.Split(newContent, "\n")

	var result strings.Builder
	result.WriteString(fmt.Sprintf("--- a/%s\n", filename))
	result.WriteString(fmt.Sprintf("+++ b/%s\n", filename))

	// Find common prefix
	commonPrefix := 0
	for commonPrefix < len(oldLines) && commonPrefix < len(newLines) {
		if oldLines[commonPrefix] != newLines[commonPrefix] {
			break
		}
		commonPrefix++
	}

	// Find common suffix
	commonSuffix := 0
	for commonSuffix < len(oldLines)-commonPrefix && commonSuffix < len(newLines)-commonPrefix {
		if oldLines[len(oldLines)-1-commonSuffix] != newLines[len(newLines)-1-commonSuffix] {
			break
		}
		commonSuffix++
	}

	// Output changed region
	if commonPrefix > 0 || commonSuffix > 0 {
		result.WriteString(fmt.Sprintf("@@ -%d,%d +%d,%d @@\n",
			commonPrefix+1, len(oldLines)-commonSuffix-commonPrefix,
			commonPrefix+1, len(newLines)-commonSuffix-commonPrefix))
	} else {
		result.WriteString("@@ @@\n")
	}

	// Show removed lines
	for _, line := range oldLines[commonPrefix : len(oldLines)-commonSuffix] {
		result.WriteString("-" + line + "\n")
	}

	// Show added lines
	for _, line := range newLines[commonPrefix : len(newLines)-commonSuffix] {
		result.WriteString("+" + line + "\n")
	}

	return result.String()
}
