package tui

import "github.com/charmbracelet/bubbles/key"

// ListKeyMap defines keybindings for the list screen.
type ListKeyMap struct {
	Up         key.Binding
	Down       key.Binding
	New        key.Binding
	Edit       key.Binding
	Delete     key.Binding
	Copy       key.Binding
	Filter     key.Binding
	Connect    key.Binding
	Tab        key.Binding
	Help       key.Binding
	Quit       key.Binding
	CycleTheme key.Binding
}

// FormKeyMap defines keybindings for the form screen.
type FormKeyMap struct {
	NextField key.Binding
	PrevField key.Binding
	Save      key.Binding
	OpenEditor key.Binding
	Back      key.Binding
}

// ConfirmKeyMap defines keybindings for the confirm dialog.
type ConfirmKeyMap struct {
	Yes   key.Binding
	No    key.Binding
	Left  key.Binding
	Right key.Binding
}

var DefaultListKeys = ListKeyMap{
	Up: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("↑/k", "up"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("↓/j", "down"),
	),
	New: key.NewBinding(
		key.WithKeys("n"),
		key.WithHelp("n", "new"),
	),
	Edit: key.NewBinding(
		key.WithKeys("e", "enter"),
		key.WithHelp("e", "edit"),
	),
	Delete: key.NewBinding(
		key.WithKeys("d"),
		key.WithHelp("d", "delete"),
	),
	Copy: key.NewBinding(
		key.WithKeys("c"),
		key.WithHelp("c", "copy URI"),
	),
	Filter: key.NewBinding(
		key.WithKeys("/"),
		key.WithHelp("/", "filter"),
	),
	Connect: key.NewBinding(
		key.WithKeys("M"),
		key.WithHelp("M", "connect MCP"),
	),
	Tab: key.NewBinding(
		key.WithKeys("tab"),
		key.WithHelp("tab", "switch pane"),
	),
	Help: key.NewBinding(
		key.WithKeys("?"),
		key.WithHelp("?", "help"),
	),
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q", "quit"),
	),
	CycleTheme: key.NewBinding(
		key.WithKeys("t"),
		key.WithHelp("t", "theme"),
	),
}

var DefaultFormKeys = FormKeyMap{
	NextField: key.NewBinding(
		key.WithKeys("tab"),
		key.WithHelp("tab", "next field"),
	),
	PrevField: key.NewBinding(
		key.WithKeys("shift+tab"),
		key.WithHelp("shift+tab", "prev field"),
	),
	Save: key.NewBinding(
		key.WithKeys("ctrl+s"),
		key.WithHelp("ctrl+s", "save"),
	),
	OpenEditor: key.NewBinding(
		key.WithKeys("ctrl+e"),
		key.WithHelp("ctrl+e", "open $EDITOR"),
	),
	Back: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "back"),
	),
}

var DefaultConfirmKeys = ConfirmKeyMap{
	Yes: key.NewBinding(
		key.WithKeys("y"),
		key.WithHelp("y", "yes"),
	),
	No: key.NewBinding(
		key.WithKeys("n", "esc", "q"),
		key.WithHelp("n/esc", "cancel"),
	),
	Left: key.NewBinding(
		key.WithKeys("left", "h"),
	),
	Right: key.NewBinding(
		key.WithKeys("right", "l"),
	),
}
