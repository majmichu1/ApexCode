package agent

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// FileMentionDetector detects file mentions in LLM responses
// Based on aider's get_file_mentions()
type FileMentionDetector struct {
	workDir      string
	allFiles     []string // All tracked files in project
	addableFiles []string // Files that can be added to chat
	chatFiles    map[string]bool // Files already in chat
}

// NewFileMentionDetector creates a new detector
func NewFileMentionDetector(workDir string, allFiles []string, chatFiles map[string]bool) *FileMentionDetector {
	return &FileMentionDetector{
		workDir:   workDir,
		allFiles:  allFiles,
		chatFiles: chatFiles,
	}
}

// Detect finds file mentions in LLM response text
func (d *FileMentionDetector) Detect(content string) []string {
	var mentioned []string
	mentionedSet := make(map[string]bool)

	// Split content into words, strip punctuation and quotes
	words := strings.Fields(content)
	
	for _, word := range words {
		// Strip punctuation and quotes
		word = stripPunctuation(word)
		if word == "" {
			continue
		}

		// Check if word matches a relative filename among addable files
		for _, f := range d.addableFiles {
			baseName := filepath.Base(f)
			
			// Exact match
			if word == f || word == baseName {
				if !mentionedSet[f] && !d.chatFiles[f] {
					// Only add if basename is unique or contains path separators
					if d.isUniqueBaseName(baseName) || strings.Contains(f, "/") || 
					   strings.Contains(f, ".") || strings.Contains(f, "_") || 
					   strings.Contains(f, "-") {
						mentioned = append(mentioned, f)
						mentionedSet[f] = true
					}
				}
			}
		}
	}

	// Check for identifier-to-filename matching
	// Split mentioned identifiers by non-alphanumeric
	identifiers := extractIdentifiers(content)
	for _, ident := range identifiers {
		if len(ident) < 5 {
			continue // Skip short identifiers
		}

		// Check if identifier matches the stem of any tracked file path
		for _, f := range d.allFiles {
			stem := fileStem(f)
			if strings.Contains(strings.ToLower(stem), strings.ToLower(ident)) {
				if !mentionedSet[f] && !d.chatFiles[f] {
					mentioned = append(mentioned, f)
					mentionedSet[f] = true
				}
			}
		}
	}

	return mentioned
}

// isUniqueBaseName checks if a basename is unique among files
func (d *FileMentionDetector) isUniqueBaseName(baseName string) bool {
	count := 0
	for _, f := range d.addableFiles {
		if filepath.Base(f) == baseName {
			count++
			if count > 1 {
				return false
			}
		}
	}
	return count == 1
}

// GetConfirmationMessage builds a message asking user to confirm adding files
func GetConfirmationMessage(mentioned []string) string {
	if len(mentioned) == 0 {
		return ""
	}

	msg := "The following files were mentioned. Add them to the chat?\n\n"
	for _, f := range mentioned {
		msg += fmt.Sprintf("- %s\n", f)
	}
	msg += "\nReply 'yes' to add, 'all' to add all, or 'skip' to ignore."
	
	return msg
}

// Helper functions

func stripPunctuation(s string) string {
	// Remove common punctuation and quote characters
	re := regexp.MustCompile(`[,"'`+"`"+`()[\]{}:;!?]`)
	return re.ReplaceAllString(s, "")
}

func extractIdentifiers(content string) []string {
	// Split by non-alphanumeric characters
	re := regexp.MustCompile(`[^a-zA-Z0-9_]+`)
	parts := re.Split(content, -1)
	
	var identifiers []string
	seen := make(map[string]bool)
	
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" && !seen[p] {
			identifiers = append(identifiers, p)
			seen[p] = true
		}
	}
	
	return identifiers
}

func fileStem(path string) string {
	base := filepath.Base(path)
	ext := filepath.Ext(base)
	if ext != "" {
		return base[:len(base)-len(ext)]
	}
	return base
}

// GetAllTrackedFiles returns all tracked files in the project
func GetAllTrackedFiles(workDir string) ([]string, error) {
	var files []string
	
	// Common code file extensions
	extensions := map[string]bool{
		".go": true, ".py": true, ".js": true, ".ts": true,
		".jsx": true, ".tsx": true, ".java": true, ".rs": true,
		".c": true, ".cpp": true, ".h": true, ".hpp": true,
		".rb": true, ".php": true, ".swift": true, ".kt": true,
		".md": true, ".txt": true,
	}

	err := filepath.Walk(workDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		if info.IsDir() {
			// Skip hidden and vendor directories
			if strings.HasPrefix(info.Name(), ".") ||
				info.Name() == "node_modules" ||
				info.Name() == "vendor" ||
				info.Name() == ".git" {
				return filepath.SkipDir
			}
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		if extensions[ext] {
			rel, _ := filepath.Rel(workDir, path)
			files = append(files, rel)
		}

		return nil
	})

	return files, err
}
