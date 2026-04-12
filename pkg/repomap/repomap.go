package repomap

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// RepoMap generates repository maps using PageRank algorithm
// Based on aider's repomap.py with tree-sitter + networkx PageRank + binary search
type RepoMap struct {
	workDir     string
	tokenBudget int
	files       []*FileNode
	edges       []*Edge
	ranks       map[string]float64
	tags        []*Tag
	importantFiles []string
}

// FileNode represents a file in the graph
type FileNode struct {
	Path     string
	Tags     []*Tag
	Rank     float64
	InChat   bool // File currently in chat
}

// Tag represents a code symbol (function, class, etc.)
type Tag struct {
	RelPath  string // Relative file path
	FilePath string // Full file path
	Line     int    // Line number
	Name     string // Symbol name
	Kind     string // "def" or "ref"
}

// Edge represents a dependency between files
type Edge struct {
	From     string // Referencing file
	To       string // Defining file
	Weight   float64
	Ident    string // Identifier that creates the link
	RefCount int    // Number of references
}

// NewRepoMap creates a new repository map generator
func NewRepoMap(workDir string, tokenBudget int) *RepoMap {
	return &RepoMap{
		workDir:     workDir,
		tokenBudget: tokenBudget,
		ranks:       make(map[string]float64),
		tags:        make([]*Tag, 0),
		importantFiles: findImportantFiles(workDir),
	}
}

// Generate creates the repository map
func (rm *RepoMap) Generate(chatFiles map[string]bool, mentionedIdents []string) (string, error) {
	// Step 1: Scan files and extract tags
	if err := rm.scanFiles(); err != nil {
		return "", err
	}

	// Step 2: Build dependency graph
	rm.buildGraph()

	// Step 3: Run PageRank with personalization
	rm.runPageRank(chatFiles, mentionedIdents)

	// Step 4: Distribute rank to tags
	rm.distributeRankToTags()

	// Step 5: Binary search for optimal tag count within token budget
	rankedTags := rm.getRankedTags()
	fittedTags := rm.fitToBudget(rankedTags)

	// Step 6: Render map
	return rm.renderMap(fittedTags), nil
}

// scanFiles walks the directory and extracts symbols
func (rm *RepoMap) scanFiles() error {
	extensions := map[string]string{
		".go":   "go", ".py": "python", ".js": "javascript",
		".ts":   "typescript", ".tsx": "typescript", ".java": "java",
		".rs":   "rust", ".c": "c", ".cpp": "cpp", ".rb": "ruby",
	}

	nodeMap := make(map[string]*FileNode)

	return filepath.Walk(rm.workDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			if info != nil && (strings.HasPrefix(info.Name(), ".") || 
				info.Name() == "node_modules" || info.Name() == ".git") {
				return filepath.SkipDir
			}
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		lang, ok := extensions[ext]
		if !ok {
			return nil
		}

		relPath, _ := filepath.Rel(rm.workDir, path)
		node := &FileNode{Path: relPath}
		nodeMap[relPath] = node
		rm.files = append(rm.files, node)

		// Extract tags (simplified - would use tree-sitter in production)
		tags := extractTags(path, relPath, lang)
		node.Tags = tags
		rm.tags = append(rm.tags, tags...)

		return nil
	})
}

// buildGraph creates the dependency graph
func (rm *RepoMap) buildGraph() {
	// Build identifier -> definition file mapping
	defMap := make(map[string][]string) // ident -> [files that define it]
	refMap := make(map[string][]string) // ident -> [files that reference it]

	for _, tag := range rm.tags {
		if tag.Kind == "def" {
			defMap[tag.Name] = append(defMap[tag.Name], tag.FilePath)
		} else if tag.Kind == "ref" {
			refMap[tag.Name] = append(refMap[tag.Name], tag.FilePath)
		}
	}

	// Create edges from referencer to definer
	for ident, refFiles := range refMap {
		defFiles, exists := defMap[ident]
		if !exists {
			continue
		}

		for _, refFile := range refFiles {
			for _, defFile := range defFiles {
				if refFile == defFile {
					continue // Skip self-references
				}

				// Calculate weight
				weight := rm.calculateEdgeWeight(ident, refFile)
				
				rm.edges = append(rm.edges, &Edge{
					From:     refFile,
					To:       defFile,
					Weight:   weight,
					Ident:    ident,
				})
			}
		}
	}

	// Add self-edges for files with no references (ensure they're not dropped)
	for _, node := range rm.files {
		hasEdge := false
		for _, e := range rm.edges {
			if e.From == node.Path || e.To == node.Path {
				hasEdge = true
				break
			}
		}
		if !hasEdge {
			rm.edges = append(rm.edges, &Edge{
				From:   node.Path,
				To:     node.Path,
				Weight: 0.1,
			})
		}
	}
}

