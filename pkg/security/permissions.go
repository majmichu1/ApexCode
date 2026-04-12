package security

import (
	"fmt"
	"strings"
)

// PermissionLevel defines access levels
type PermissionLevel int

const (
	// ReadOnly allows only reading files
	ReadOnly PermissionLevel = iota
	// ReadWrite allows reading and writing files
	ReadWrite
	// FullAccess allows all operations including bash execution
	FullAccess
)

// PermissionGate controls what operations are allowed
type PermissionGate struct {
	level          PermissionLevel
	allowedTools   map[string]bool
	blockedPaths   []string
	untrustedRepo  bool
}

// NewPermissionGate creates a new permission gate
func NewPermissionGate(untrustedRepo bool) *PermissionGate {
	pg := &PermissionGate{
		level:         ReadWrite,
		untrustedRepo: untrustedRepo,
		allowedTools:  make(map[string]bool),
		blockedPaths:  []string{"/etc", "/usr", "/root", "~/.ssh"},
	}

	// Default allowed tools
	pg.allowedTools["read_file"] = true
	pg.allowedTools["grep"] = true
	pg.allowedTools["glob"] = true
	pg.allowedTools["edit_file"] = true
	pg.allowedTools["write_file"] = true
	pg.allowedTools["web_fetch"] = true

	if !untrustedRepo {
		pg.allowedTools["bash"] = true
		pg.level = FullAccess
	} else {
		// Untrusted repo: block bash and file writes
		pg.allowedTools["bash"] = false
		pg.allowedTools["write_file"] = false
		pg.level = ReadOnly
	}

	return pg
}

// CheckTool verifies if a tool execution is allowed
func (pg *PermissionGate) CheckTool(toolName string) error {
	if allowed, exists := pg.allowedTools[toolName]; exists {
		if allowed {
			return nil
		}
		return fmt.Errorf("tool %s is not permitted in current security level", toolName)
	}
	return fmt.Errorf("unknown tool: %s", toolName)
}

// CheckPath verifies if a file path is accessible
func (pg *PermissionGate) CheckPath(path string) error {
	// Check blocked paths
	for _, blocked := range pg.blockedPaths {
		if strings.HasPrefix(path, blocked) {
			return fmt.Errorf("access denied: path %s is in blocked directory %s", path, blocked)
		}
	}

	return nil
}

// SetLevel changes the permission level
func (pg *PermissionGate) SetLevel(level PermissionLevel) {
	pg.level = level
	
	switch level {
	case ReadOnly:
		pg.allowedTools["bash"] = false
		pg.allowedTools["write_file"] = false
		pg.allowedTools["edit_file"] = false
	case ReadWrite:
		pg.allowedTools["bash"] = false
		pg.allowedTools["write_file"] = true
		pg.allowedTools["edit_file"] = true
	case FullAccess:
		for tool := range pg.allowedTools {
			pg.allowedTools[tool] = true
		}
	}
}

// ApproveAll grants full access (user approval)
func (pg *PermissionGate) ApproveAll() {
	pg.level = FullAccess
	for tool := range pg.allowedTools {
		pg.allowedTools[tool] = true
	}
}

// IsUntrusted returns true if the repo is marked as untrusted
func (pg *PermissionGate) IsUntrusted() bool {
	return pg.untrustedRepo
}

// GetSecurityReport returns a summary of current security settings
func (pg *PermissionGate) GetSecurityReport() string {
	var report strings.Builder
	
	report.WriteString("Security Level: ")
	switch pg.level {
	case ReadOnly:
		report.WriteString("READ ONLY")
	case ReadWrite:
		report.WriteString("READ/WRITE")
	case FullAccess:
		report.WriteString("FULL ACCESS")
	}
	
	if pg.untrustedRepo {
		report.WriteString("\n⚠️  Repository marked as untrusted\n")
	}

	report.WriteString("\nAllowed tools:\n")
	for tool, allowed := range pg.allowedTools {
		if allowed {
			report.WriteString(fmt.Sprintf("  ✅ %s\n", tool))
		}
	}

	return report.String()
}
