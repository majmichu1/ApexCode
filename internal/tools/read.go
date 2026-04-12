package tools

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	// MAX_BYTES is the maximum file size to read (50KB like opencode)
	MAX_BYTES = 50 * 1024
	// MAX_LINE_LENGTH is the maximum length per line
	MAX_LINE_LENGTH = 2000
	// DEFAULT_LIMIT is the default number of lines to read
	DEFAULT_LIMIT = 2000
)

// Binary extensions that should not be read as text
var binaryExtensions = map[string]bool{
	".zip": true, ".exe": true, ".class": true, ".dll": true,
	".so": true, ".dylib": true, ".o": true, ".a": true,
	".png": true, ".jpg": true, ".jpeg": true, ".gif": true,
	".bmp": true, ".ico": true, ".svg": true,
	".pdf": true, ".doc": true, ".docx": true,
	".xls": true, ".xlsx": true,
	".mp3": true, ".mp4": true, ".avi": true, ".mov": true,
	".tar": true, ".gz": true, ".bz2": true, ".7z": true,
	".bin": true, ".dat": true,
}

// FileReadTool reads file contents with streaming, binary detection, and limits
// Based on opencode's read.ts
type FileReadTool struct {
	workDir string
}

func NewFileReadTool(workDir string) *FileReadTool {
	return &FileReadTool{workDir: workDir}
}

func (t *FileReadTool) Name() string {
	return "read_file"
}

func (t *FileReadTool) Description() string {
	return "Read the contents of a file or directory. Supports offset/limit for large files. Binary files are rejected."
}

func (t *FileReadTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"file_path": map[string]interface{}{
				"type":        "string",
				"description": "Absolute path to the file or directory to read",
			},
			"offset": map[string]interface{}{
				"type":        "integer",
				"description": "Starting line number (1-indexed, default: 1)",
			},
			"limit": map[string]interface{}{
				"type":        "integer",
				"description": "Maximum number of lines to read (default: 2000)",
			},
		},
		"required": []string{"file_path"},
	}
}

func (t *FileReadTool) Execute(ctx context.Context, args map[string]interface{}) (string, error) {
	filePath, ok := args["file_path"].(string)
	if !ok {
		return "", fmt.Errorf("file_path argument is required")
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

	// Check if path exists
	info, err := os.Stat(filePath)
	if err != nil {
		return "", fmt.Errorf("path does not exist: %s", filePath)
	}

	// If directory, list contents
	if info.IsDir() {
		return t.readDirectory(filePath, args)
	}

	// Check for binary file
	if isBinaryExtension(filePath) {
		return "", fmt.Errorf("cannot read binary file: %s", filePath)
	}

	// Read file with streaming
	return t.readFile(filePath, args)
}

func (t *FileReadTool) readFile(filePath string, args map[string]interface{}) (string, error) {
	// Get file info
	info, err := os.Stat(filePath)
	if err != nil {
		return "", fmt.Errorf("stating file: %w", err)
	}

	// Check size limit
	if info.Size() > MAX_BYTES {
		// File is too large, will read with limits
	}

	// Open file
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("opening file: %w", err)
	}
	defer file.Close()

	// Check for binary content by sampling first 4096 bytes
	if isBinaryContent(file) {
		return "", fmt.Errorf("file appears to be binary: %s", filePath)
	}

	// Reset to beginning
	file.Seek(0, 0)

	// Parse offset and limit
	offset := 1
	if o, ok := args["offset"].(float64); ok {
		offset = int(o)
	}

	limit := DEFAULT_LIMIT
	if l, ok := args["limit"].(float64); ok && l > 0 {
		limit = int(l)
	}

	// Read lines using streaming readline
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, MAX_LINE_LENGTH), MAX_LINE_LENGTH)

	var lines []string
	lineNum := 0
	bytesRead := 0
	more := false
	cut := false

	startLine := offset - 1 // Convert to 0-indexed

	for scanner.Scan() {
		line := scanner.Text()

		// Truncate long lines
		if len(line) > MAX_LINE_LENGTH {
			line = line[:MAX_LINE_LENGTH] + "... (line truncated)"
			cut = true
		}

		// Apply offset/limit
		if lineNum >= startLine {
			if len(lines) < limit {
				lines = append(lines, fmt.Sprintf("%d: %s", lineNum+1, line))
			} else {
				more = true
			}
		}

		bytesRead += len(line) + 1 // +1 for newline
		if bytesRead > MAX_BYTES {
			cut = true
			break
		}

		lineNum++
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("reading file: %w", err)
	}

	// Build result
	relPath, _ := filepath.Rel(t.workDir, filePath)
	var result strings.Builder
	result.WriteString(fmt.Sprintf("File: %s (%d lines)\n\n", relPath, lineNum))
	result.WriteString(strings.Join(lines, "\n"))

	if cut {
		result.WriteString("\n\n... (content truncated, exceeded size limit)")
	}
	if more {
		result.WriteString(fmt.Sprintf("\n\n... (%d more lines, use offset/limit to read more)", lineNum-startLine-limit))
	}

	return result.String(), nil
}

func (t *FileReadTool) readDirectory(dirPath string, args map[string]interface{}) (string, error) {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return "", fmt.Errorf("reading directory: %w", err)
	}

	// Parse offset and limit
	offset := 0
	if o, ok := args["offset"].(float64); ok {
		offset = int(o)
	}

	limit := len(entries)
	if l, ok := args["limit"].(float64); ok && l > 0 {
		limit = int(l)
	}

	// Apply offset/limit
	if offset >= len(entries) {
		return "", nil
	}
	if offset+limit > len(entries) {
		limit = len(entries) - offset
	}
	entries = entries[offset : offset+limit]

	// Build tree-like output
	relPath, _ := filepath.Rel(t.workDir, dirPath)
	var result strings.Builder
	result.WriteString(fmt.Sprintf("Directory: %s/\n\n", relPath))

	for _, entry := range entries {
		prefix := "  "
		if entry.IsDir() {
			result.WriteString(fmt.Sprintf("%s📁 %s/\n", prefix, entry.Name()))
		} else {
			result.WriteString(fmt.Sprintf("%s📄 %s\n", prefix, entry.Name()))
		}
	}

	return result.String(), nil
}

// isBinaryExtension checks if a file extension is typically binary
func isBinaryExtension(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return binaryExtensions[ext]
}

// isBinaryContent samples file content to detect binary files
func isBinaryContent(file *os.File) bool {
	buf := make([]byte, 4096)
	n, err := file.Read(buf)
	if err != nil || n == 0 {
		return false // Empty file is not binary
	}

	// Count non-printable characters
	nonPrintable := 0
	for _, b := range buf[:n] {
		if b < 32 && b != 9 && b != 10 && b != 13 { // Allow tab, newline, carriage return
			nonPrintable++
		}
	}

	// If >30% non-printable, consider it binary
	return float64(nonPrintable)/float64(n) > 0.3
}
