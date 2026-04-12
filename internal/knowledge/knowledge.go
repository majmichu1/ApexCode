package knowledge

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// KnowledgeBase implements Karpathy's LLM Wiki workflow
// Collect → Compile → Query pipeline with backlinks
type KnowledgeBase struct {
	rootDir  string
	inboxDir string
	templatesDir string
	folders  []string // projects, research, reference, meetings, etc.
}

// Note represents a wiki note
type Note struct {
	Title      string
	Summary    string
	Tags       []string
	Created    time.Time
	LastUpdated time.Time
	Content    string
	RelatedNotes []string // [[backlinks]]
	FilePath   string
}

// NewKnowledgeBase creates a new knowledge base
func NewKnowledgeBase(rootDir string) (*KnowledgeBase, error) {
	kb := &KnowledgeBase{
		rootDir:  rootDir,
		inboxDir: filepath.Join(rootDir, "inbox"),
		templatesDir: filepath.Join(rootDir, "_templates"),
		folders:  []string{"projects", "research", "reference", "meetings"},
	}

	// Create directory structure
	if err := kb.initialize(); err != nil {
		return nil, fmt.Errorf("initializing knowledge base: %w", err)
	}

	return kb, nil
}

// initialize creates the directory structure
func (kb *KnowledgeBase) initialize() error {
	dirs := []string{
		kb.rootDir,
		kb.inboxDir,
		kb.templatesDir,
	}

	for _, folder := range kb.folders {
		dirs = append(dirs, filepath.Join(kb.rootDir, folder))
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}

	// Create default template if not exists
	templatePath := filepath.Join(kb.templatesDir, "note.md")
	if _, err := os.Stat(templatePath); os.IsNotExist(err) {
		template := `# {{Title}}
**Summary**: {{Summary}}
**Tags**: {{Tags}}
**Created**: {{Created}}
**Last Updated**: {{Updated}}

---

## Content
{{Content}}

## Related Notes
{{Backlinks}}
`
		if err := os.WriteFile(templatePath, []byte(template), 0644); err != nil {
			return err
		}
	}

	return nil
}

// AddToInbox adds a raw note to the inbox
func (kb *KnowledgeBase) AddToInbox(title, content string) error {
	// Sanitize filename
	filename := sanitizeFilename(title) + ".md"
	path := filepath.Join(kb.inboxDir, filename)

	note := Note{
		Title:   title,
		Content: content,
		Created: time.Now(),
		LastUpdated: time.Now(),
	}

	if err := kb.saveNote(path, note); err != nil {
		return fmt.Errorf("saving to inbox: %w", err)
	}

	return nil
}

// ListInbox returns all items in the inbox
func (kb *KnowledgeBase) ListInbox() ([]string, error) {
	entries, err := os.ReadDir(kb.inboxDir)
	if err != nil {
		return nil, err
	}

	var items []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".md") {
			items = append(items, entry.Name())
		}
	}

	return items, nil
}

// TriageInbox processes inbox items and suggests where to file them
// In production, would call LLM for this
func (kb *KnowledgeBase) TriageInbox() map[string]string {
	// Returns filename -> suggested folder mapping
	suggestions := make(map[string]string)

	items, err := kb.ListInbox()
	if err != nil {
		return suggestions
	}

	for _, item := range items {
		// Simple heuristic-based triaging
		// In production, would use LLM to determine best folder
		content, _ := os.ReadFile(filepath.Join(kb.inboxDir, item))
		contentStr := strings.ToLower(string(content))

		if strings.Contains(contentStr, "project") || strings.Contains(contentStr, "feature") {
			suggestions[item] = "projects"
		} else if strings.Contains(contentStr, "research") || strings.Contains(contentStr, "study") {
			suggestions[item] = "research"
		} else if strings.Contains(contentStr, "api") || strings.Contains(contentStr, "reference") {
			suggestions[item] = "reference"
		} else if strings.Contains(contentStr, "meeting") || strings.Contains(contentStr, "discussion") {
			suggestions[item] = "meetings"
		} else {
			suggestions[item] = "reference" // Default
		}
	}

	return suggestions
}

