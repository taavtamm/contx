package tui

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	blist "github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/taavtamm/contx/internal/store"
)

// Messages sent from the list screen to the app.
type NewContextMsg struct{}
type EditContextMsg struct{ Ctx *store.Context }
type DeleteContextMsg struct{ Ctx *store.Context }
type CycleThemeMsg struct{}
type ImportFileMsg struct{ File store.UnmanagedFile }

// Internal clipboard messages.
type copiedMsg struct{}
type clearCopiedMsg struct{}

// listItem wraps a *store.Context for the bubbles list.
type listItem struct{ Ctx *store.Context }

func (i listItem) FilterValue() string { return i.Ctx.Name }
func (i listItem) Title() string       { return i.Ctx.Name }
func (i listItem) Description() string { return i.Ctx.Description }

// sectionHeader is a non-selectable section header item.
type sectionHeader struct{ Label string }

func (s sectionHeader) FilterValue() string { return "" }
func (s sectionHeader) Title() string       { return s.Label }
func (s sectionHeader) Description() string { return "" }

// branchItem shows the git branch below the PROJECT header in a muted style.
type branchItem struct{ Branch string }

func (b branchItem) FilterValue() string { return "" }
func (b branchItem) Title() string       { return b.Branch }
func (b branchItem) Description() string { return "" }

// unmanagedItem represents a repo file not yet imported as a contx context.
type unmanagedItem struct{ File store.UnmanagedFile }

func (u unmanagedItem) FilterValue() string { return u.File.Name }
func (u unmanagedItem) Title() string       { return u.File.Name }
func (u unmanagedItem) Description() string { return u.File.RelPath }

// listDelegate renders items in the list.
type listDelegate struct{ styles *Styles }

func (d listDelegate) Height() int                               { return 1 }
func (d listDelegate) Spacing() int                              { return 0 }
func (d listDelegate) Update(_ tea.Msg, _ *blist.Model) tea.Cmd { return nil }

func (d listDelegate) Render(w io.Writer, m blist.Model, index int, item blist.Item) {
	s := d.styles
	switch it := item.(type) {
	case sectionHeader:
		io.WriteString(w, s.SectionHeader.Render(it.Label))
	case branchItem:
		io.WriteString(w, lipgloss.NewStyle().Foreground(s.Theme.Green).Padding(0, 1).Render("⎇ "+it.Branch))
	case unmanagedItem:
		label := it.File.RelPath
		if index == m.Index() {
			io.WriteString(w, s.ItemSelected.Render(
				lipgloss.NewStyle().Width(m.Width()-2).Render(label),
			))
		} else {
			io.WriteString(w, lipgloss.NewStyle().Foreground(s.Theme.Yellow).Padding(0, 1).
				Render(lipgloss.NewStyle().Width(m.Width()-2).Render(label)))
		}
	case listItem:
		name := it.Ctx.Name
		if index == m.Index() {
			io.WriteString(w, s.ItemSelected.Render(
				lipgloss.NewStyle().Width(m.Width()-2).Render(name),
			))
		} else {
			io.WriteString(w, s.ItemNormal.Render(
				lipgloss.NewStyle().Width(m.Width()-2).Render(name),
			))
		}
	}
}

func buildListItems(contexts []*store.Context, projectName, gitBranch string, unmanaged []store.UnmanagedFile) []blist.Item {
	var global, project []*store.Context
	for _, c := range contexts {
		if c.Scope == store.ScopeGlobal {
			global = append(global, c)
		} else {
			project = append(project, c)
		}
	}

	var items []blist.Item
	if len(global) > 0 {
		items = append(items, sectionHeader{Label: "◆ GLOBAL"})
		items = append(items, sectionHeader{Label: strings.Repeat("─", 23)})
		for _, c := range global {
			items = append(items, listItem{Ctx: c})
		}
	}
	if len(project) > 0 {
		if len(global) > 0 {
			items = append(items, sectionHeader{Label: ""})
		}
		label := "◆ PROJECT"
		if projectName != "" {
			label += " — " + projectName
		}
		items = append(items, sectionHeader{Label: label})
		if gitBranch != "" {
			items = append(items, branchItem{Branch: gitBranch})
		}
		items = append(items, sectionHeader{Label: strings.Repeat("─", 23)})
		for _, c := range project {
			items = append(items, listItem{Ctx: c})
		}
	}
	if len(unmanaged) > 0 {
		if len(global) > 0 || len(project) > 0 {
			items = append(items, sectionHeader{Label: ""})
		}
		label := "◆ UNMANAGED"
		if projectName != "" {
			label += " — " + projectName
		}
		items = append(items, sectionHeader{Label: label})
		if gitBranch != "" {
			items = append(items, branchItem{Branch: gitBranch})
		}
		items = append(items, sectionHeader{Label: strings.Repeat("─", 23)})
		for _, f := range unmanaged {
			items = append(items, unmanagedItem{File: f})
		}
	}
	return items
}

