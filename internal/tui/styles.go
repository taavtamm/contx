package tui

import "github.com/charmbracelet/lipgloss"

// Styles holds all prebuilt lipgloss styles for a given theme.
type Styles struct {
	Theme Theme

	// Layout
	Header      lipgloss.Style
	Footer      lipgloss.Style
	Divider     lipgloss.Style
	PaneDivider lipgloss.Style

	// List pane
	SectionHeader   lipgloss.Style
	SectionDivider  lipgloss.Style
	ItemNormal      lipgloss.Style
	ItemSelected    lipgloss.Style
	ItemCursor      string

	// Preview pane
	PreviewTitle    lipgloss.Style
	PreviewDivider  lipgloss.Style
	PreviewBody     lipgloss.Style
	PreviewMeta     lipgloss.Style
	PreviewMetaKey  lipgloss.Style

	// Form
	FormLabel       lipgloss.Style
	FormLabelActive lipgloss.Style
	FormInput       lipgloss.Style
	FormInputActive lipgloss.Style
	FormRadio       lipgloss.Style
	FormRadioActive lipgloss.Style
	Breadcrumb      lipgloss.Style

	// Dialog
	DialogBorder   lipgloss.Style
	DialogTitle    lipgloss.Style
	DialogBody     lipgloss.Style
	BtnDelete      lipgloss.Style
	BtnCancel      lipgloss.Style

	// General
	AppTitle   lipgloss.Style
	ThemeName  lipgloss.Style
	HintKey    lipgloss.Style
	HintDesc   lipgloss.Style
	URIStyle   lipgloss.Style
	TagStyle   lipgloss.Style
}

// NewStyles builds a Styles from the given Theme.
func NewStyles(t Theme) *Styles {
	s := &Styles{Theme: t}

	s.Header = lipgloss.NewStyle().
		Background(t.Surface).
		Foreground(t.Text).
		Bold(true).
		Padding(0, 1)

	s.Footer = lipgloss.NewStyle().
		Background(t.Surface).
		Foreground(t.Comment).
		Padding(0, 1)

	s.Divider = lipgloss.NewStyle().
		Foreground(t.Accent)

	s.PaneDivider = lipgloss.NewStyle().
		Foreground(t.Overlay)

	s.SectionHeader = lipgloss.NewStyle().
		Foreground(t.Accent).
		Bold(true).
		Padding(0, 1)

	s.SectionDivider = lipgloss.NewStyle().
		Foreground(t.Overlay)

	s.ItemNormal = lipgloss.NewStyle().
		Foreground(t.Text).
		Padding(0, 1)

	s.ItemSelected = lipgloss.NewStyle().
		Background(t.Accent).
		Foreground(t.Base).
		Bold(true).
		Padding(0, 1)

	s.ItemCursor = "▶"

	s.PreviewTitle = lipgloss.NewStyle().
		Foreground(t.Text).
		Bold(true)

	s.PreviewDivider = lipgloss.NewStyle().
		Foreground(t.Overlay)

	s.PreviewBody = lipgloss.NewStyle().
		Foreground(t.Text)

	s.PreviewMeta = lipgloss.NewStyle().
		Foreground(t.Subtle)

	s.PreviewMetaKey = lipgloss.NewStyle().
		Foreground(t.Subtle)

	s.FormLabel = lipgloss.NewStyle().
		Foreground(t.Muted)

	s.FormLabelActive = lipgloss.NewStyle().
		Foreground(t.Accent).
		Bold(true)

	s.FormInput = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.Overlay).
		Foreground(t.Text).
		Padding(0, 1)

	s.FormInputActive = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.Accent).
		Foreground(t.Text).
		Padding(0, 1)

	s.FormRadio = lipgloss.NewStyle().
		Foreground(t.Muted)

	s.FormRadioActive = lipgloss.NewStyle().
		Foreground(t.Accent).
		Bold(true)

	s.Breadcrumb = lipgloss.NewStyle().
		Foreground(t.Subtle)

	s.DialogBorder = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.Red).
		Padding(1, 2)

	s.DialogTitle = lipgloss.NewStyle().
		Foreground(t.Text).
		Bold(true)

	s.DialogBody = lipgloss.NewStyle().
		Foreground(t.Subtle)

	s.BtnDelete = lipgloss.NewStyle().
		Background(t.Red).
		Foreground(t.Base).
		Bold(true).
		Padding(0, 2)

	s.BtnCancel = lipgloss.NewStyle().
		Foreground(t.Muted).
		Padding(0, 2)

	s.AppTitle = lipgloss.NewStyle().
		Foreground(t.Accent).
		Bold(true)

	s.ThemeName = lipgloss.NewStyle().
		Foreground(t.Comment)

	s.HintKey = lipgloss.NewStyle().
		Foreground(t.Accent)

	s.HintDesc = lipgloss.NewStyle().
		Foreground(t.Subtle)

	s.URIStyle = lipgloss.NewStyle().
		Foreground(t.Cyan)

	s.TagStyle = lipgloss.NewStyle().
		Foreground(t.Yellow)

	return s
}
