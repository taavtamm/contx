package tui

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/taavtamm/contx/internal/store"
)

// FormSavedMsg is sent when the form is saved.
type FormSavedMsg struct{ Context *store.Context }

// FormCancelledMsg is sent when the form is cancelled.
type FormCancelledMsg struct{}

type formField int

const (
	fieldName formField = iota
	fieldDescription
	fieldTags
	fieldScope
	fieldBody
	fieldCount
)

// scopeOption indices
const (
	scopeGlobalIdx    = 0
	scopeProjectIdx   = 1
	scopeUnmanagedIdx = 2
)

// FormModel is the create/edit form.
type FormModel struct {
	styles      *Styles
	keys        FormKeyMap
	ms          *store.MultiStore
	editing     *store.Context
	active      formField
	name        textinput.Model
	description textinput.Model
	tags        textinput.Model
	body        textarea.Model
	scopeIdx    int // 0=Global 1=Project 2=Unmanaged (Unmanaged only valid when editing)
	width       int
	height      int
	err         string
	formScroll  int // scroll offset for the scrollable content area
}

func NewForm(styles *Styles, ms *store.MultiStore, editing *store.Context, w, h int) FormModel {
	name := textinput.New()
	name.Placeholder = "my-context"
	name.CharLimit = 64

	desc := textinput.New()
	desc.Placeholder = "Short description"
	desc.CharLimit = 128

	tags := textinput.New()
	tags.Placeholder = "tag1, tag2"
	tags.CharLimit = 256

	body := textarea.New()
	body.Placeholder = "Context content..."
	body.ShowLineNumbers = false
	body.SetWidth(w - 20)
	body.SetHeight(calcBodyHeight(h))

	// Apply explicit text colors so inputs are readable on dark backgrounds.
	t := styles.Theme
	textSt := lipgloss.NewStyle().Foreground(t.Text)
	placeholderSt := lipgloss.NewStyle().Foreground(t.Comment)
	promptSt := lipgloss.NewStyle().Foreground(t.Subtle)
	for _, ti := range []*textinput.Model{&name, &desc, &tags} {
		ti.TextStyle = textSt
		ti.PlaceholderStyle = placeholderSt
		ti.PromptStyle = promptSt
		ti.Cursor.TextStyle = textSt
	}
	body.FocusedStyle.Text = textSt
	body.BlurredStyle.Text = textSt
	body.FocusedStyle.Placeholder = placeholderSt
	body.BlurredStyle.Placeholder = placeholderSt
	body.FocusedStyle.Prompt = promptSt
	body.BlurredStyle.Prompt = promptSt

	scopeIdx := scopeGlobalIdx

	if editing != nil {
		name.SetValue(editing.Name)
		desc.SetValue(editing.Description)
		tags.SetValue(strings.Join(editing.Tags, ", "))
		body.SetValue(editing.Body)
		if editing.Scope == store.ScopeProject {
			scopeIdx = scopeProjectIdx
		}
	}

	name.Focus()

	return FormModel{
		styles:   styles,
		keys:     DefaultFormKeys,
		ms:       ms,
		editing:  editing,
		active:   fieldName,
		name:     name,
		description: desc,
		tags:     tags,
		body:     body,
		scopeIdx: scopeIdx,
		width:    w,
		height:   h,
	}
}

// NewImportForm creates a new-context form pre-filled from an unmanaged file.
// The scope defaults to project since the file belongs to the current project.
func NewImportForm(styles *Styles, ms *store.MultiStore, name, body string, w, h int) FormModel {
	m := NewForm(styles, ms, nil, w, h)
	m.name.SetValue(name)
	m.body.SetValue(body)
	m.scopeIdx = scopeProjectIdx // default to project scope for repo files
	// Jump focus to description so the user can add a short description.
	m.blurActive()
	m.active = fieldDescription
	m.focusActive()
	m.scrollToActive()
	return m
}

func (m FormModel) Init() tea.Cmd {
	return textinput.Blink
}

type editorDoneMsg struct{ content string }