// ListModel is the two-pane list + preview screen.
type ListModel struct {
	styles        *Styles
	keys          ListKeyMap
	ms            *store.MultiStore
	list          blist.Model
	contexts      []*store.Context
	unmanaged     []store.UnmanagedFile
	projectName   string // folder name of the project root, or ""
	gitBranch     string // current git branch, or ""
	width         int
	height        int
	focusRight    bool // true = preview pane has keyboard focus
	previewScroll int  // number of body lines scrolled in the preview
	copied        bool // true while "copied!" flash is visible
}

func NewList(styles *Styles, ms *store.MultiStore, w, h int) (ListModel, error) {
	contexts, err := ms.List()
	if err != nil {
		return ListModel{}, err
	}

	projectName, gitBranch := projectInfo(ms.ProjectRoot)

	managedNames := make(map[string]bool, len(contexts))
	for _, c := range contexts {
		managedNames[c.Name] = true
	}
	scanRoot := ms.ProjectRoot
	if scanRoot == "" {
		scanRoot, _ = os.Getwd()
	}
	unmanaged := store.FindUnmanagedFiles(scanRoot, managedNames)

	delegate := listDelegate{styles: styles}
	items := buildListItems(contexts, projectName, gitBranch, unmanaged)

	l := blist.New(items, delegate, w/2-2, h-6)
	l.SetShowHelp(false)
	l.SetShowTitle(false)
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.DisableQuitKeybindings()

	return ListModel{
		styles:      styles,
		keys:        DefaultListKeys,
		ms:          ms,
		list:        l,
		contexts:    contexts,
		unmanaged:   unmanaged,
		projectName: projectName,
		gitBranch:   gitBranch,
		width:       w,
		height:      h,
	}, nil
}

func (m ListModel) Init() tea.Cmd { return nil }

func (m ListModel) Update(msg tea.Msg) (ListModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keys.Quit):
			return m, tea.Quit

		case key.Matches(msg, m.keys.Tab):
			m.focusRight = !m.focusRight
			return m, nil

		case key.Matches(msg, m.keys.Up):
			if m.focusRight {
				if m.previewScroll > 0 {
					m.previewScroll--
				}
				return m, nil
			}
			var cmd tea.Cmd
			m.list, cmd = m.list.Update(msg)
			m.skipHeaders(false)
			m.previewScroll = 0
			return m, cmd

		case key.Matches(msg, m.keys.Down):
			if m.focusRight {
				m.previewScroll++ // clamped in renderPreview
				return m, nil
			}
			var cmd tea.Cmd
			m.list, cmd = m.list.Update(msg)
			m.skipHeaders(true)
			m.previewScroll = 0
			return m, cmd

		case key.Matches(msg, m.keys.New):
			if !m.focusRight {
				return m, func() tea.Msg { return NewContextMsg{} }
			}

		case key.Matches(msg, m.keys.Edit):
			if !m.focusRight {
				if c := m.selectedContext(); c != nil {
					return m, func() tea.Msg { return EditContextMsg{Ctx: c} }
				}
				if f := m.selectedUnmanaged(); f != nil {
					file := *f
					return m, func() tea.Msg { return ImportFileMsg{File: file} }
				}
			}

		case key.Matches(msg, m.keys.Delete):
			if !m.focusRight {
				if c := m.selectedContext(); c != nil {
					return m, func() tea.Msg { return DeleteContextMsg{Ctx: c} }
				}
			}

		case key.Matches(msg, m.keys.Copy):
			if !m.focusRight {
				if c := m.selectedContext(); c != nil {
					return m, copyToClipboard(c.URI())
				}
			}

		case key.Matches(msg, m.keys.CycleTheme):
			return m, func() tea.Msg { return CycleThemeMsg{} }
		}

	case copiedMsg:
		m.copied = true
		return m, tea.Tick(2*time.Second, func(time.Time) tea.Msg { return clearCopiedMsg{} })

	case clearCopiedMsg:
		m.copied = false
		return m, nil

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.list.SetSize(m.width/2-2, m.height-6)
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m *ListModel) skipHeaders(down bool) {
	items := m.list.Items()
	idx := m.list.Index()
	for idx >= 0 && idx < len(items) {
		switch items[idx].(type) {
		case listItem, unmanagedItem:
			return
		}
		if down {
			if idx+1 < len(items) {
				m.list.Select(idx + 1)
				idx++
			} else {
				break
			}
		} else {
			if idx > 0 {
				m.list.Select(idx - 1)
				idx--
			} else {
				break
			}
		}
	}
}

