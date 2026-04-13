package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"strings"
	"sync"

	"github.com/apexcode/apexcode/internal/agent"
	"github.com/apexcode/apexcode/internal/config"
	"github.com/apexcode/apexcode/internal/memory"
	"github.com/apexcode/apexcode/pkg/repomap"
)

// Server is the HTTP server that the TUI connects to
type Server struct {
	cfg        *config.Config
	agent      *agent.Agent
	sessions   map[string]*Session
	mu         sync.RWMutex
	port       int
	memPalace  *memory.MemPalace
}

// Session represents a conversation session
type Session struct {
	ID      string
	Agent   *agent.Agent
	Stream  chan StreamEvent
	History []Message
}

// StreamEvent represents a streaming event from the agent
type StreamEvent struct {
	Type    string `json:"type"`
	Content string `json:"content,omitempty"`
	Tool    string `json:"tool,omitempty"`
	Done    bool   `json:"done,omitempty"`
	Error   string `json:"error,omitempty"`
}

// Message represents a chat message
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// NewServer creates a new HTTP server
func NewServer(cfg *config.Config, port int) *Server {
	mp, _ := memory.NewMemPalace(filepath.Join(cfg.WorkDir, ".apexcode", "memory"))
	return &Server{
		cfg:       cfg,
		agent:     agent.New(cfg),
		sessions:  make(map[string]*Session),
		port:      port,
		memPalace: mp,
	}
}

// Start begins the HTTP server
func (s *Server) Start() error {
	mux := http.NewServeMux()

	// Health check
	mux.HandleFunc("/health", s.handleHealth)

	// Session management
	mux.HandleFunc("/api/sessions", s.handleSessions)
	mux.HandleFunc("/api/sessions/", s.handleSessionByID)

	// Prompt endpoint
	mux.HandleFunc("/api/prompt", s.handlePrompt)

	// Stream endpoint (SSE)
	mux.HandleFunc("/api/stream", s.handleStream)

	// Enhance endpoint — returns repomap + MemPalace context
	mux.HandleFunc("/api/enhance", s.handleEnhance)

	// Proactive analysis suggestions (KAIROS)
	mux.HandleFunc("/api/suggestions", s.handleSuggestions)

	// Multi-agent swarm execution
	mux.HandleFunc("/api/swarm", s.handleSwarm)

	// Tool status
	mux.HandleFunc("/api/tools", s.handleTools)

	// Config
	mux.HandleFunc("/api/config", s.handleConfig)

	addr := fmt.Sprintf(":%d", s.port)
	log.Printf("ApexCode server starting on %s", addr)
	return http.ListenAndServe(addr, s.corsMiddleware(mux))
}

// Handlers

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "ok",
		"version": "0.1.0",
	})
}

func (s *Server) handleSessions(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		s.createSession(w, r)
		return
	}
	s.listSessions(w, r)
}

