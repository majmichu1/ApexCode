package tui

import "github.com/charmbracelet/lipgloss"

// Theme defines the color scheme for the TUI
type Theme struct {
	Background   lipgloss.TerminalColor
	Foreground   lipgloss.TerminalColor
	Primary      lipgloss.TerminalColor
	Secondary    lipgloss.TerminalColor
	Success      lipgloss.TerminalColor
	Warning      lipgloss.TerminalColor
	Error        lipgloss.TerminalColor
	Subtle       lipgloss.TerminalColor
	Border       lipgloss.TerminalColor
	Selected     lipgloss.TerminalColor
	UserMsg      lipgloss.TerminalColor
	AssistantMsg lipgloss.TerminalColor
	ToolMsg      lipgloss.TerminalColor
}

// DefaultTheme returns the default dark theme
func DefaultTheme() Theme {
	return Theme{
		Background:   lipgloss.Color("#1a1b26"),
		Foreground:   lipgloss.Color("#a9b1d6"),
		Primary:      lipgloss.Color("#7aa2f7"),
		Secondary:    lipgloss.Color("#bb9af7"),
		Success:      lipgloss.Color("#9ece6a"),
		Warning:      lipgloss.Color("#e0af68"),
		Error:        lipgloss.Color("#f7768e"),
		Subtle:       lipgloss.Color("#565f89"),
		Border:       lipgloss.Color("#3b4261"),
		Selected:     lipgloss.Color("#7dcfff"),
		UserMsg:      lipgloss.Color("#7aa2f7"),
		AssistantMsg: lipgloss.Color("#a9b1d6"),
		ToolMsg:      lipgloss.Color("#bb9af7"),
	}
}