func (m FormModel) Update(msg tea.Msg) (FormModel, tea.Cmd) {
	switch msg := msg.(type) {
	case editorDoneMsg:
		m.body.SetValue(msg.content)
		return m, nil

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.body.SetWidth(msg.Width - 20)
		m.body.SetHeight(m.bodyHeight())
		m.scrollToActive()
		return m, nil

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keys.Back):
			return m, func() tea.Msg { return FormCancelledMsg{} }

		case key.Matches(msg, m.keys.Save):
			return m.save()

		case key.Matches(msg, m.keys.OpenEditor) && m.active == fieldBody:
			return m, m.openEditor()

		case key.Matches(msg, m.keys.NextField):
			m = m.nextField()
			return m, nil

		case key.Matches(msg, m.keys.PrevField):
			m = m.prevField()
			return m, nil

		default:
			if m.active == fieldScope {
				s := msg.String()
				n := m.numScopes()
				if s == "right" || s == "l" || s == " " {
					m.scopeIdx = (m.scopeIdx + 1) % n
					return m, nil
				}
				if s == "left" || s == "h" {
					m.scopeIdx = (m.scopeIdx - 1 + n) % n
					return m, nil
				}
			}
		}
	}

	var cmd tea.Cmd
	switch m.active {
	case fieldName:
		m.name, cmd = m.name.Update(msg)
	case fieldDescription:
		m.description, cmd = m.description.Update(msg)
	case fieldTags:
		m.tags, cmd = m.tags.Update(msg)
	case fieldBody:
		m.body, cmd = m.body.Update(msg)
	}
	return m, cmd
}

func (m FormModel) nextField() FormModel {
	m.blurActive()
	m.active = (m.active + 1) % fieldCount
	m.focusActive()
	m.scrollToActive()
	return m
}

func (m FormModel) prevField() FormModel {
	m.blurActive()
	if m.active == 0 {
		m.active = fieldCount - 1
	} else {
		m.active--
	}
	m.focusActive()
	m.scrollToActive()
	return m
}

// calcBodyHeight returns the textarea height for a given terminal height.
// It fills all remaining vertical space, with a minimum of 8 lines.
// Fixed overhead: 4 lines outside viewport + 15 lines of form fields before body + 2 border lines.
func calcBodyHeight(termH int) int {
	h := termH - 21 // 4 fixed + 15 pre-body content + 2 body border
	if h < 8 {
		return 8
	}
	return h
}

func (m FormModel) bodyHeight() int {
	return calcBodyHeight(m.height)
}

// scrollToActive adjusts formScroll so the active field is visible in the
// viewport. The scrollable area sits between the fixed header and footer
// (each 2 lines), so viewportH = m.height - 4.
//
// Flat line positions inside the scrollable content:
//   0      blank before Name
//   1–3    Name field (3 lines, rounded border)
//   4      blank
//   5–7    Description field
//   8      blank
//   9–11   Tags field
//   12     blank
//   13     Scope widget
//   14     blank
//   15–(15+bh+1)  Body textarea (bh content + 2 border lines)
func (m *FormModel) scrollToActive() {
	viewportH := m.height - 4
	if viewportH < 1 {
		return
	}

	bh := m.bodyHeight()
	var start, end int
	switch m.active {
	case fieldName:
		start, end = 1, 4
	case fieldDescription:
		start, end = 5, 8
	case fieldTags:
		start, end = 9, 12
	case fieldScope:
		start, end = 13, 15
	case fieldBody:
		start, end = 15, 15+bh+2
	default:
		return
	}

	if start < m.formScroll {
		m.formScroll = start
	}
	if end > m.formScroll+viewportH {
		m.formScroll = end - viewportH
	}
	if m.formScroll < 0 {
		m.formScroll = 0
	}
}

func (m *FormModel) blurActive() {
	switch m.active {
	case fieldName:
		m.name.Blur()
	case fieldDescription:
		m.description.Blur()
	case fieldTags:
		m.tags.Blur()
	case fieldBody:
		m.body.Blur()
	}
}

func (m *FormModel) focusActive() {
	switch m.active {
	case fieldName:
		m.name.Focus()
	case fieldDescription:
		m.description.Focus()
	case fieldTags:
		m.tags.Focus()
	case fieldBody:
		m.body.Focus()
	}
}