// FileNote moves a note from inbox to appropriate folder
func (kb *KnowledgeBase) FileNote(filename, folder string) error {
	src := filepath.Join(kb.inboxDir, filename)
	dest := filepath.Join(kb.rootDir, folder, filename)

	// Read note
	note, err := kb.loadNote(src)
	if err != nil {
		return err
	}

	// Format note with template
	formatted := kb.formatNote(note)
	if err := os.WriteFile(dest, []byte(formatted), 0644); err != nil {
		return err
	}

	// Remove from inbox
	os.Remove(src)

	// Update backlinks
	kb.updateBacklinks(dest)

	return nil
}

// Query searches the knowledge base
func (kb *KnowledgeBase) Query(query string) ([]Note, error) {
	var results []Note
	queryLower := strings.ToLower(query)

	// Search all folders
	allFolders := append(kb.folders, "inbox")
	for _, folder := range allFolders {
		folderPath := filepath.Join(kb.rootDir, folder)
		entries, err := os.ReadDir(folderPath)
		if err != nil {
			continue
		}

		for _, entry := range entries {
			if !strings.HasSuffix(entry.Name(), ".md") {
				continue
			}

			note, err := kb.loadNote(filepath.Join(folderPath, entry.Name()))
			if err != nil {
				continue
			}

			// Check if note matches query (via summary, tags, or content)
			if kb.noteMatchesQuery(note, queryLower) {
				results = append(results, note)
			}
		}
	}

	return results, nil
}

// GetBacklinks returns notes that link to a given note
func (kb *KnowledgeBase) GetBacklinks(notePath string) ([]string, error) {
	var backlinks []string
	noteTitle := filepath.Base(notePath)
	noteTitle = strings.TrimSuffix(noteTitle, ".md")

	// Search for [[noteTitle]] pattern in all notes
	allFolders := append(kb.folders, "inbox")
	for _, folder := range allFolders {
		folderPath := filepath.Join(kb.rootDir, folder)
		entries, err := os.ReadDir(folderPath)
		if err != nil {
			continue
		}

		for _, entry := range entries {
			if !strings.HasSuffix(entry.Name(), ".md") {
				continue
			}

			content, err := os.ReadFile(filepath.Join(folderPath, entry.Name()))
			if err != nil {
				continue
			}

			// Check for backlink pattern
			if strings.Contains(string(content), fmt.Sprintf("[[%s]]", noteTitle)) {
				backlinks = append(backlinks, filepath.Join(folderPath, entry.Name()))
			}
		}
	}

	return backlinks, nil
}

// CompileNote compiles a raw source into a structured wiki article
// In production, would call LLM to restructure
func (kb *KnowledgeBase) CompileNote(title, summary string, tags []string, content string) error {
	note := Note{
		Title:   title,
		Summary: summary,
		Tags:    tags,
		Content: content,
		Created: time.Now(),
		LastUpdated: time.Now(),
	}

	// Save to appropriate folder
	folder := "reference" // Default
	if len(tags) > 0 {
		// Use first tag to determine folder
		tag := strings.ToLower(tags[0])
		for _, f := range kb.folders {
			if strings.Contains(f, tag) {
				folder = f
				break
			}
		}
	}

	path := filepath.Join(kb.rootDir, folder, sanitizeFilename(title)+".md")
	return kb.saveNote(path, note)
}

// Helper functions

func (kb *KnowledgeBase) loadNote(path string) (Note, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Note{}, err
	}

	return parseNote(string(data), path), nil
}

func (kb *KnowledgeBase) saveNote(path string, note Note) error {
	formatted := kb.formatNote(note)
	return os.WriteFile(path, []byte(formatted), 0644)
}

