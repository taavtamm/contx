package tui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/taavtamm/contx/internal/store"
)

// ConfirmedMsg is sent when the user confirms deletion.
type ConfirmedMsg struct {
	Name  string
	Scope store.Scope
}

// ConfirmCancelledMsg is sent when the user cancels the confirm dialog.
type ConfirmCancelledMsg struct{}

// ConfirmModel is the delete confirmation dialog.
type ConfirmModel struct {
	styles      *Styles
	keys        ConfirmKeyMap
	contextName string
	scope       store.Scope
	focusDelete bool
	width       int
	height      int
}

func NewConfirm(styles *Styles, contextName string, scope store.Scope, w, h int) ConfirmModel {
	return ConfirmModel{
		styles:      styles,
		keys:        DefaultConfirmKeys,
		contextName: contextName,
		scope:       scope,
		focusDelete: false,
		width:       w,
		height:      h,
	}
}

func (m ConfirmModel) Init() tea.Cmd { return nil }

func (m ConfirmModel) Update(msg tea.Msg) (ConfirmModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keys.No):
			return m, func() tea.Msg { return ConfirmCancelledMsg{} }
		case key.Matches(msg, m.keys.Yes):
			if m.focusDelete {
				name, scope := m.contextName, m.scope
				return m, func() tea.Msg { return ConfirmedMsg{Name: name, Scope: scope} }
			}
			return m, func() tea.Msg { return ConfirmCancelledMsg{} }
		case key.Matches(msg, m.keys.Left):
			m.focusDelete = true
		case key.Matches(msg, m.keys.Right):
			m.focusDelete = false
		case msg.String() == "enter":
			if m.focusDelete {
				name, scope := m.contextName, m.scope
				return m, func() tea.Msg { return ConfirmedMsg{Name: name, Scope: scope} }
			}
			return m, func() tea.Msg { return ConfirmCancelledMsg{} }
		}
	}
	return m, nil
}

func (m ConfirmModel) View() string {
	s := m.styles

	title := s.DialogTitle.Render(fmt.Sprintf(`Delete "%s"?`, m.contextName))
	body := s.DialogBody.Render("This cannot be undone.")

	var btnDelete, btnCancel string
	if m.focusDelete {
		btnDelete = s.BtnDelete.Render("Yes, delete")
		btnCancel = s.BtnCancel.Render("Cancel")
	} else {
		btnDelete = lipgloss.NewStyle().
			Background(s.Theme.Surface).
			Foreground(s.Theme.Red).
			Padding(0, 2).
			Render("Yes, delete")
		btnCancel = lipgloss.NewStyle().
			Foreground(s.Theme.Text).
			Bold(true).
			Padding(0, 2).
			Render("Cancel")
	}

	buttons := lipgloss.JoinHorizontal(lipgloss.Center, btnDelete, "  ", btnCancel)

	inner := lipgloss.JoinVertical(lipgloss.Center,
		title,
		"",
		body,
		"",
		buttons,
	)

	dialog := s.DialogBorder.Render(inner)
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, dialog)
}