func (m FormModel) save() (FormModel, tea.Cmd) {
	// Unmanaged = delete the context and return it to the unmanaged file list.
	if m.scopeIdx == scopeUnmanagedIdx && m.editing != nil {
		m.ms.Delete(m.editing.Scope, m.editing.Name) //nolint:errcheck
		return m, func() tea.Msg { return FormSavedMsg{} }
	}

	name := strings.TrimSpace(m.name.Value())
	if name == "" {
		m.err = "name is required"
		return m, nil
	}
	if strings.ContainsAny(name, " /\\") {
		m.err = "name must not contain spaces or slashes"
		return m, nil
	}

	var tagList []string
	for _, t := range strings.Split(m.tags.Value(), ",") {
		if t = strings.TrimSpace(t); t != "" {
			tagList = append(tagList, t)
		}
	}

	scope := store.ScopeProject
	if m.scopeIdx == scopeGlobalIdx {
		scope = store.ScopeGlobal
	}

	now := time.Now().UTC()
	c := &store.Context{
		Name:        name,
		Description: strings.TrimSpace(m.description.Value()),
		Tags:        tagList,
		Scope:       scope,
		Body:        m.body.Value(),
		UpdatedAt:   now,
	}
	if m.editing != nil {
		c.CreatedAt = m.editing.CreatedAt
	} else {
		c.CreatedAt = now
	}

	if err := m.ms.Save(c); err != nil {
		if errors.Is(err, store.ErrNoProjectRoot) {
			m.err = "no project root found; cannot save project-scoped context"
		} else {
			m.err = fmt.Sprintf("save failed: %v", err)
		}
		return m, nil
	}

	return m, func() tea.Msg { return FormSavedMsg{Context: c} }
}

func (m FormModel) openEditor() tea.Cmd {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vi"
	}
	tmp, err := os.CreateTemp("", "contx-*.md")
	if err != nil {
		return nil
	}
	tmp.WriteString(m.body.Value())
	tmp.Close()
	tmpPath := tmp.Name()

	return tea.ExecProcess(exec.Command(editor, tmpPath), func(err error) tea.Msg {
		defer os.Remove(tmpPath)
		if err != nil {
			return nil
		}
		content, readErr := os.ReadFile(tmpPath)
		if readErr != nil {
			return nil
		}
		return editorDoneMsg{content: string(content)}
	})
}

func (m FormModel) View() string {
	s := m.styles
	w := m.width - 4

	title := "New Context"
	if m.editing != nil {
		title = "Edit: " + m.editing.Name
	}

	appTitleStr := s.AppTitle.Render("contx")
	crumbStr := s.Breadcrumb.Render("›  " + title)
	hintStr := s.HintDesc.Render("esc back")
	padding := lipgloss.NewStyle().
		Width(w - lipgloss.Width(appTitleStr) - lipgloss.Width(crumbStr) - lipgloss.Width(hintStr) - 2).
		Render("")
	header := appTitleStr + "  " + crumbStr + padding + hintStr
	header = lipgloss.NewStyle().Padding(0, 1).Width(w+4).Render(header)

	divider := s.Divider.Render(strings.Repeat("━", w+4))
	footer := s.Footer.Width(w + 4).Render("  tab next field   ctrl+s save   ctrl+e open $EDITOR   esc back")

	inputWidth := w - 6
	labelWidth := 16

	nameLabel, nameBox := m.renderInput("Name", m.tiContent(m.name, m.active == fieldName), m.active == fieldName, inputWidth)
	descLabel, descBox := m.renderInput("Description", m.tiContent(m.description, m.active == fieldDescription), m.active == fieldDescription, inputWidth)
	tagsLabel, tagsBox := m.renderInput("Tags", m.tiContent(m.tags, m.active == fieldTags), m.active == fieldTags, inputWidth)
	tagsHint := s.PreviewMeta.Render("[comma-sep]")
	scopeLabel, scopeWidget := m.renderScope()
	bodyLabel, bodyBox := m.renderTextarea(inputWidth)

	errLine := ""
	if m.err != "" {
		errLine = "  " + lipgloss.NewStyle().Foreground(s.Theme.Red).Render("Error: "+m.err)
	}

	// Build all scrollable rows then flatten to individual terminal lines.
	scrollRows := []string{
		"",
		lipgloss.JoinHorizontal(lipgloss.Top,
			lipgloss.NewStyle().Width(labelWidth).Render("  "+nameLabel),
			nameBox,
		),
		"",
		lipgloss.JoinHorizontal(lipgloss.Top,
			lipgloss.NewStyle().Width(labelWidth).Render("  "+descLabel),
			descBox,
		),
		"",
		lipgloss.JoinHorizontal(lipgloss.Top,
			lipgloss.NewStyle().Width(labelWidth).Render(
				lipgloss.JoinVertical(lipgloss.Left, "  "+tagsLabel, "  "+tagsHint),
			),
			tagsBox,
		),
		"",
		lipgloss.JoinHorizontal(lipgloss.Top,
			lipgloss.NewStyle().Width(labelWidth).Render("  "+scopeLabel),
			scopeWidget,
		),
		"",
		lipgloss.JoinHorizontal(lipgloss.Top,
			lipgloss.NewStyle().Width(labelWidth).Render("  "+bodyLabel),
			bodyBox,
		),
		errLine,
		"",
	}

	allLines := strings.Split(strings.Join(scrollRows, "\n"), "\n")

	// header(1) + divider(1) + divider(1) + footer(1) = 4 fixed lines
	viewportH := m.height - 4
	if viewportH < 1 {
		viewportH = 1
	}

	maxScroll := len(allLines) - viewportH
	if maxScroll < 0 {
		maxScroll = 0
	}
	scroll := m.formScroll
	if scroll > maxScroll {
		scroll = maxScroll
	}
	if scroll < 0 {
		scroll = 0
	}

	end := scroll + viewportH
	if end > len(allLines) {
		end = len(allLines)
	}
	visible := allLines[scroll:end]
	for len(visible) < viewportH {
		visible = append(visible, "")
	}

	return strings.Join([]string{
		header,
		divider,
		strings.Join(visible, "\n"),
		divider,
		footer,
	}, "\n")
}