func (m *ListModel) selectedContext() *store.Context {
	item, ok := m.list.SelectedItem().(listItem)
	if !ok {
		return nil
	}
	return item.Ctx
}

func (m *ListModel) selectedUnmanaged() *store.UnmanagedFile {
	item, ok := m.list.SelectedItem().(unmanagedItem)
	if !ok {
		return nil
	}
	return &item.File
}

func (m *ListModel) Reload() error {
	contexts, err := m.ms.List()
	if err != nil {
		return err
	}
	m.contexts = contexts
	// Refresh git branch in case the user switched branches while contx was open.
	m.projectName, m.gitBranch = projectInfo(m.ms.ProjectRoot)

	managedNames := make(map[string]bool, len(contexts))
	for _, c := range contexts {
		managedNames[c.Name] = true
	}
	scanRoot := m.ms.ProjectRoot
	if scanRoot == "" {
		scanRoot, _ = os.Getwd()
	}
	m.unmanaged = store.FindUnmanagedFiles(scanRoot, managedNames)

	m.list.SetItems(buildListItems(contexts, m.projectName, m.gitBranch, m.unmanaged))
	m.previewScroll = 0
	return nil
}

func (m ListModel) View() string {
	s := m.styles
	w := m.width

	// Header
	appTitleStr := s.AppTitle.Render("contx")
	themeStr := s.ThemeName.Render(s.Theme.Name)
	quitStr := s.HintDesc.Render("q quit")
	rightStr := themeStr + "  " + quitStr
	middlePad := lipgloss.NewStyle().
		Width(w - lipgloss.Width(appTitleStr) - lipgloss.Width(rightStr) - 4).
		Render("")
	header := lipgloss.NewStyle().
		Padding(0, 1).
		Width(w).
		Render(lipgloss.JoinHorizontal(lipgloss.Top, appTitleStr, middlePad, rightStr))

	divider := s.Divider.Render(strings.Repeat("━", w))

	leftWidth := w/2 - 1
	rightWidth := w - leftWidth - 3

	m.list.SetSize(leftWidth-2, m.height-6)
	leftPane := lipgloss.NewStyle().Width(leftWidth).Height(m.height - 6).Render(m.list.View())

	rightPane := m.renderPreview(rightWidth, m.height-6)

	// Vertical separator — accent colour when preview is focused
	sepStyle := s.PaneDivider
	if m.focusRight {
		sepStyle = s.Divider
	}
	sepLines := make([]string, m.height-6)
	for i := range sepLines {
		sepLines[i] = sepStyle.Render("│")
	}
	separator := strings.Join(sepLines, "\n")

	content := lipgloss.JoinHorizontal(lipgloss.Top, leftPane, separator, rightPane)

	// Footer — hints change based on which pane is active
	var hints []string
	if m.focusRight {
		hints = []string{
			s.HintKey.Render("tab") + " " + s.HintDesc.Render("← list"),
			s.HintKey.Render("↑/↓") + " " + s.HintDesc.Render("scroll"),
			s.HintKey.Render("t") + " " + s.HintDesc.Render("theme"),
		}
	} else {
		editHint := "edit"
		if m.selectedUnmanaged() != nil {
			editHint = "import"
		}
		copyHint := s.HintKey.Render("c") + " " + s.HintDesc.Render("copy URI")
		if m.copied {
			copyHint = lipgloss.NewStyle().Foreground(s.Theme.Green).Render("✓ copied!")
		}
		hints = []string{
			s.HintKey.Render("n") + " " + s.HintDesc.Render("new"),
			s.HintKey.Render("e") + " " + s.HintDesc.Render(editHint),
			s.HintKey.Render("d") + " " + s.HintDesc.Render("delete"),
			copyHint,
			s.HintKey.Render("tab") + " " + s.HintDesc.Render("→ preview"),
			s.HintKey.Render("t") + " " + s.HintDesc.Render("theme"),
		}
	}
	footer := s.Footer.Width(w).Render("  " + strings.Join(hints, "   "))

	return lipgloss.JoinVertical(lipgloss.Left, header, divider, "", content, "", divider, footer)
}

