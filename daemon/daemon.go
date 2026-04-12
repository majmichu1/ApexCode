package daemon

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/apexcode/apexcode/internal/memory"
)

// Daemon represents the background service
// Similar to KAIROS from Claude Code
type Daemon struct {
	memory     *memory.MemPalace
	workDir    string
	logFile    string
	running    bool
	tickInterval time.Duration
	briefMode  bool
}

// NewDaemon creates a new background daemon
func NewDaemon(workDir string, mem *memory.MemPalace) *Daemon {
	logFile := filepath.Join(workDir, ".apexcode", "daemon.log")

	return &Daemon{
		memory:       mem,
		workDir:      workDir,
		logFile:      logFile,
		tickInterval: 30 * time.Second,
		briefMode:    false,
	}
}

// Start begins the daemon loop
func (d *Daemon) Start() error {
	if d.running {
		return fmt.Errorf("daemon already running")
	}

	d.running = true

	// Create log file
	if err := os.MkdirAll(filepath.Dir(d.logFile), 0755); err != nil {
		return fmt.Errorf("creating log directory: %w", err)
	}

	logFile, err := os.OpenFile(d.logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("opening log file: %w", err)
	}

	log.SetOutput(logFile)
	log.Println("=== Daemon started ===")

	// Main tick loop
	ticker := time.NewTicker(d.tickInterval)
	defer ticker.Stop()

	for d.running {
		select {
		case <-ticker.C:
			d.tick()
		}
	}

	log.Println("=== Daemon stopped ===")
	return nil
}

// Stop stops the daemon
func (d *Daemon) Stop() {
	d.running = false
}

// tick performs periodic work
func (d *Daemon) tick() {
	log.Printf("[TICK] %s", time.Now().Format(time.RFC3339))

	// Memory consolidation
	if err := d.consolidateMemory(); err != nil {
		log.Printf("[ERROR] Memory consolidation failed: %v", err)
	}

	// Check for idle state
	if d.isIdle() {
		d.idleAction()
	}
}

// consolidateMemory performs memory cleanup and optimization
func (d *Daemon) consolidateMemory() error {
	if d.memory == nil {
		return nil
	}

	// Add diary entry about current state
	diaryEntry := fmt.Sprintf("Daemon tick at %s - system healthy", time.Now().Format(time.RFC1502))
	
	if err := d.memory.AddDiaryEntry(diaryEntry, "system"); err != nil {
		return err
	}

	// In production, would also:
	// - Merge duplicate drawers
	// - Update importance scores
	// - Clean old entries
	// - Compress with AAAK

	return nil
}

// isIdle checks if the system is idle
func (d *Daemon) isIdle() bool {
	// Check if there's been recent activity
	// In production, would check file system changes, user input, etc.
	return true
}

// idleAction performs actions during idle periods
func (d *Daemon) idleAction() {
	log.Println("[IDLE] Performing idle tasks")

	// Auto memory consolidation
	if d.memory != nil {
		if err := d.consolidateMemory(); err != nil {
			log.Printf("[IDLE ERROR] %v", err)
		}
	}

	// Could also:
	// - Pre-fetch context for likely next actions
	// - Update repo map
	// - Run background analysis
}

// SetBriefMode enables/disables brief output mode
func (d *Daemon) SetBriefMode(brief bool) {
	d.briefMode = brief
}

// GetStatus returns daemon status
func (d *Daemon) GetStatus() string {
	status := "running"
	if !d.running {
		status = "stopped"
	}

	return fmt.Sprintf("Daemon: %s | Tick interval: %v | Log: %s",
		status, d.tickInterval, d.logFile)
}