func (kb *KnowledgeBase) formatNote(note Note) string {
	tagsStr := strings.Join(note.Tags, " ")
	backlinks := ""
	if len(note.RelatedNotes) > 0 {
		for _, related := range note.RelatedNotes {
			backlinks += fmt.Sprintf("- [[%s]]\n", related)
		}
	} else {
		backlinks = "_No related notes yet_\n"
	}

	template := fmt.Sprintf(`# %s
**Summary**: %s
**Tags**: %s
**Created**: %s
**Last Updated**: %s

---

## Content
%s

## Related Notes
%s
`,
		note.Title,
		note.Summary,
		tagsStr,
		note.Created.Format(time.RFC3339),
		note.LastUpdated.Format(time.RFC3339),
		note.Content,
		backlinks,
	)

	return template
}

func parseNote(content, path string) Note {
	note := Note{
		FilePath: path,
	}

	lines := strings.Split(content, "\n")
	
	// Parse title
	if len(lines) > 0 && strings.HasPrefix(lines[0], "# ") {
		note.Title = strings.TrimPrefix(lines[0], "# ")
	}

	// Parse metadata
	for _, line := range lines {
		if strings.HasPrefix(line, "**Summary**:") {
			note.Summary = strings.TrimSpace(strings.TrimPrefix(line, "**Summary**:"))
		}
		if strings.HasPrefix(line, "**Tags**:") {
			tags := strings.TrimSpace(strings.TrimPrefix(line, "**Tags**:"))
			note.Tags = strings.Fields(tags)
		}
		if strings.HasPrefix(line, "**Created**:") {
			t := strings.TrimSpace(strings.TrimPrefix(line, "**Created**:"))
			note.Created, _ = time.Parse(time.RFC3339, t)
		}
	}

	// Parse content (everything after ---)
	contentIdx := -1
	for i, line := range lines {
		if line == "---" {
			contentIdx = i
			break
		}
	}

	if contentIdx >= 0 && contentIdx+1 < len(lines) {
		content := strings.Join(lines[contentIdx+1:], "\n")
		content = strings.TrimSpace(content)
		if !strings.HasPrefix(content, "## Content") {
			note.Content = content
		} else {
			// Skip the "## Content" header
			contentLines := strings.SplitN(content, "\n", 2)
			if len(contentLines) > 1 {
				note.Content = contentLines[1]
			}
		}
	}

	// Parse backlinks
	backlinkRe := regexp.MustCompile(`\[\[(.+?)\]\]`)
	matches := backlinkRe.FindAllStringSubmatch(content, -1)
	for _, match := range matches {
		note.RelatedNotes = append(note.RelatedNotes, match[1])
	}

	return note
}

func (kb *KnowledgeBase) noteMatchesQuery(note Note, queryLower string) bool {
	// Check summary
	if strings.Contains(strings.ToLower(note.Summary), queryLower) {
		return true
	}

	// Check tags
	for _, tag := range note.Tags {
		if strings.Contains(strings.ToLower(tag), queryLower) {
			return true
		}
	}

	// Check content
	if strings.Contains(strings.ToLower(note.Content), queryLower) {
		return true
	}

	return false
}

func (kb *KnowledgeBase) updateBacklinks(notePath string) {
	// After saving a note, scan for backlinks and update Related Notes
	note, err := kb.loadNote(notePath)
	if err != nil {
		return
	}

	// Find all notes that reference this note
	backlinks, err := kb.GetBacklinks(notePath)
	if err != nil {
		return
	}

	// Update the note's related notes
	note.RelatedNotes = make([]string, 0, len(backlinks))
	for _, bl := range backlinks {
		note.RelatedNotes = append(note.RelatedNotes, filepath.Base(bl))
	}

	// Save updated note
	kb.saveNote(notePath, note)
}

func sanitizeFilename(name string) string {
	// Replace unsafe characters
	re := regexp.MustCompile(`[^a-zA-Z0-9_\- ]`)
	name = re.ReplaceAllString(name, "_")
	// Collapse multiple underscores
	re2 := regexp.MustCompile(`_+`)
	return re2.ReplaceAllString(name, "_")
}