func (m ListModel) renderPreview(width, height int) string {
	s := m.styles

	if f := m.selectedUnmanaged(); f != nil {
		return m.renderUnmanagedPreview(*f, width, height)
	}

	c := m.selectedContext()
	if c == nil {
		return lipgloss.NewStyle().Width(width).Height(height).
			Foreground(s.Theme.Subtle).Render("  No context selected")
	}

	const indent = "  "
	const indentLen = 2
	bodyWidth := width - indentLen - 1
	if bodyWidth < 10 {
		bodyWidth = 10
	}

	// Word-wrap the full body and get all visual lines.
	wrappedBody := s.PreviewBody.Width(bodyWidth).Render(c.Body)
	allBodyLines := strings.Split(wrappedBody, "\n")
	totalBodyLines := len(allBodyLines)

	// Fixed chrome: 3 header lines (title, divider, blank) + 5 footer lines
	// (blank, metaDivider, 3 meta rows).
	const fixedLines = 8
	bodyAvail := height - fixedLines
	if bodyAvail < 1 {
		bodyAvail = 1
	}

	// Clamp scroll offset.
	maxScroll := totalBodyLines - bodyAvail
	if maxScroll < 0 {
		maxScroll = 0
	}
	scroll := m.previewScroll
	if scroll > maxScroll {
		scroll = maxScroll
	}

	end := scroll + bodyAvail
	if end > totalBodyLines {
		end = totalBodyLines
	}
	bodyText := strings.Join(allBodyLines[scroll:end], "\n")

	// Title row — append scroll position when content overflows.
	title := s.PreviewTitle.Render(c.Name)
	if totalBodyLines > bodyAvail {
		pos := s.PreviewMeta.Render(fmt.Sprintf("%d/%d", scroll+bodyAvail, totalBodyLines))
		pad := width - lipgloss.Width(title) - lipgloss.Width(pos) - indentLen*2
		if pad > 0 {
			title = title + strings.Repeat(" ", pad) + pos
		}
	}
	titleDiv := s.PreviewDivider.Render(strings.Repeat("─", width-2))

	// Meta block.
	tagStr := strings.Join(c.Tags, "  ")
	if tagStr == "" {
		tagStr = s.PreviewMeta.Render("—")
	} else {
		tagStr = s.TagStyle.Render(tagStr)
	}

	scopeColor := s.Theme.Accent
	if c.Scope == store.ScopeProject {
		scopeColor = s.Theme.Purple
	}

	metaKey := func(label string) string {
		return s.PreviewMetaKey.Width(8).Render(label)
	}
	metaBlock := lipgloss.JoinVertical(lipgloss.Left,
		metaKey("Tags")+tagStr,
		metaKey("Scope")+lipgloss.NewStyle().Foreground(scopeColor).Render(string(c.Scope)),
		metaKey("Updated")+s.PreviewMeta.Render(fmtTime(c.UpdatedAt)),
	)

	metaDiv := s.PreviewDivider.Render(strings.Repeat("─", width-2))

	preview := lipgloss.JoinVertical(lipgloss.Left,
		indent+title,
		indent+titleDiv,
		"",
		indentBlock(bodyText, indent),
		"",
		indent+metaDiv,
		indentBlock(metaBlock, indent),
	)

	return lipgloss.NewStyle().Width(width).Height(height).Render(preview)
}

