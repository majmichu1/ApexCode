package git

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// RepoManager handles git operations with auto-commit
// Based on aider's repo.py
type RepoManager struct {
	workDir    string
	repoRoot   string
	isGitRepo  bool
	commitHashes []string // Track aider commits for undo
}

// NewRepoManager creates a new git repo manager
func NewRepoManager(workDir string) *RepoManager {
	rm := &RepoManager{
		workDir: workDir,
	}

	// Find git root
	root, err := rm.runGit("rev-parse", "--show-toplevel")
	if err == nil {
		rm.repoRoot = strings.TrimSpace(root)
		rm.isGitRepo = true
	}

	return rm
}

// AutoCommit commits changes with LLM-generated message
func (rm *RepoManager) AutoCommit(files []string, prompt string) error {
	if !rm.isGitRepo {
		return nil // Not a git repo, skip
	}

	// Check for dirty files
	if len(files) > 0 {
		// Commit files that were modified before our edits
		dirtyFiles := rm.getDirtyFiles(files)
		if len(dirtyFiles) > 0 {
			if err := rm.commitFiles(dirtyFiles, fmt.Sprintf("Committing %s before applying edits", strings.Join(dirtyFiles, ", "))); err != nil {
				// Continue even if this fails
			}
		}
	}

	// Generate commit message using LLM
	// (In real implementation, would call LLM with diff context)
	commitMsg := rm.generateCommitMessage(files)

	// Add files
	for _, f := range files {
		rm.runGit("add", f)
	}

	// Commit
	if err := rm.commitFiles(files, commitMsg); err != nil {
		return err
	}

	return nil
}

// Undo undoes the last aider commit
func (rm *RepoManager) Undo() error {
	if len(rm.commitHashes) == 0 {
		return fmt.Errorf("no commits to undo")
	}

	lastHash := rm.commitHashes[len(rm.commitHashes)-1]
	rm.commitHashes = rm.commitHashes[:len(rm.commitHashes)-1]

	// Reset to previous commit
	_, err := rm.runGit("reset", "--hard", "HEAD~1")
	if err != nil {
		return fmt.Errorf("undo failed: %w", err)
	}

	// Restore files to previous state
	_, err = rm.runGit("checkout", "HEAD~1", "--", ".")
	return err
}

// GetDiff returns current diff
func (rm *RepoManager) GetDiff() (string, error) {
	return rm.runGit("diff")
}

// GetStatus returns git status
func (rm *RepoManager) GetStatus() (string, error) {
	return rm.runGit("status", "--short")
}

// GetLog returns recent commits
func (rm *RepoManager) GetLog(n int) (string, error) {
	return rm.runGit("log", fmt.Sprintf("--oneline -%d", n))
}

// GetCurrentBranch returns current branch name
func (rm *RepoManager) GetCurrentBranch() (string, error) {
	out, err := rm.runGit("rev-parse", "--abbrev-ref", "HEAD")
	return strings.TrimSpace(out), err
}

// IsClean returns true if no uncommitted changes
func (rm *RepoManager) IsClean() bool {
	out, err := rm.runGit("status", "--porcelain")
	return err == nil && strings.TrimSpace(out) == ""
}

// CommitFiles commits specific files
func (rm *RepoManager) commitFiles(files []string, message string) error {
	// Build commit command
	args := []string{"commit", "-m", message}
	
	// Add aider attribution
	args = append(args, "--author=aider <aider@aider.chat>")

	_, err := rm.runGit(args...)
	if err != nil {
		return err
	}

	// Track commit
	hash, err := rm.runGit("rev-parse", "HEAD")
	if err == nil {
		rm.commitHashes = append(rm.commitHashes, strings.TrimSpace(hash))
	}

	return nil
}

// getDirtyFiles finds files modified before our edits
func (rm *RepoManager) getDirtyFiles(files []string) []string {
	var dirty []string
	
	status, err := rm.runGit("status", "--porcelain")
	if err != nil {
		return dirty
	}

	for _, line := range strings.Split(status, "\n") {
		if line == "" {
			continue
		}
		// Parse status line (XY filename)
		parts := strings.SplitN(strings.TrimSpace(line), " ", 2)
		if len(parts) == 2 {
			filename := parts[1]
			// Check if this file is in our edit list
			for _, f := range files {
				if strings.HasSuffix(f, filename) {
					dirty = append(dirty, f)
					break
				}
			}
		}
	}

	return dirty
}

// generateCommitMessage creates a conventional commit message
func (rm *RepoManager) generateCommitMessage(files []string) string {
	// In production, would call LLM with diff to generate message
	// For now, use conventional commit format
	action := "update"
	if len(files) == 1 {
		action = "edit"
	}

	fileNames := make([]string, len(files))
	for i, f := range files {
		fileNames[i] = filepath.Base(f)
	}

	return fmt.Sprintf("feat: %s %s", action, strings.Join(fileNames, ", "))
}

// runGit executes a git command
func (rm *RepoManager) runGit(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = rm.workDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return string(output), fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
	}
	return string(output), nil
}

// Init initializes a git repo
func (rm *RepoManager) Init() error {
	if rm.isGitRepo {
		return nil
	}

	_, err := rm.runGit("init")
	if err == nil {
		rm.isGitRepo = true
		rm.repoRoot = rm.workDir
	}
	return err
}

// AddGitignore creates a .gitignore file
func (rm *RepoManager) AddGitignore(patterns []string) error {
	gitignorePath := filepath.Join(rm.workDir, ".gitignore")
	
	existing := ""
	if data, err := os.ReadFile(gitignorePath); err == nil {
		existing = string(data)
		if strings.HasSuffix(existing, "\n") {
			existing = existing[:len(existing)-1]
		}
	}

	var newContent strings.Builder
	newContent.WriteString(existing)
	
	for _, p := range patterns {
		if !strings.Contains(existing, p) {
			newContent.WriteString("\n" + p)
		}
	}

	return os.WriteFile(gitignorePath, []byte(newContent.String()), 0644)
}

// GetAiderignore returns the .aiderignore file patterns
func (rm *RepoManager) GetAiderignore() ([]string, error) {
	path := filepath.Join(rm.workDir, ".aiderignore")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var patterns []string
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "#") {
			patterns = append(patterns, line)
		}
	}
	return patterns, nil
}
