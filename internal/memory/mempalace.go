package memory

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// MemPalace implements the memory palace architecture
// Based on the method of loci: Wings -> Rooms -> Halls -> Drawers
type MemPalace struct {
	// Path to the palace data directory
	dataDir string
	
	// SQLite database for knowledge graph
	kgDB *sql.DB
	
	// ChromaDB client (would use CGO or HTTP API)
	// For now, we'll use a simplified vector store
	vectorStore *VectorStore
	
	// Current wing context
	currentWing string
}

// Drawer represents a single memory unit in the palace
type Drawer struct {
	ID        string    `json:"id"`
	Wing      string    `json:"wing"`
	Room      string    `json:"room"`
	Hall      string    `json:"hall"`
	Content   string    `json:"content"`
	AAAK      string    `json:"aaak"` // Compressed form
	Source    string    `json:"source"`
	Importance int      `json:"importance"`
	CreatedAt time.Time `json:"created_at"`
	Tags      []string  `json:"tags"`
}

// Wing represents a top-level memory category
type Wing struct {
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Rooms       []*Room    `json:"rooms"`
	CreatedAt   time.Time  `json:"created_at"`
}

// Room represents a sub-category within a wing
type Room struct {
	Name      string     `json:"name"`
	Wing      string     `json:"wing"`
	Halls     []*Hall    `json:"halls"`
	Tunnels   []string   `json:"tunnels"` // Links to rooms in other wings
	CreatedAt time.Time  `json:"created_at"`
}

// Hall represents a category within a room
type Hall struct {
	Name      string     `json:"name"`
	Room      string     `json:"room"`
	Drawers   []*Drawer  `json:"drawers"`
	CreatedAt time.Time  `json:"created_at"`
}

// VectorStore is a simplified in-memory vector store
// In production, this would use ChromaDB or similar
type VectorStore struct {
	drawers []*Drawer
}

// NewMemPalace creates a new memory palace
func NewMemPalace(dataDir string) (*MemPalace, error) {
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("creating data directory: %w", err)
	}

	// Initialize SQLite for knowledge graph
	dbPath := filepath.Join(dataDir, "knowledge_graph.db")
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("opening SQLite: %w", err)
	}

	// Create tables
	if err := initKnowledgeGraph(db); err != nil {
		return nil, fmt.Errorf("initializing knowledge graph: %w", err)
	}

	palace := &MemPalace{
		dataDir:     dataDir,
		kgDB:        db,
		vectorStore: &VectorStore{drawers: make([]*Drawer, 0)},
	}

	// Load existing drawers into vector store
	if err := palace.loadDrawers(); err != nil {
		return nil, fmt.Errorf("loading drawers: %w", err)
	}

	return palace, nil
}

// initKnowledgeGraph creates the SQLite schema for the knowledge graph
func initKnowledgeGraph(db *sql.DB) error {
	schema := `
	CREATE TABLE IF NOT EXISTS entities (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		type TEXT NOT NULL,
		wing TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS triples (
		id TEXT PRIMARY KEY,
		subject TEXT NOT NULL,
		predicate TEXT NOT NULL,
		object TEXT NOT NULL,
		wing TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (subject) REFERENCES entities(id),
		FOREIGN KEY (object) REFERENCES entities(id)
	);

	CREATE TABLE IF NOT EXISTS diary (
		id TEXT PRIMARY KEY,
		content TEXT NOT NULL,
		timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
		wing TEXT
	);

	CREATE INDEX IF NOT EXISTS idx_triples_subject ON triples(subject);
	CREATE INDEX IF NOT EXISTS idx_triples_predicate ON triples(predicate);
	CREATE INDEX IF NOT EXISTS idx_triples_object ON triples(object);
	`

	_, err := db.Exec(schema)
	return err
}

// loadDrawers loads existing drawers from disk
func (p *MemPalace) loadDrawers() error {
	drawersPath := filepath.Join(p.dataDir, "drawers.json")
	
	if _, err := os.Stat(drawersPath); os.IsNotExist(err) {
		return nil // No existing data
	}

	data, err := os.ReadFile(drawersPath)
	if err != nil {
		return fmt.Errorf("reading drawers file: %w", err)
	}

	return json.Unmarshal(data, &p.vectorStore.drawers)
}

// Save persists the memory palace to disk
func (p *MemPalace) Save() error {
	// Save drawers
	drawersPath := filepath.Join(p.dataDir, "drawers.json")
	data, err := json.MarshalIndent(p.vectorStore.drawers, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling drawers: %w", err)
	}

	if err := os.WriteFile(drawersPath, data, 0644); err != nil {
		return fmt.Errorf("writing drawers file: %w", err)
	}

	return nil
}

