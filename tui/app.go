package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/apexcode/apexcode/internal/agent"
	"github.com/apexcode/apexcode/internal/config"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type Mode int

const (
	PlanMode Mode = iota
	BuildMode
)

type App struct {
	cfg       *config.Config
	agent     *agent.Agent
	theme     Theme
	mode      Mode
	input     string
	messages  []UIMessage
	width     int
	height    int
	quitting  bool
	showHelp  bool
	streaming bool
	statusMsg string
	scrollPos int
}

type UIMessage struct {
	Role    string
	Content string
	Time    time.Time
}

type streamChunk struct{ text string }
type agentDone struct{ result string }
type agentErr struct{ err error }

func NewApp(cfg *config.Config) *App {
	return &App{
		cfg:   cfg,
		theme: DefaultTheme(),
		mode:  BuildMode,
	}
}

func (a *App) Run(ag *agent.Agent) error {
	a.agent = ag
	p := tea.NewProgram(a, tea.WithAltScreen(), tea.WithMouseCellMotion())
	_, err := p.Run()
	return err
}

func (a *App) Init() tea.Cmd {
	a.statusMsg = "Ready — ask me anything"
	return nil
}

func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		return a, nil

	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyCtrlQ:
			a.quitting = true
			return a, tea.Quit
		case tea.KeyEnter:
			if a.input != "" && !a.streaming {
				input := a.input
				a.input = ""
				a.messages = append(a.messages, UIMessage{
					Role:    "user",
					Content: input,
					Time:    time.Now(),
				})
				a.streaming = true
				a.statusMsg = "⏳ Thinking..."
				a.messages = append(a.messages, UIMessage{
					Role:    "assistant",
					Content: "",
					Time:    time.Now(),
				})
				return a, a.runAgent(input)
			}
			return a, nil
		case tea.KeyBackspace:
			if len(a.input) > 0 {
				a.input = a.input[:len(a.input)-1]
			}
			return a, nil
		case tea.KeyTab:
			if a.mode == PlanMode {
				a.mode = BuildMode
				a.statusMsg = "🔧 Build mode — AI can read/write files"
			} else {
				a.mode = PlanMode
				a.statusMsg = "👁️ Plan mode — read-only analysis"
			}
			return a, nil
		default:
			if len(msg.String()) == 1 && !a.streaming {
				a.input += msg.String()
			}
			if msg.String() == "?" {
				a.showHelp = !a.showHelp
			}
			return a, nil
		}

	case streamChunk:
		if len(a.messages) > 0 {
			last := &a.messages[len(a.messages)-1]
			if last.Role == "assistant" {
				last.Content += msg.text
			}
		}
		return a, nil

	case agentDone:
		a.streaming = false
		a.statusMsg = fmt.Sprintf("✅ Done — %d turns", a.agent.GetStats()["turn_count"])
		return a, nil

	case agentErr:
		a.streaming = false
		a.statusMsg = fmt.Sprintf("❌ Error: %v", msg.err)
		return a, nil
	}

	return a, nil
}

func (a *App) View() string {
	if a.quitting {
		return "Goodbye!\n"
	}

	var b strings.Builder

	// Status bar (top)
	b.WriteString(a.statusBar())
	b.WriteString("\n")

	// Messages area
	msgHeight := a.height - 5
	b.WriteString(a.renderMessages(msgHeight))

	// Input bar (bottom)
	b.WriteString(a.inputBar())

	return b.String()
}

func (a *App) statusBar() string {
	modeStr := " BUILD "
	if a.mode == PlanMode {
		modeStr = " PLAN  "
	}

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(a.theme.Primary).
		Background(lipgloss.Color("#1a1b26")).
		Padding(0, 1).
		Render("ApexCode")

	modeStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#1a1b26")).
		Background(a.modeColor()).
		Padding(0, 1).
		Render(modeStr)

	statusStyle := lipgloss.NewStyle().
		Foreground(a.theme.Subtle).
		Render(" " + a.statusMsg)

	rightStyle := lipgloss.NewStyle().
		Foreground(a.theme.Subtle).
		Render(fmt.Sprintf("%s | ? help ", a.cfg.Provider))

	return lipgloss.NewStyle().
		Width(a.width).
		Background(lipgloss.Color("#1a1b26")).
		Render(titleStyle + " " + modeStyle + statusStyle + " " + rightStyle)
}

