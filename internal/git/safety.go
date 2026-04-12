package git

import (
	"fmt"
	"os/exec"
	"strings"
)

// SafetyManager provides Git safety operations
type SafetyManager struct {
	workDir string
	enabled bool
}

// NewSafetyManager creates a new Git safety manager
func NewSafetyManager(workDir string, enabled bool) *SafetyManager {
	return &SafetyManager{
		workDir: workDir,
		enabled: enabled,
	}
}

// AutoStash creates a stash before risky operations
func (g *SafetyManager) AutoStash() error {
	if !g.enabled {
		return nil
	}

	// Check if there are uncommitted changes
	output, err := g.runGit("status", "--porcelain")
	if err != nil {
		return fmt.Errorf("checking git status: %w", err)
	}

	if strings.TrimSpace(output) != "" {
		// Stash changes
		_, err = g.runGit("stash", "push", "-m", "apexcode-auto-stash")
		if err != nil {
			return fmt.Errorf("auto-stash failed: %w", err)
		}
	}

	return nil
}

// PopStash restores the most recent stash
func (g *SafetyManager) PopStash() error {
	if !g.enabled {
		return nil
	}

	_, err := g.runGit("stash", "pop")
	return err
}

// CreateBranch creates a new branch for changes
func (g *SafetyManager) CreateBranch(branchName string) error {
	if !g.enabled {
		return nil
	}

	_, err := g.runGit("checkout", "-b", branchName)
	return err
}

// PreCommitCheck runs checks before committing
func (g *SafetyManager) PreCommitCheck() error {
	if !g.enabled {
		return nil
	}

	// Get status
	status, err := g.runGit("status", "--porcelain")
	if err != nil {
		return fmt.Errorf("getting git status: %w", err)
	}

	// Get diff
	diff, err := g.runGit("diff", "--stat")
	if err != nil {
		return fmt.Errorf("getting git diff: %w", err)
	}

	fmt.Printf("Git Status:\n%s\n\nGit Diff:\n%s\n", status, diff)
	return nil
}

// SafeCommit performs a safe commit with checks
func (g *SafetyManager) SafeCommit(message string) error {
	if !g.enabled {
		return nil
	}

	// Run pre-commit checks
	if err := g.PreCommitCheck(); err != nil {
		return fmt.Errorf("pre-commit checks failed: %w", err)
	}

	// Add all changes
	_, err := g.runGit("add", "-A")
	if err != nil {
		return fmt.Errorf("adding files: %w", err)
	}

	// Commit
	_, err = g.runGit("commit", "-m", message)
	return err
}

// SafePush performs a safe push with warnings
func (g *SafetyManager) SafePush(branch string, force bool) error {
	if !g.enabled {
		return nil
	}

	if force {
		fmt.Println("⚠️  WARNING: Force push detected! This can overwrite remote history.")
		fmt.Println("Proceed with caution. Consider using regular push instead.")
	}

	args := []string{"push"}
	if force {
		args = append(args, "--force")
	}
	if branch != "" {
		args = append(args, "origin", branch)
	}

	_, err := g.runGit(args...)
	return err
}

// GetCurrentBranch returns the current branch name
func (g *SafetyManager) GetCurrentBranch() (string, error) {
	output, err := g.runGit("rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(output), nil
}

// IsClean returns true if there are no uncommitted changes
func (g *SafetyManager) IsClean() (bool, error) {
	output, err := g.runGit("status", "--porcelain")
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(output) == "", nil
}

// GetDiff returns the current diff
func (g *SafetyManager) GetDiff() (string, error) {
	return g.runGit("diff")
}

// GetLog returns recent commits
func (g *SafetyManager) GetLog(n int) (string, error) {
	return g.runGit("log", fmt.Sprintf("--oneline -%d", n))
}

// runGit executes a git command
func (g *SafetyManager) runGit(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = g.workDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return string(output), fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
	}
	return string(output), nil
}
