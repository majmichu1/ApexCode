package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/apexcode/apexcode/internal/tools/stringutils"
)

// FileEditTool performs string replacement with 9 fallback strategies
type FileEditTool struct {
	workDir     string
	lineEnding  string
	fileModTime map[string]time.Time
}

func NewFileEditTool(workDir string) *FileEditTool {
	return &FileEditTool{
		workDir:     workDir,
		lineEnding:  "\n",
		fileModTime: make(map[string]time.Time),
	}
}

func (t *FileEditTool) Name() string {
	return "edit_file"
}

func (t *FileEditTool) Description() string {
	return "Edit a file by replacing exact text. Uses 9 fallback strategies to find matches. Read the file first to see current content."
}

func (t *FileEditTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"path": map[string]interface{}{
				"type":        "string",
				"description": "Absolute path to the file to edit",
			},
			"old_string": map[string]interface{}{
				"type":        "string",
				"description": "Exact text to find and replace",
			},
			"new_string": map[string]interface{}{
				"type":        "string",
				"description": "New text to replace the old text with",
			},
			"replace_all": map[string]interface{}{
				"type":        "boolean",
				"description": "Replace all occurrences (default: false)",
			},
		},
		"required": []string{"path", "old_string", "new_string"},
	}
}

func (t *FileEditTool) Execute(ctx context.Context, args map[string]interface{}) (string, error) {
	path, ok := args["path"].(string)
	if !ok {
		return "", fmt.Errorf("path argument is required")
	}
	oldStr, ok := args["old_string"].(string)
	if !ok {
		return "", fmt.Errorf("old_string argument is required")
	}
	newStr, ok := args["new_string"].(string)
	if !ok {
		return "", fmt.Errorf("new_string argument is required")
	}
	replaceAll, _ := args["replace_all"].(bool)

	if !filepath.IsAbs(path) {
		path = filepath.Join(t.workDir, path)
	}
	rel, err := filepath.Rel(t.workDir, path)
	if err != nil || strings.HasPrefix(rel, "..") {
		return "", fmt.Errorf("access denied: path outside working directory")
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("reading file: %w", err)
	}
	contentStr := string(content)

	if oldStr == "" {
		if err := os.WriteFile(path, []byte(contentStr+newStr), 0644); err != nil {
			return "", fmt.Errorf("writing file: %w", err)
		}
		return fmt.Sprintf("Successfully appended to %s", path), nil
	}

	t.lineEnding = detectLineEnding2(contentStr)
	normalizedContent := normalizeLineEndings2(contentStr, "\n")
	normalizedSearch := normalizeLineEndings2(oldStr, "\n")

	replacers := []stringutils.Replacer{
		&stringutils.SimpleReplacer{},
		&stringutils.LineTrimmedReplacer{},
		&stringutils.BlockAnchorReplacer{},
		&stringutils.WhitespaceNormalizedReplacer{},
		&stringutils.IndentationFlexibleReplacer{},
		&stringutils.EscapeNormalizedReplacer{},
		&stringutils.TrimmedBoundaryReplacer{},
		&stringutils.ContextAwareReplacer{},
	}

	var match string
	var matchCount int
	for _, replacer := range replacers {
		found := t.collectMatches2(replacer, normalizedContent, normalizedSearch)
		if len(found) > 0 {
			match = found[0]
			matchCount = len(found)
			break
		}
	}

	if match == "" {
		return "", t.generateNotFoundError2(path, oldStr)
	}
	if matchCount > 1 && !replaceAll {
		return "", fmt.Errorf("found %d occurrences. Provide more context or set replace_all=true", matchCount)
	}

	var newContent string
	if replaceAll {
		newContent = strings.ReplaceAll(contentStr, match, newStr)
	} else {
		newContent = strings.Replace(contentStr, match, newStr, 1)
	}

	if t.lineEnding == "\r\n" {
		newContent = strings.ReplaceAll(newContent, "\n", "\r\n")
	}

	if err := os.WriteFile(path, []byte(newContent), 0644); err != nil {
		return "", fmt.Errorf("writing file: %w", err)
	}
	t.fileModTime[path] = time.Now()
	return fmt.Sprintf("Successfully edited %s", rel), nil
}

func (t *FileEditTool) collectMatches2(replacer stringutils.Replacer, content, search string) []string {
	var matches []string
	ch := replacer.Find(content, search)
	for m := range ch {
		matches = append(matches, m)
	}
	return matches
}

func (t *FileEditTool) generateNotFoundError2(path, search string) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("text not found in %s: %w", path, err)
	}
	preview := string(content)
	if len(preview) > 200 {
		preview = preview[:200] + "..."
	}
	return fmt.Errorf("text not found in %s. Use read_file first. Preview:\n%s", path, preview)
}

func detectLineEnding2(content string) string {
	if strings.Contains(content, "\r\n") {
		return "\r\n"
	}
	return "\n"
}

func normalizeLineEndings2(content, target string) string {
	content = strings.ReplaceAll(content, "\r\n", "\n")
	if target == "\r\n" {
		return strings.ReplaceAll(content, "\n", "\r\n")
	}
	return content
}