// runPageRank runs personalized PageRank
func (rm *RepoMap) runPageRank(chatFiles map[string]bool, mentionedIdents []string) {
	// Build personalization vector
	personalization := make(map[string]float64)
	for _, node := range rm.files {
		p := 1.0 // Base personalization

		// Boost files in chat (50x)
		if chatFiles[node.Path] {
			p *= 50
			node.InChat = true
		}

		// Boost files matching mentioned identifiers (10x)
		for _, ident := range mentionedIdents {
			if strings.Contains(strings.ToLower(node.Path), strings.ToLower(ident)) {
				p *= 10
			}
		}

		personalization[node.Path] = p
	}

	// Normalize personalization
	total := 0.0
	for _, v := range personalization {
		total += v
	}
	for k := range personalization {
		personalization[k] /= total
	}

	// Run PageRank iterations (simplified - would use networkx in Python)
	damping := 0.85
	iterations := 100

	// Initialize ranks
	for _, node := range rm.files {
		rm.ranks[node.Path] = 1.0 / float64(len(rm.files))
	}

	for iter := 0; iter < iterations; iter++ {
		newRanks := make(map[string]float64)
		
		for _, node := range rm.files {
			// Random jump factor with personalization
			newRanks[node.Path] = (1 - damping) * personalization[node.Path]
		}

		// Distribute rank along edges
		for _, edge := range rm.edges {
			if rm.ranks[edge.From] > 0 {
				newRanks[edge.To] += damping * rm.ranks[edge.From] * edge.Weight
			}
		}

		// Update
		for path := range newRanks {
			rm.ranks[path] = newRanks[path]
		}
	}
}

// calculateEdgeWeight calculates the weight for an edge
func (rm *RepoMap) calculateEdgeWeight(ident, refFile string) float64 {
	weight := 1.0

	// *10 for multi-word names (snake_case, kebab-case, camelCase, length >= 8)
	if len(ident) >= 8 && (strings.Contains(ident, "_") || strings.Contains(ident, "-") || isCamelCase(ident)) {
		weight *= 10
	}

	// *0.1 for private identifiers (_ prefix)
	if strings.HasPrefix(ident, "_") {
		weight *= 0.1
	}

	// Count how many files define this identifier
	defCount := 0
	for _, tag := range rm.tags {
		if tag.Kind == "def" && tag.Name == ident {
			defCount++
		}
	}
	if defCount > 5 {
		weight *= 0.1 // Too common, reduce weight
	}

	return weight
}

// distributeRankToTags distributes file rank to its tags
func (rm *RepoMap) distributeRankToTags() {
	for _, tag := range rm.tags {
		nodeRank := rm.ranks[tag.FilePath]
		// Distribute proportionally
		tagRank := nodeRank / float64(len(rm.tags))
		_ = tagRank // Would store with tag
	}
}

// getRankedTags returns tags sorted by importance
func (rm *RepoMap) getRankedTags() []*Tag {
	// Sort tags by their file's PageRank
	sort.Slice(rm.tags, func(i, j int) bool {
		rankI := rm.ranks[rm.tags[i].FilePath]
		rankJ := rm.ranks[rm.tags[j].FilePath]
		return rankI > rankJ
	})

	return rm.tags
}

// fitToBudget uses binary search to find optimal tag count within budget
func (rm *RepoMap) fitToBudget(tags []*Tag) []*Tag {
	if len(tags) == 0 {
		return tags
	}

	// Binary search for the number of tags that fits
	low := 1
	high := len(tags)
	best := tags[:1]

	for low <= high {
		mid := (low + high) / 2
		candidate := tags[:mid]
		
		// Estimate token count (sample ~1% of lines)
		tokens := rm.estimateTokens(candidate)
		
		if tokens <= rm.tokenBudget {
			best = candidate
			low = mid + 1
		} else {
			high = mid - 1
		}

		// Allow 15% error margin for early termination
		err := float64(tokens-rm.tokenBudget) / float64(rm.tokenBudget)
		if err < 0.15 {
			break
		}
	}

	return best
}

// estimateTokens estimates the token count for tags
func (rm *RepoMap) estimateTokens(tags []*Tag) int {
	// Rough estimate: 1.3 tokens per word
	total := 0
	for _, tag := range tags {
		line := fmt.Sprintf("- %s `%s` (line %d)", tag.Kind, tag.Name, tag.Line)
		total += len(strings.Fields(line))
	}
	return int(float64(total) * 1.3)
}

