package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// OpenConnectViewMsg is sent to open the connect (MCP) instructions screen.
type OpenConnectViewMsg struct{}

// ConnectBackMsg is sent when the user leaves the connect view.
type ConnectBackMsg struct{}

// ConnectModel is the MCP connect-instructions screen.
type ConnectModel struct {
	styles    *Styles
	keys      ConnectKeyMap
	contxPath string
	width     int
	height    int
	scroll    int
	copied    bool // flash after copy
}

// ConnectKeyMap defines keybindings for the connect view.
type ConnectKeyMap struct {
	Back         key.Binding
	CopyCursor   key.Binding
	CopyClaude   key.Binding
	CopyRegister key.Binding
}

var DefaultConnectKeys = ConnectKeyMap{
	Back: key.NewBinding(
		key.WithKeys("esc", "q"),
		key.WithHelp("esc", "back"),
	),
	CopyCursor: key.NewBinding(
		key.WithKeys("1"),
		key.WithHelp("1", "copy Cursor config"),
	),
	CopyClaude: key.NewBinding(
		key.WithKeys("2"),
		key.WithHelp("2", "copy Claude command"),
	),
	CopyRegister: key.NewBinding(
		key.WithKeys("3"),
		key.WithHelp("3", "copy user-scoped command"),
	),
}

// contxBinaryPath returns the full path to the contx binary for use in commands and config.
func contxBinaryPath() string {
	exe, err := os.Executable()
	if err != nil {
		return "contx"
	}
	abs, err := filepath.Abs(exe)
	if err != nil {
		return exe
	}
	resolved, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return abs
	}
	return resolved
}

// cursorMCPConfigPath returns the path to Cursor's MCP config file for the platform.
func cursorMCPConfigPath() string {
	home, _ := os.UserHomeDir()
	switch runtime.GOOS {
	case "windows":
		return filepath.Join(os.Getenv("APPDATA"), "Cursor", "mcp.json")
	case "darwin":
		return filepath.Join(home, ".cursor", "mcp.json")
	default:
		return filepath.Join(home, ".config", "cursor", "mcp.json")
	}
}

func NewConnect(styles *Styles, w, h int) ConnectModel {
	return ConnectModel{
		styles:    styles,
		keys:      DefaultConnectKeys,
		contxPath: contxBinaryPath(),
		width:     w,
		height:    h,
	}
}

func (m ConnectModel) Init() tea.Cmd { return nil }

func (m ConnectModel) Update(msg tea.Msg) (ConnectModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keys.Back):
			return m, func() tea.Msg { return ConnectBackMsg{} }
		case key.Matches(msg, m.keys.CopyCursor):
			cfg := m.cursorConfigSnippet()
			return m, copyToClipboard(cfg)
		case key.Matches(msg, m.keys.CopyClaude):
			cmd := m.claudeCommand()
			return m, copyToClipboard(cmd)
		case key.Matches(msg, m.keys.CopyRegister):
			cmd := m.claudeUserScopedCommand()
			return m, copyToClipboard(cmd)
		case msg.String() == "up" || msg.String() == "k":
			if m.scroll > 0 {
				m.scroll--
			}
			return m, nil
		case msg.String() == "down" || msg.String() == "j":
			m.scroll++
			return m, nil
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case copiedMsg:
		m.copied = true
		return m, tea.Tick(2*time.Second, func(time.Time) tea.Msg { return clearCopiedMsg{} })
	case clearCopiedMsg:
		m.copied = false
		return m, nil
	}
	return m, nil
}

func (m ConnectModel) cursorConfigSnippet() string {
	return fmt.Sprintf(`"contx": {
  "command": "%s",
  "args": ["serve"]
}`, strings.ReplaceAll(m.contxPath, `\`, `\\`))
}

func (m ConnectModel) claudeCommand() string {
	return fmt.Sprintf("claude mcp add contx %s serve", m.contxPath)
}

func (m ConnectModel) claudeUserScopedCommand() string {
	return fmt.Sprintf("claude mcp add --scope user contx %s serve", m.contxPath)
}

func (m ConnectModel) View() string {
	s := m.styles
	w := m.width
	if w < 20 {
		w = 20
	}
	h := m.height
	if h < 10 {
		h = 10
	}

	// Content width with padding
	contentW := w - 4
	if contentW < 40 {
		contentW = 40
	}

	title := s.AppTitle.Render("Connect MCP")
	subtitle := s.PreviewMeta.Render("Add contx to Cursor, Claude Code, or other MCP clients")
	header := lipgloss.NewStyle().Padding(0, 1).Render(title + "\n" + subtitle)

	cursorPath := cursorMCPConfigPath()
	cursorInstructions := []string{
		"",
		"◆ Cursor",
		"",
		"1. Open or create the MCP config file:",
		"   " + cursorPath,
		"",
		"2. Add contx under mcpServers. If the file is empty or doesn't have mcpServers yet, use the full JSON below. Otherwise merge the contx entry into your existing mcpServers object.",
		"",
		"   Snippet to add (key 1 to copy):",
		"",
		m.cursorConfigSnippet(),
		"",
		"3. Restart Cursor. Then you can @-mention contexts: contx://global/<name> or contx://project/<name>.",
		"",
		"───",
		"",
		"◆ Claude Code (Claude desktop / API)",
		"",
		"Run in terminal (key 2 to copy project-scoped, key 3 for user-scoped):",
		"",
		"  " + m.claudeCommand(),
		"",
		"  User-scoped (all projects):",
		"  " + m.claudeUserScopedCommand(),
		"",
		"Then restart Claude Code and use @contx:// in conversations.",
		"",
	}

	body := strings.Join(cursorInstructions, "\n")
	lines := strings.Split(body, "\n")
	contentH := h - 6
	if contentH < 1 {
		contentH = 1
	}
	maxScroll := len(lines) - contentH
	if maxScroll < 0 {
		maxScroll = 0
	}
	scroll := m.scroll
	if scroll > maxScroll {
		scroll = maxScroll
	}
	start := scroll
	end := start + contentH
	if end > len(lines) {
		end = len(lines)
	}
	visible := lines[start:end]
	for i, line := range visible {
		visible[i] = ansi.Truncate(line, contentW, "")
	}
	bodyRendered := strings.Join(visible, "\n")

	bodyBlock := lipgloss.NewStyle().
		Width(contentW).
		Padding(0, 1).
		Foreground(s.Theme.Text).
		Render(bodyRendered)

	// Footer
	copyHint := "1 Cursor  2 Claude  3 user-scoped"
	if m.copied {
		copyHint = lipgloss.NewStyle().Foreground(s.Theme.Green).Render("✓ copied!")
	}
	footer := s.Footer.Width(w).Render("  " + s.HintKey.Render("esc") + " " + s.HintDesc.Render("back") + "   " + copyHint)

	view := lipgloss.JoinVertical(lipgloss.Left, header, "", bodyBlock, "", footer)
	return limitLineCount(view, h, w)
}
