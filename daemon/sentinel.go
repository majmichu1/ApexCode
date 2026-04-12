package daemon

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/apexcode/apexcode/internal/memory"
	"github.com/apexcode/apexcode/internal/knowledge"
)

// Sentinel represents the background service
// Equivalent to KAIROS from Claude Code but with different code/naming
type Sentinel struct {
	memory       *memory.MemPalace
	knowledgeBase *knowledge.KnowledgeBase
	workDir      string
	logFile      string
	running      bool
	tickInterval time.Duration
	briefMode    bool
	maxTickBudget time.Duration // Max 15 seconds blocking per tick (like Claude Code)
	idleCallback func()         // Callback for idle actions
}

// NewSentinel creates a new background daemon
func NewSentinel(workDir string, mem *memory.MemPalace, kb *knowledge.KnowledgeBase) *Sentinel {
	logFile := filepath.Join(workDir, ".apexcode", "sentinel.log")

	return &Sentinel{
		memory:       mem,
		knowledgeBase: kb,
		workDir:      workDir,
		logFile:      logFile,
		tickInterval: 30 * time.Second,
		briefMode:    false,
		maxTickBudget: 15 * time.Second,
	}
}

// Start begins the sentinel loop
func (s *Sentinel) Start() error {
	if s.running {
		return fmt.Errorf("sentinel already running")
	}

	s.running = true

	// Create log file
	if err := os.MkdirAll(filepath.Dir(s.logFile), 0755); err != nil {
		return fmt.Errorf("creating log directory: %w", err)
	}

	logFile, err := os.OpenFile(s.logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("opening log file: %w", err)
	}
	defer logFile.Close()

	log.SetOutput(logFile)
	log.Println("=== Sentinel service started ===")

	// Main tick loop
	ticker := time.NewTicker(s.tickInterval)
	defer ticker.Stop()

	for s.running {
		select {
		case <-ticker.C:
			s.tick()
		}
	}

	log.Println("=== Sentinel service stopped ===")
	return nil
}

// Stop stops the sentinel
func (s *Sentinel) Stop() {
	s.running = false
}

// tick performs periodic work with budget enforcement
func (s *Sentinel) tick() {
	startTime := time.Now()
	log.Printf("[TICK] %s", time.Now().Format(time.RFC3339))

	// Check budget
	if time.Since(startTime) > s.maxTickBudget {
		log.Printf("[BUDGET] Exceeded %v budget, skipping remaining tasks", s.maxTickBudget)
		return
	}

	// Memory consolidation
	if err := s.consolidateMemory(); err != nil {
		log.Printf("[ERROR] Memory consolidation: %v", err)
	}

	// Process knowledge base inbox
	if s.knowledgeBase != nil {
		if err := s.processInbox(); err != nil {
			log.Printf("[ERROR] Inbox processing: %v", err)
		}
	}

	// Idle detection and action
	if s.isIdle() {
		s.performIdleAction()
	}
}

// consolidateMemory performs memory cleanup and optimization
func (s *Sentinel) consolidateMemory() error {
	if s.memory == nil {
		return nil
	}

	// Add diary entry about current state
	diaryEntry := fmt.Sprintf("Sentinel tick at %s - system healthy", 
		time.Now().Format(time.RFC3339))
	
	if err := s.memory.AddDiaryEntry(diaryEntry, "system"); err != nil {
		return err
	}

	// In production, would also:
	// - Merge duplicate drawers
	// - Update importance scores based on access patterns
	// - Clean old entries
	// - Apply AAAK compression to rarely-accessed drawers
	// - Update knowledge graph with new facts

	return nil
}

// processInbox processes items from the Karpathy knowledge base inbox
func (s *Sentinel) processInbox() error {
	if s.knowledgeBase == nil {
		return nil
	}

	// Check if inbox has items
	items, err := s.knowledgeBase.ListInbox()
	if err != nil {
		return err
	}

	if len(items) == 0 {
		return nil
	}

	log.Printf("[INBOX] %d items to process", len(items))

	// In production, would use LLM to:
	// 1. Read each inbox item
	// 2. Determine appropriate category/tags
	// 3. Move to appropriate folder
	// 4. Create backlinks to related notes
	// 5. Update knowledge graph

	for _, item := range items {
		log.Printf("[INBOX] Processing: %s", item)
		// Would call LLM to triage and file
	}

	return nil
}

// isIdle checks if the system is idle
func (s *Sentinel) isIdle() bool {
	// Check if there's been recent activity
	// In production, would check:
	// - Last user input
	// - File system changes
	// - Terminal activity
	// - Running processes
	return true
}

// performIdleAction performs actions during idle periods
func (s *Sentinel) performIdleAction() {
	log.Println("[IDLE] Performing idle tasks")

	// Memory consolidation
	if s.memory != nil {
		if err := s.consolidateMemory(); err != nil {
			log.Printf("[IDLE ERROR] %v", err)
		}
	}

	// Knowledge base processing
	if s.knowledgeBase != nil {
		if err := s.processInbox(); err != nil {
			log.Printf("[IDLE ERROR] %v", err)
		}
	}

	// Proactive suggestions - analyze patterns and suggest helpful actions
	if s.idleCallback != nil {
		s.idleCallback()
	}
}

// AnalyzePatterns scans recent activity for proactive suggestions
func (s *Sentinel) AnalyzePatterns() []ProactiveSuggestion {
	var suggestions []ProactiveSuggestion

	// Check for common patterns:
	
	// 1. Repeated manual test runs → suggest test watcher
	// 2. Frequent file edits without commits → suggest auto-commit
	// 3. Same error appearing multiple times → suggest fix pattern
	// 4. Long idle time with uncommitted changes → suggest commit reminder
	// 5. Files modified but not added to chat → suggest context expansion

	// These would be powered by:
	// - Git log analysis (uncommitted changes, commit frequency)
	// - File modification timestamps
	// - Session history from MemPalace
	// - Error frequency from tool results

	return suggestions
}

// ProactiveSuggestion represents a proactive suggestion from the Sentinel
type ProactiveSuggestion struct {
	Message    string   // What to suggest
	Type       string   // "reminder", "optimization", "automation", "insight"
	Priority   int      // 1-10, how urgent the suggestion is
	CanAutoFix bool     // Whether Sentinel can auto-apply the fix
	Action     string   // What action would be taken if accepted
}

// GetStatus returns sentinel status
func (s *Sentinel) GetStatus() string {
	status := "stopped"
	if s.running {
		status = "running"
	}

	var kbStatus string
	if s.knowledgeBase != nil {
		inbox, _ := s.knowledgeBase.ListInbox()
		kbStatus = fmt.Sprintf("%d items in inbox", len(inbox))
	}

	return fmt.Sprintf(
		"Sentinel: %s\n"+
		"  Tick interval: %v\n"+
		"  Log: %s\n"+
		"  Knowledge base: %s",
		status, s.tickInterval, s.logFile, kbStatus,
	)
}

// SetBriefMode enables/disables brief output
func (s *Sentinel) SetBriefMode(brief bool) {
	s.briefMode = brief
}

// SetIdleCallback sets the idle callback
func (s *Sentinel) SetIdleCallback(fn func()) {
	s.idleCallback = fn
}

// GetLog returns recent log entries
func (s *Sentinel) GetLog(lines int) (string, error) {
	data, err := os.ReadFile(s.logFile)
	if err != nil {
		return "", err
	}

	allLines := strings.Split(string(data), "\n")
	start := len(allLines) - lines
	if start < 0 {
		start = 0
	}

	return strings.Join(allLines[start:], "\n"), nil
}