// renderMap generates the text representation
func (rm *RepoMap) renderMap(tags []*Tag) string {
	var b strings.Builder

	// Prepend important files
	b.WriteString("# Repository Map\n\n")
	for _, f := range rm.importantFiles {
		b.WriteString(fmt.Sprintf("## ⭐ %s\n", f))
	}
	b.WriteString("\n")

	// Group tags by file
	fileTags := make(map[string][]*Tag)
	for _, tag := range tags {
		fileTags[tag.FilePath] = append(fileTags[tag.FilePath], tag)
	}

	// Sort files
	files := make([]string, 0, len(fileTags))
	for f := range fileTags {
		files = append(files, f)
	}
	sort.Strings(files)

	// Render
	for _, file := range files {
		tags := fileTags[file]
		b.WriteString(fmt.Sprintf("## %s\n", file))
		
		for _, tag := range tags {
			b.WriteString(fmt.Sprintf("- %s `%s` (line %d)\n", tag.Kind, tag.Name, tag.Line))
		}
		b.WriteString("\n")
	}

	return b.String()
}

// Helper functions

func extractTags(filePath, relPath, lang string) []*Tag {
	// Simplified extraction - would use tree-sitter in production
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil
	}

	lines := strings.Split(string(content), "\n")
	var tags []*Tag

	for lineNum, line := range lines {
		line = strings.TrimSpace(line)
		
		// Function definitions
		if isFunctionDef(line, lang) {
			name := extractName(line, lang)
			if name != "" {
				tags = append(tags, &Tag{
					RelPath:  relPath,
					FilePath: relPath,
					Line:     lineNum + 1,
					Name:     name,
					Kind:     "def",
				})
			}
		}

		// Function calls (references)
		if refs := extractReferences(line, lang); len(refs) > 0 {
			for _, ref := range refs {
				tags = append(tags, &Tag{
					RelPath:  relPath,
					FilePath: relPath,
					Line:     lineNum + 1,
					Name:     ref,
					Kind:     "ref",
				})
			}
		}
	}

	return tags
}

func isFunctionDef(line, lang string) bool {
	switch lang {
	case "go":
		return strings.HasPrefix(line, "func ")
	case "python":
		return strings.HasPrefix(line, "def ") || strings.HasPrefix(line, "class ")
	case "javascript", "typescript":
		return strings.HasPrefix(line, "function ") || 
			   strings.HasPrefix(line, "const ") ||
			   strings.HasPrefix(line, "class ")
	default:
		return false
	}
}

func extractName(line, lang string) string {
	fields := strings.Fields(line)
	for i, f := range fields {
		if f == "func" || f == "def" || f == "class" || f == "const" {
			if i+1 < len(fields) {
				name := fields[i+1]
				name = strings.TrimRight(name, "(")
				name = strings.TrimRight(name, "{")
				return name
			}
		}
	}
	return ""
}

func extractReferences(line, lang string) []string {
	// Very simplified - would use tree-sitter for accurate extraction
	var refs []string
	// Look for function calls
	fields := strings.Fields(line)
	for _, f := range fields {
		f = strings.TrimRight(f, "(")
		f = strings.TrimRight(f, ")")
		f = strings.TrimRight(f, ".")
		f = strings.TrimRight(f, ",")
		f = strings.TrimRight(f, ";")
		if len(f) > 2 && f[0] >= 'a' && f[0] <= 'z' {
			// Probably a function call
			refs = append(refs, f)
		}
	}
	return refs
}

func isCamelCase(s string) bool {
	hasLower := false
	hasUpper := false
	for _, r := range s {
		if r >= 'a' && r <= 'z' {
			hasLower = true
		}
		if r >= 'A' && r <= 'Z' {
			hasUpper = true
		}
	}
	return hasLower && hasUpper
}

func findImportantFiles(workDir string) []string {
	// Common important files (config, build, README, etc.)
	importantPatterns := []string{
		"README.md", "package.json", "go.mod", "Cargo.toml",
		"pyproject.toml", "Makefile", "Dockerfile", "docker-compose.yml",
		".github/workflows", "CMakeLists.txt", "build.gradle",
	}

	var important []string
	filepath.Walk(workDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}

		rel, _ := filepath.Rel(workDir, path)
		for _, pattern := range importantPatterns {
			if strings.Contains(rel, pattern) {
				important = append(important, rel)
				break
			}
		}
		return nil
	})

	return important
}