// StoreContent stores content into the memory palace with wing/room organization
func (p *MemPalace) StoreContent(content, wing, room, hall, source string) (*Drawer, error) {
	// Create drawer
	drawer := &Drawer{
		ID:        fmt.Sprintf("drawer_%d", time.Now().UnixNano()),
		Wing:      wing,
		Room:      room,
		Hall:      hall,
		Content:   content,
		AAAK:      AAAKCompress(content),
		Source:    source,
		Importance: calculateImportance(content),
		CreatedAt: time.Now(),
	}

	p.vectorStore.drawers = append(p.vectorStore.drawers, drawer)

	// Save to disk
	if err := p.Save(); err != nil {
		return nil, err
	}

	return drawer, nil
}

// Retrieve performs progressive context loading
// Level 0: Identity (170 tokens)
// Level 1: Top-15 important drawers
// Level 2: Wing/room scoped search
// Level 3: Full semantic search
func (p *MemPalace) Retrieve(query string, level int, wing, room string) ([]string, error) {
	switch level {
	case 0:
		return p.retrieveIdentity()
	case 1:
		return p.retrieveTopDrawers(15)
	case 2:
		return p.retrieveByScope(query, wing, room)
	case 3:
		return p.retrieveSemantic(query)
	default:
		return nil, fmt.Errorf("invalid retrieval level: %d", level)
	}
}

// retrieveIdentity returns minimal identity context (~170 tokens)
func (p *MemPalace) retrieveIdentity() ([]string, error) {
	// Return the most important drawers from current wing
	var results []string
	
	// Find high-import drawers
	for _, d := range p.vectorStore.drawers {
		if d.Wing == p.currentWing && d.Importance >= 8 {
			results = append(results, d.AAAK)
			if len(results) >= 5 {
				break
			}
		}
	}

	return results, nil
}

// retrieveTopDrawers returns the top N most important drawers
func (p *MemPalace) retrieveTopDrawers(n int) ([]string, error) {
	// Sort by importance (simple approach)
	sorted := make([]*Drawer, len(p.vectorStore.drawers))
	copy(sorted, p.vectorStore.drawers)
	
	// Simple bubble sort by importance
	for i := 0; i < len(sorted); i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[j].Importance > sorted[i].Importance {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}

	var results []string
	for i := 0; i < n && i < len(sorted); i++ {
		results = append(results, sorted[i].Content)
	}

	return results, nil
}

// retrieveByScope searches within a specific wing/room
func (p *MemPalace) retrieveByScope(query, wing, room string) ([]string, error) {
	var results []string
	
	for _, d := range p.vectorStore.drawers {
		if wing != "" && d.Wing != wing {
			continue
		}
		if room != "" && d.Room != room {
			continue
		}
		// Simple keyword matching
		if strings.Contains(strings.ToLower(d.Content), strings.ToLower(query)) {
			results = append(results, d.Content)
		}
	}

	return results, nil
}

// retrieveSemantic performs full semantic search
func (p *MemPalace) retrieveSemantic(query string) ([]string, error) {
	var results []string
	queryLower := strings.ToLower(query)
	
	// Simple keyword search (would use embeddings in production)
	for _, d := range p.vectorStore.drawers {
		if strings.Contains(strings.ToLower(d.Content), queryLower) {
			results = append(results, d.Content)
		}
	}

	return results, nil
}

// AddEntity adds an entity to the knowledge graph
func (p *MemPalace) AddEntity(id, name, entityType, wing string) error {
	_, err := p.kgDB.Exec(
		"INSERT OR REPLACE INTO entities (id, name, type, wing) VALUES (?, ?, ?, ?)",
		id, name, entityType, wing,
	)
	return err
}

// AddTriple adds a fact to the knowledge graph
func (p *MemPalace) AddTriple(subject, predicate, object, wing string) error {
	id := fmt.Sprintf("triple_%d", time.Now().UnixNano())
	_, err := p.kgDB.Exec(
		"INSERT INTO triples (id, subject, predicate, object, wing) VALUES (?, ?, ?, ?, ?)",
		id, subject, predicate, object, wing,
	)
	return err
}

