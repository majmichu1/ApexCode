package main

import (
	"fmt"
	"os"

	"github.com/apexcode/apexcode/internal/config"
	"github.com/apexcode/apexcode/internal/agent"
	"github.com/apexcode/apexcode/internal/knowledge"
	"github.com/apexcode/apexcode/internal/server"
	"github.com/apexcode/apexcode/tui"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	// Check for command flags
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "--version", "-v":
			fmt.Println("ApexCode v0.1.0")
			return
		case "--help", "-h":
			printUsage()
			return
		case "--init":
			if err := config.InitProject(); err != nil {
				fmt.Fprintf(os.Stderr, "Error initializing project: %v\n", err)
				os.Exit(1)
			}
			// Initialize knowledge base
			if err := initKnowledgeBase(cfg.WorkDir); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to init knowledge base: %v\n", err)
			}
			return
		case "--serve":
			// Start HTTP server for TUI
			srv := server.NewServer(cfg, 7777)
			fmt.Println("🚀 ApexCode server starting on http://localhost:7777")
			fmt.Println("   Point the ApexCode TUI to this URL")
			if err := srv.Start(); err != nil {
				fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
				os.Exit(1)
			}
			return
		}
	}

	// Start the TUI
	app := tui.NewApp(cfg)
	
	// Initialize the agent
	ag := agent.New(cfg)
	
	// Run the application
	if err := app.Run(ag); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// initKnowledgeBase creates the Karpathy-style knowledge base
func initKnowledgeBase(workDir string) error {
	kbDir := fmt.Sprintf("%s/.apexcode/knowledge", workDir)
	
	_, err := knowledge.NewKnowledgeBase(kbDir)
	if err != nil {
		return err
	}

	fmt.Println("✓ Knowledge base initialized at .apexcode/knowledge/")
	fmt.Println("  Structure:")
	fmt.Println("    - inbox/          Drop raw notes here")
	fmt.Println("    - projects/       Project documentation")
	fmt.Println("    - research/       Research notes")
	fmt.Println("    - reference/      Reference materials")
	fmt.Println("    - meetings/       Meeting notes")
	fmt.Println("    - _templates/     Note templates")
	
	return nil
}

func printUsage() {
	fmt.Println(`ApexCode - The Ultimate AI Coding Agent

Usage:
  apex [command] [flags]

Commands:
  --init        Initialize ApexCode in current project
  --version     Show version information
  --help        Show this help message

Examples:
  apex                    Start interactive mode
  apex --init             Create APEX.md config file and knowledge base
  apex "fix the bug"      Run a single task

Knowledge Base (Karpathy Workflow):
  Add raw notes to .apexcode/knowledge/inbox/
  The Sentinel daemon will auto-triage and file them
  Query with: "search my knowledge base for X"

For more information, visit: https://github.com/apexcode/apexcode`)
}