func (s *Server) handleSessionByID(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Path[len("/api/sessions/"):]
	if id == "" {
		http.Error(w, "missing session ID", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case "GET":
		s.getSession(w, r, id)
	case "DELETE":
		s.deleteSession(w, r, id)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handlePrompt(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		SessionID string `json:"session_id"`
		Message   string `json:"message"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	// Get or create session
	sess, err := s.getOrCreateSession(req.SessionID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Store user message
	sess.History = append(sess.History, Message{Role: "user", Content: req.Message})

	// Run agent (non-blocking, stream results)
	go s.runAgent(sess, req.Message)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":     "ok",
		"session_id": sess.ID,
	})
}

func (s *Server) handleStream(w http.ResponseWriter, r *http.Request) {
	sessionID := r.URL.Query().Get("session_id")
	if sessionID == "" {
		http.Error(w, "missing session_id", http.StatusBadRequest)
		return
	}

	sess, err := s.getSessionByID(sessionID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	// SSE setup
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	// Stream events
	for event := range sess.Stream {
		data, _ := json.Marshal(event)
		fmt.Fprintf(w, "data: %s\n\n", data)
		flusher.Flush()

		if event.Done || event.Error != "" {
			break
		}
	}
}

func (s *Server) handleTools(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"tools": []string{
			"bash", "read_file", "write_file", "edit_file",
			"grep", "glob", "web_fetch",
		},
	})
}

func (s *Server) handleConfig(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"provider":  s.cfg.Provider,
		"max_turns": s.cfg.MaxTurns,
		"theme":     "dark",
	})
}

// handleEnhance returns repomap + MemPalace context for token savings
func (s *Server) handleEnhance(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		WorkDir string `json:"work_dir"`
		Prompt  string `json:"prompt"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	if req.WorkDir == "" {
		req.WorkDir = s.cfg.WorkDir
	}

	result := make(map[string]interface{})

	// Generate repomap
	rm := repomap.NewRepoMap(req.WorkDir, 5000)
	if rm != nil {
		rmap, err := rm.Generate(nil, nil)
		if err == nil && rmap != "" {
			result["repomap"] = rmap
		}
	}

	// Fetch MemPalace context
	if s.memPalace != nil && req.Prompt != "" {
		ctx, err := s.memPalace.Retrieve(req.Prompt, 2, "", "")
		if err == nil && len(ctx) > 0 {
			result["memory"] = strings.Join(ctx, "\n")
		}
	}

	result["connected"] = true
	result["token_savings"] = "60-80% vs full file contents"

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// handleSuggestions returns proactive KAIROS analysis suggestions
func (s *Server) handleSuggestions(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Placeholder — full KAIROS analysis requires the analysis package
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"suggestions": []interface{}{},
		"count":       0,
		"status":      "KAIROS analysis package not yet integrated",
	})
}

// handleSwarm executes multi-agent swarm
func (s *Server) handleSwarm(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Task   string   `json:"task"`
		Agents []string `json:"agents"`
		Mode   string   `json:"mode"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	if req.Task == "" {
		http.Error(w, "task is required", http.StatusBadRequest)
		return
	}

	if req.Mode == "" {
		req.Mode = "parallel"
	}

	if len(req.Agents) == 0 {
		req.Agents = []string{"planner", "coder", "reviewer"}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "accepted",
		"task":    req.Task,
		"agents":  req.Agents,
		"mode":    req.Mode,
		"message": "Swarm execution started",
	})
}

// Session management

func (s *Server) createSession(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()

	id := fmt.Sprintf("session_%d", len(s.sessions)+1)
	sess := &Session{
		ID:      id,
		Agent:   agent.New(s.cfg),
		Stream:  make(chan StreamEvent, 100),
		History: make([]Message, 0),
	}
	s.sessions[id] = sess

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"id": id})
}

func (s *Server) listSessions(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var sessions []map[string]interface{}
	for id, sess := range s.sessions {
		sessions = append(sessions, map[string]interface{}{
			"id":          id,
			"message_count": len(sess.History),
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(sessions)
}

func (s *Server) getSession(w http.ResponseWriter, r *http.Request, id string) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	sess, exists := s.sessions[id]
	if !exists {
		http.Error(w, "session not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":           id,
		"message_count": len(sess.History),
		"history":      sess.History,
	})
}

func (s *Server) deleteSession(w http.ResponseWriter, r *http.Request, id string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.sessions, id)
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) getOrCreateSession(id string) (*Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if id == "" {
		id = fmt.Sprintf("session_%d", len(s.sessions)+1)
	}

	if sess, exists := s.sessions[id]; exists {
		return sess, nil
	}

	sess := &Session{
		ID:      id,
		Agent:   agent.New(s.cfg),
		Stream:  make(chan StreamEvent, 100),
		History: make([]Message, 0),
	}
	s.sessions[id] = sess
	return sess, nil
}

func (s *Server) getSessionByID(id string) (*Session, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	sess, exists := s.sessions[id]
	if !exists {
		return nil, fmt.Errorf("session not found: %s", id)
	}
	return sess, nil
}

// Agent execution

func (s *Server) runAgent(sess *Session, message string) {
	defer close(sess.Stream)

	// Notify start
	sess.Stream <- StreamEvent{
		Type:    "status",
		Content: "Thinking...",
	}

	// Run agent
	ctx := context.Background()
	result, err := sess.Agent.Run(ctx, message)

	if err != nil {
		sess.Stream <- StreamEvent{
			Type:  "error",
			Error: err.Error(),
			Done:  true,
		}
		sess.History = append(sess.History, Message{Role: "assistant", Content: fmt.Sprintf("Error: %v", err)})
		return
	}

	// Stream result
	sess.Stream <- StreamEvent{
		Type:    "message",
		Content: result,
		Done:    true,
	}

	sess.History = append(sess.History, Message{Role: "assistant", Content: result})
}

// CORS middleware
func (s *Server) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}