func (a *App) modeColor() lipgloss.TerminalColor {
	if a.mode == PlanMode {
		return a.theme.Warning
	}
	return a.theme.Success
}

func (a *App) renderMessages(height int) string {
	if len(a.messages) == 0 {
		return a.welcomeScreen()
	}

	var lines []string
	for _, msg := range a.messages {
		switch msg.Role {
		case "user":
			userStyle := lipgloss.NewStyle().
				Foreground(a.theme.UserMsg).
				Bold(true).
				MarginLeft(2).
				MarginTop(1).
				Render("❯ " + msg.Content)
			lines = append(lines, userStyle)

		case "assistant":
			content := msg.Content
			if a.streaming && content == "" {
				content = "▊"
			} else if a.streaming {
				content += "▊"
			}
			
			// Word wrap
			wrapped := wrapText(content, a.width-4)
			assistantStyle := lipgloss.NewStyle().
				Foreground(a.theme.AssistantMsg).
				MarginLeft(2).
				Render(wrapped)
			lines = append(lines, assistantStyle)
		}
	}

	result := strings.Join(lines, "\n")
	resultLines := strings.Split(result, "\n")

	// Auto-scroll to bottom
	if len(resultLines) > height {
		resultLines = resultLines[len(resultLines)-height:]
	}

	return strings.Join(resultLines, "\n")
}

func (a *App) welcomeScreen() string {
	logo := lipgloss.NewStyle().
		Bold(true).
		Foreground(a.theme.Primary).
		Render("  ╔═╗╔╦╗╔═╗╔═╗  ╔═╗╔╦╗╔═╗\n  ╠╣  ║║║╣ ╠═╝  ╚═╗ ║ ║╣ \n  ╚═╝╚═╝╚═╝╩    ╚═╝ ╩ ╚═╝")

	subtitle := lipgloss.NewStyle().
		Foreground(a.theme.Subtle).
		Render("  The Ultimate AI Coding Agent\n")

	commands := lipgloss.NewStyle().
		Foreground(a.theme.Secondary).
		Render("\n  🔧 Build mode  — AI can read/write files, execute commands\n" +
			"  👁️ Plan mode   — Read-only analysis\n" +
			"  Tab           — Toggle mode\n" +
			"  Ctrl+C        — Quit\n")

	prompt := lipgloss.NewStyle().
		Foreground(a.theme.Subtle).
		Render("\n  Ask me to write code, fix bugs, explain files, or refactor.\n")

	return "\n" + logo + "\n" + subtitle + commands + prompt
}

func (a *App) inputBar() string {
	var prompt string
	if a.streaming {
		prompt = lipgloss.NewStyle().
			Foreground(a.theme.Warning).
			Render(" ⏳ Processing...")
	} else {
		cursor := ""
		if true {
			cursor = "█"
		}
		prompt = lipgloss.NewStyle().
			Foreground(a.theme.Primary).
			Render(" ❯ ") + a.input + cursor
	}

	return lipgloss.NewStyle().
		Width(a.width).
		Border(lipgloss.NormalBorder(), true, false, false, false).
		BorderForeground(a.theme.Border).
		Padding(0, 1).
		Render(prompt)
}

func (a *App) runAgent(input string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		
		// Simulate streaming by using callback
		var collected string
		a.agent.SetStreamCallback(func(text string) {
			collected += text
		})

		result, err := a.agent.Run(ctx, input)
		if err != nil {
			return agentErr{err}
		}

		return agentDone{result: result}
	}
}

// wrapText wraps text to the given width
func wrapText(text string, width int) string {
	if width <= 0 {
		return text
	}

	var result strings.Builder
	var lineLen int

	for _, r := range text {
		if r == '\n' {
			result.WriteRune(r)
			lineLen = 0
			continue
		}

		if lineLen >= width {
			result.WriteString("\n")
			lineLen = 0
		}

		result.WriteRune(r)
		lineLen++
	}

	return result.String()
}