// tiContent returns the display string for a textinput.
// When active, uses the full View() (cursor etc.). When blurred, renders the
// raw value directly with explicit colors so lipgloss v1.x doesn't swallow them.
func (m FormModel) tiContent(ti textinput.Model, active bool) string {
	if active {
		return ti.View()
	}
	s := m.styles
	prompt := lipgloss.NewStyle().Foreground(s.Theme.Subtle).Render(ti.Prompt)
	if ti.Value() == "" {
		return prompt + lipgloss.NewStyle().Foreground(s.Theme.Comment).Render(ti.Placeholder)
	}
	return prompt + lipgloss.NewStyle().Foreground(s.Theme.Text).Render(ti.Value())
}

func (m FormModel) renderInput(label, value string, active bool, width int) (string, string) {
	s := m.styles
	lStyle, bStyle := s.FormLabel, s.FormInput.Width(width)
	if active {
		lStyle = s.FormLabelActive
		bStyle = s.FormInputActive.Width(width)
	}
	return lStyle.Render(label), bStyle.Render(value)
}


func (m FormModel) numScopes() int {
	if m.editing != nil {
		return 3
	}
	return 2
}

func (m FormModel) renderScope() (string, string) {
	s := m.styles
	lStyle := s.FormLabel
	if m.active == fieldScope {
		lStyle = s.FormLabelActive
	}

	labels := []string{"Global", "Project"}
	if m.editing != nil {
		labels = append(labels, "Unmanaged")
	}

	parts := make([]string, len(labels))
	for i, label := range labels {
		if i == m.scopeIdx {
			parts[i] = s.FormRadioActive.Render("(●) " + label)
		} else {
			parts[i] = s.FormRadio.Render("(○) " + label)
		}
	}

	widget := strings.Join(parts, "    ")
	return lStyle.Render("Scope"), widget
}

func (m FormModel) renderTextarea(width int) (string, string) {
	s := m.styles
	lStyle, bStyle := s.FormLabel, s.FormInput.Width(width)
	active := m.active == fieldBody
	if active {
		lStyle = s.FormLabelActive
		bStyle = s.FormInputActive.Width(width)
	}

	bodyH := m.bodyHeight()

	var content string
	if active {
		content = m.body.View()
	} else {
		// Render at a fixed bodyH lines so the form height never shifts between
		// focused and blurred states (blurred content could otherwise be 100+ lines).
		prompt := lipgloss.NewStyle().Foreground(s.Theme.Subtle).Render("┃ ")
		textSt := lipgloss.NewStyle().Foreground(s.Theme.Text)
		mutedSt := lipgloss.NewStyle().Foreground(s.Theme.Comment)

		lines := strings.Split(m.body.Value(), "\n")
		rendered := make([]string, 0, bodyH)

		if len(lines) > bodyH {
			for _, l := range lines[:bodyH-1] {
				rendered = append(rendered, prompt+textSt.Render(l))
			}
			rendered = append(rendered, mutedSt.Render(fmt.Sprintf("  … +%d lines", len(lines)-(bodyH-1))))
		} else {
			for _, l := range lines {
				rendered = append(rendered, prompt+textSt.Render(l))
			}
			for len(rendered) < bodyH {
				rendered = append(rendered, "")
			}
		}
		content = strings.Join(rendered, "\n")
	}
	return lStyle.Render("Body"), bStyle.Render(content)
}