// QueryKnowledgeGraph queries the knowledge graph
func (p *MemPalace) QueryKnowledgeGraph(subject, predicate string) ([]string, error) {
	query := `SELECT t.object FROM triples t 
			  WHERE t.subject = ? AND t.predicate = ?
			  ORDER BY t.created_at DESC`
	
	rows, err := p.kgDB.Query(query, subject, predicate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []string
	for rows.Next() {
		var obj string
		if err := rows.Scan(&obj); err != nil {
			return nil, err
		}
		results = append(results, obj)
	}

	return results, nil
}

// AddDiaryEntry adds a timestamped diary entry
func (p *MemPalace) AddDiaryEntry(content, wing string) error {
	id := fmt.Sprintf("diary_%d", time.Now().UnixNano())
	_, err := p.kgDB.Exec(
		"INSERT INTO diary (id, content, wing) VALUES (?, ?, ?)",
		id, content, wing,
	)
	return err
}

// MineCodebase recursively scans a codebase and populates the memory palace
func (p *MemPalace) MineCodebase(rootDir string) error {
	// Supported file extensions
	extensions := map[string]bool{
		".go": true, ".py": true, ".js": true, ".ts": true,
		".jsx": true, ".tsx": true, ".java": true, ".rs": true,
		".c": true, ".cpp": true, ".h": true, ".rb": true,
		".php": true, ".md": true, ".json": true, ".yaml": true,
		".yml": true, ".toml": true,
	}

	return filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors
		}

		if info.IsDir() {
			// Skip hidden directories and common non-code dirs
			if strings.HasPrefix(info.Name(), ".") || 
			   info.Name() == "node_modules" || 
			   info.Name() == "vendor" ||
			   info.Name() == ".git" {
				return filepath.SkipDir
			}
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		if !extensions[ext] {
			return nil
		}

		// Read file
		content, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		// Chunk content (800 chars, 100 overlap like MemPalace)
		chunks := chunkContent(string(content), 800, 100)
		
		// Determine wing/room based on path
		wing, room, hall := classifyPath(path)

		for i, chunk := range chunks {
			source := fmt.Sprintf("%s:chunk_%d", path, i)
			_, err := p.StoreContent(chunk, wing, room, hall, source)
			if err != nil {
				// Continue even if some chunks fail
				continue
			}
		}

		return nil
	})
}

// chunkContent splits content into overlapping chunks
func chunkContent(content string, chunkSize, overlap int) []string {
	var chunks []string
	
	for i := 0; i < len(content); i += chunkSize - overlap {
		end := i + chunkSize
		if end > len(content) {
			end = len(content)
		}
		chunks = append(chunks, content[i:end])
	}

	return chunks
}

// classifyPath determines the wing/room/hall for a file path
func classifyPath(path string) (wing, room, hall string) {
	// Default classification
	wing = "code"
	room = "source"
	hall = "implementation"

	// Check for specific patterns
	if strings.Contains(path, "/test") || strings.Contains(path, "_test.") {
		room = "tests"
	}
	
	if strings.Contains(path, "/docs") || strings.HasSuffix(path, ".md") {
		wing = "documentation"
		room = "guides"
	}

	if strings.Contains(path, "/config") || 
	   strings.HasSuffix(path, ".json") ||
	   strings.HasSuffix(path, ".yaml") ||
	   strings.HasSuffix(path, ".yml") ||
	   strings.HasSuffix(path, ".toml") {
		wing = "configuration"
		room = "files"
	}

	return wing, room, hall
}

// calculateImportance calculates the importance score of content
func calculateImportance(content string) int {
	score := 5 // Base score

	// Keywords that indicate importance
	importantKeywords := []string{
		"interface", "struct", "class", "func", "const",
		"var", "type", "package", "import", "export",
	}

	for _, keyword := range importantKeywords {
		if strings.Contains(strings.ToLower(content), keyword) {
			score++
		}
	}

	// Cap at 10
	if score > 10 {
		score = 10
	}

	return score
}

// AAAKCompress applies the AAAK compression algorithm
// Deterministic, rule-based abbreviation scheme
func AAAKCompress(content string) string {
	// Simplified AAAK compression
	// In production, this would use regex entity codes, keyword frequency, etc.
	
	// Extract key sentences (first sentence or lines with keywords)
	lines := strings.Split(content, "\n")
	var keyLines []string
	
	keywords := []string{"func", "type", "interface", "struct", "class", "const", "def"}
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		
		// Check if line contains important keywords
		for _, kw := range keywords {
			if strings.Contains(line, kw) {
				keyLines = append(keyLines, truncateString(line, 55))
				break
			}
		}
		
		// Limit key lines
		if len(keyLines) >= 5 {
			break
		}
	}

	return strings.Join(keyLines, " | ")
}

// truncateString truncates a string to maxLen characters
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// SetCurrentWing sets the current wing context
func (p *MemPalace) SetCurrentWing(wing string) {
	p.currentWing = wing
}

// Close closes the memory palace
func (p *MemPalace) Close() error {
	return p.kgDB.Close()
}