func (m ListModel) renderUnmanagedPreview(f store.UnmanagedFile, width, height int) string {
	s := m.styles
	const indent = "  "
	const indentLen = 2
	bodyWidth := width - indentLen - 1
	if bodyWidth < 10 {
		bodyWidth = 10
	}

	title := lipgloss.NewStyle().Foreground(s.Theme.Yellow).Bold(true).Render(f.RelPath)
	titleDiv := s.PreviewDivider.Render(strings.Repeat("─", width-2))
	hint := s.PreviewMeta.Render("e / ↩  import as context")

	// Word-wrap the preview content and apply scroll.
	wrappedBody := s.PreviewBody.Width(bodyWidth).Render(f.Preview)
	allBodyLines := strings.Split(wrappedBody, "\n")
	totalBodyLines := len(allBodyLines)

	const fixedLines = 6 // title, divider, blank, blank, hint divider, hint
	bodyAvail := height - fixedLines
	if bodyAvail < 1 {
		bodyAvail = 1
	}
	maxScroll := totalBodyLines - bodyAvail
	if maxScroll < 0 {
		maxScroll = 0
	}
	scroll := m.previewScroll
	if scroll > maxScroll {
		scroll = maxScroll
	}
	end := scroll + bodyAvail
	if end > totalBodyLines {
		end = totalBodyLines
	}
	bodyText := strings.Join(allBodyLines[scroll:end], "\n")

	scrollInfo := ""
	if totalBodyLines > bodyAvail {
		scrollInfo = s.PreviewMeta.Render(fmt.Sprintf(" %d/%d", scroll+bodyAvail, totalBodyLines))
	}

	hintDiv := s.PreviewDivider.Render(strings.Repeat("─", width-2))

	preview := lipgloss.JoinVertical(lipgloss.Left,
		indent+title+scrollInfo,
		indent+titleDiv,
		"",
		indentBlock(bodyText, indent),
		"",
		indent+hintDiv,
		indent+hint,
	)
	return lipgloss.NewStyle().Width(width).Height(height).Render(preview)
}

func indentBlock(text, prefix string) string {
	lines := strings.Split(text, "\n")
	for i, l := range lines {
		lines[i] = prefix + l
	}
	return strings.Join(lines, "\n")
}

func fmtTime(t time.Time) string {
	if t.IsZero() {
		return "—"
	}
	return fmt.Sprintf("%d-%02d-%02d", t.Year(), t.Month(), t.Day())
}

// projectInfo returns the folder name and current git branch for the given root.
// Both values are empty strings when root is "".
func projectInfo(root string) (name, branch string) {
	if root == "" {
		return "", ""
	}
	name = filepath.Base(root)
	branch = readGitBranch(root)
	return name, branch
}

// copyToClipboard writes text to the system clipboard and returns a copiedMsg.
func copyToClipboard(text string) tea.Cmd {
	return func() tea.Msg {
		cmd := clipboardCmd()
		if cmd == nil {
			return copiedMsg{} // no clipboard tool found — fail silently
		}
		cmd.Stdin = strings.NewReader(text)
		cmd.Run() //nolint:errcheck — clipboard failure is non-fatal
		return copiedMsg{}
	}
}

// clipboardCmd returns the appropriate clipboard write command for the current
// platform and display environment, or nil if none is available.
func clipboardCmd() *exec.Cmd {
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("pbcopy")
	default: // Linux and others
		// Wayland takes priority when WAYLAND_DISPLAY is set.
		if os.Getenv("WAYLAND_DISPLAY") != "" {
			if path, err := exec.LookPath("wl-copy"); err == nil {
				return exec.Command(path)
			}
		}
		// X11 fallbacks.
		if path, err := exec.LookPath("xclip"); err == nil {
			return exec.Command(path, "-selection", "clipboard")
		}
		if path, err := exec.LookPath("xsel"); err == nil {
			return exec.Command(path, "--clipboard", "--input")
		}
		return nil
	}
}

// readGitBranch reads the current branch from <root>/.git/HEAD.
// Returns "" when the repo or file cannot be read.
func readGitBranch(root string) string {
	data, err := os.ReadFile(filepath.Join(root, ".git", "HEAD"))
	if err != nil {
		return ""
	}
	s := strings.TrimSpace(string(data))
	// "ref: refs/heads/main" → "main"
	if strings.HasPrefix(s, "ref: refs/heads/") {
		return strings.TrimPrefix(s, "ref: refs/heads/")
	}
	// Detached HEAD — show abbreviated hash
	if len(s) >= 7 {
		return s[:7]
	}
	return s
}
