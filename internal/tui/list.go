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
	"github.com/charmbracelet/x/ansi"
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

func (u unmanagedItem) FilterValue() string { return u.File.RelPath }
func (u unmanagedItem) Title() string       { return u.File.Name }
func (u unmanagedItem) Description() string { return u.File.RelPath }

// listDelegate renders items in the list.
type listDelegate struct{ styles *Styles }

func (d listDelegate) Height() int                              { return 1 }
func (d listDelegate) Spacing() int                             { return 0 }
func (d listDelegate) Update(_ tea.Msg, _ *blist.Model) tea.Cmd { return nil }

func (d listDelegate) Render(w io.Writer, m blist.Model, index int, item blist.Item) {
	s := d.styles
	iw := m.Width() - 3 // keep a 1-col safety gutter for terminal width mismatches
	if iw < 1 {
		iw = 1
	}
	rowW := m.Width()
	if rowW < 1 {
		rowW = 1
	}
	// truncPad truncates text at iw chars then pads to exactly iw — guarantees one line.
	truncPad := func(text string) string {
		t := ansi.Truncate(sanitizeForTerminal(text), iw, "")
		return lipgloss.NewStyle().Width(iw).Render(t)
	}
	// paintFullRow guarantees we overwrite the entire terminal row and don't leave
	// stale glyphs from previously longer lines.
	paintFullRow := func(rendered string) string {
		return lipgloss.NewStyle().Width(rowW).MaxWidth(rowW).Render(rendered)
	}
	switch it := item.(type) {
	case sectionHeader:
		cell := truncPad(it.Label)
		io.WriteString(w, paintFullRow(s.SectionHeader.Render(cell)))
	case branchItem:
		cell := truncPad("⎇ " + it.Branch)
		row := lipgloss.NewStyle().Foreground(s.Theme.Green).Padding(0, 1).Render(cell)
		io.WriteString(w, paintFullRow(row))
	case unmanagedItem:
		// Path in theme subtle, filename in theme yellow (Tokyo Night and others provide Yellow).
		dir := filepath.Dir(it.File.RelPath)
		pathPart := ""
		if dir != "." {
			pathPart = dir + "/"
		}
		namePart := filepath.Base(it.File.RelPath)
		pathSanitized := sanitizeForTerminal(pathPart)
		nameSanitized := sanitizeForTerminal(namePart)
		fullRaw := pathSanitized + nameSanitized
		truncatedRaw := ansi.Truncate(fullRaw, iw, "")
		var cell string
		if pathPart == "" {
			cell = lipgloss.NewStyle().Foreground(s.Theme.Yellow).Render(truncatedRaw)
		} else if len(truncatedRaw) <= len(pathSanitized) {
			cell = lipgloss.NewStyle().Foreground(s.Theme.Subtle).Render(truncatedRaw)
		} else {
			pathStyled := lipgloss.NewStyle().Foreground(s.Theme.Subtle).Render(truncatedRaw[:len(pathSanitized)])
			nameStyled := lipgloss.NewStyle().Foreground(s.Theme.Yellow).Render(truncatedRaw[len(pathSanitized):])
			cell = pathStyled + nameStyled
		}
		cell = lipgloss.NewStyle().Width(iw).Render(cell)
		if index == m.Index() {
			io.WriteString(w, paintFullRow(s.ItemSelected.Render(cell)))
		} else {
			io.WriteString(w, paintFullRow(lipgloss.NewStyle().Padding(0, 1).Render(cell)))
		}
	case listItem:
		cell := truncPad(it.Ctx.Name)
		if index == m.Index() {
			io.WriteString(w, paintFullRow(s.ItemSelected.Render(cell)))
		} else {
			io.WriteString(w, paintFullRow(s.ItemNormal.Render(cell)))
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
	l.SetFilteringEnabled(true)
	l.DisableQuitKeybindings()
	l.SetShowPagination(false) // disable built-in pagination to avoid PerPage/TotalPages mismatch
	
	m := ListModel{
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
	}
	m.resize(w, h)
	return m, nil
}

func (m ListModel) Init() tea.Cmd { return nil }

func (m ListModel) Update(msg tea.Msg) (ListModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// When the list filter is active, only quit is handled here; all other keys go to the list.
		if m.list.SettingFilter() {
			if key.Matches(msg, m.keys.Quit) {
				return m, tea.Quit
			}
		} else {
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

			case key.Matches(msg, m.keys.Connect):
				if !m.focusRight {
					return m, func() tea.Msg { return OpenConnectViewMsg{} }
				}
			}
		}

	case copiedMsg:
		m.copied = true
		return m, tea.Tick(2*time.Second, func(time.Time) tea.Msg { return clearCopiedMsg{} })

	case clearCopiedMsg:
		m.copied = false
		return m, nil

	case tea.WindowSizeMsg:
		m.resize(msg.Width, msg.Height)
	}

	prevIdx := m.list.Index()
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	if newIdx := m.list.Index(); newIdx != prevIdx {
		m.skipHeaders(newIdx > prevIdx)
	}
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
	m.resize(m.width, m.height)
	m.previewScroll = 0
	return nil
}

// resize adjusts the list model dimensions to fit the pane layout.
func (m *ListModel) resize(w, h int) {
	m.width = w
	m.height = h
	if w < 1 || h < 1 {
		m.list.SetSize(0, 0)
		return
	}

	safeW := w
	if safeW > 2 {
		// Keep two columns free so terminals never auto-wrap our last column.
		safeW -= 2
	}
	// Left pane gets roughly half the safe width.
	// rightWidth is safeW - leftWidth - 3 (sep + padding).
	leftWidth := safeW/2 - 1
	if leftWidth < 1 {
		leftWidth = 1
	}

	// paneHeight excludes header(2) + footer(1) + divider(1) + margin(2) = 6 lines.
	paneHeight := h - 6
	if paneHeight < 1 {
		paneHeight = 1
	}

	m.list.SetSize(leftWidth, paneHeight)
}

func (m ListModel) View() string {
	s := m.styles
	w := m.width
	if w < 1 || m.height < 1 {
		return ""
	}
	// Recalculate layout dimensions to match resize().
	safeW := w
	if safeW > 2 {
		safeW -= 2
	}
	leftWidth := safeW/2 - 1
	if leftWidth < 1 {
		leftWidth = 1
	}
	rightWidth := safeW - leftWidth - 3
	if rightWidth < 1 {
		rightWidth = 1
	}
	paneHeight := m.height - 6
	if paneHeight < 1 {
		paneHeight = 1
	}

	// Header
	appTitleStr := s.AppTitle.Render("contx")
	themeStr := s.ThemeName.Render(s.Theme.Name)
	quitStr := s.HintDesc.Render("q quit")
	rightStr := themeStr + "  " + quitStr
	padW := safeW - lipgloss.Width(appTitleStr) - lipgloss.Width(rightStr) - 4
	if padW < 0 {
		padW = 0
	}
	middlePad := lipgloss.NewStyle().
		Width(padW).
		Render("")
	header := lipgloss.NewStyle().
		Padding(0, 1).
		Width(safeW).
		Render(lipgloss.JoinHorizontal(lipgloss.Top, appTitleStr, middlePad, rightStr))

	divider := s.Divider.Render(strings.Repeat("━", safeW))

	// Left pane: use list view directly, no extra wrapping.
	// truncateLines handles final safety check.
	leftPane := m.list.View()

	// Right pane: wrap preview content.
	rightPane := lipgloss.NewStyle().Width(rightWidth).MaxWidth(rightWidth).Height(paneHeight).MaxHeight(paneHeight).Render(
		truncateLines(m.renderPreview(rightWidth, paneHeight), rightWidth))

	// Vertical separator — accent colour when preview is focused
	sepStyle := s.PaneDivider
	if m.focusRight {
		sepStyle = s.Divider
	}
	sepLines := make([]string, paneHeight)
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
			s.HintKey.Render("/") + " " + s.HintDesc.Render("filter"),
			s.HintKey.Render("n") + " " + s.HintDesc.Render("new"),
			s.HintKey.Render("e") + " " + s.HintDesc.Render(editHint),
			s.HintKey.Render("d") + " " + s.HintDesc.Render("delete"),
			copyHint,
			s.HintKey.Render("tab") + " " + s.HintDesc.Render("→ preview"),
			s.HintKey.Render("M") + " " + s.HintDesc.Render("connect MCP"),
			s.HintKey.Render("t") + " " + s.HintDesc.Render("theme"),
		}
		if m.list.SettingFilter() || m.list.IsFiltered() {
			hints = append(hints, s.HintKey.Render("esc")+" "+s.HintDesc.Render("clear filter"))
		}
		if tp := m.list.Paginator.TotalPages; tp > 1 {
			hints = append(hints, s.HintDesc.Render(fmt.Sprintf("%d/%d", m.list.Paginator.Page+1, tp)))
		}
	}
	footer := s.Footer.Width(safeW).Render("  " + strings.Join(hints, "   "))
	// Exactly 6 non-content lines: header, divider, blank, divider, footer (no extra blank).
	// Total = 1+1+1+paneHeight+1+1 = paneHeight+6 = m.height so header/footer stay visible.
	view := lipgloss.JoinVertical(lipgloss.Left, header, divider, "", content, divider, footer)
	// Enforce exact line count and width so no wrap/overflow pushes header/footer off screen.
	return limitLineCount(view, m.height, safeW)
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
	wrappedBody := s.PreviewBody.Width(bodyWidth).MaxWidth(bodyWidth).Render(sanitizeForTerminal(c.Body))
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
	title := s.PreviewTitle.Render(sanitizeForTerminal(c.Name))
	if totalBodyLines > bodyAvail {
		pos := s.PreviewMeta.Render(fmt.Sprintf("%d/%d", scroll+bodyAvail, totalBodyLines))
		pad := width - lipgloss.Width(title) - lipgloss.Width(pos) - indentLen*2
		if pad > 0 {
			title = title + strings.Repeat(" ", pad) + pos
		}
	}
	titleDiv := s.PreviewDivider.Render(strings.Repeat("─", width-2))

	// Meta block.
	tagStr := sanitizeForTerminal(strings.Join(c.Tags, "  "))
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

	return lipgloss.NewStyle().Width(width).Height(height).Render(truncateLines(preview, width))
}

func (m ListModel) renderUnmanagedPreview(f store.UnmanagedFile, width, height int) string {
	s := m.styles
	const indent = "  "
	const indentLen = 2
	bodyWidth := width - indentLen - 1
	if bodyWidth < 10 {
		bodyWidth = 10
	}

	title := lipgloss.NewStyle().Foreground(s.Theme.Yellow).Bold(true).MaxWidth(bodyWidth).Render(sanitizeForTerminal(f.RelPath))
	if f.Truncated {
		tag := s.PreviewMeta.Render(" (preview)")
		title += tag
	}
	titleDiv := s.PreviewDivider.Render(strings.Repeat("─", width-2))
	hint := s.PreviewMeta.Render("e / ↩  import as context")

	// Word-wrap the preview content and apply scroll.
	wrappedBody := s.PreviewBody.Width(bodyWidth).MaxWidth(bodyWidth).Render(sanitizeForTerminal(f.Preview))
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
	return lipgloss.NewStyle().Width(width).Height(height).Render(truncateLines(preview, width))
}

func indentBlock(text, prefix string) string {
	lines := strings.Split(text, "\n")
	for i, l := range lines {
		lines[i] = prefix + l
	}
	return strings.Join(lines, "\n")
}

// truncateLines ensures every line in text is at most maxWidth display columns.
// This guards against CJK / emoji characters whose terminal width may disagree
// with lipgloss calculations.
func truncateLines(text string, maxWidth int) string {
	// Keep a one-column gutter to avoid auto-wrap in terminals that disagree
	// on East Asian / emoji width.
	if maxWidth > 1 {
		maxWidth--
	}
	lines := strings.Split(text, "\n")
	for i, l := range lines {
		lines[i] = ansi.Truncate(l, maxWidth, "")
	}
	return strings.Join(lines, "\n")
}

// limitLineCount returns at most maxLines lines of text, each truncated to maxWidth.
// Ensures the TUI never outputs more than the terminal height/width so header/footer stay visible.
func limitLineCount(text string, maxLines, maxWidth int) string {
	if maxLines < 1 || maxWidth < 1 {
		return ""
	}
	if maxWidth > 1 {
		maxWidth--
	}
	lines := strings.Split(text, "\n")
	if len(lines) > maxLines {
		lines = lines[:maxLines]
	}
	for i, l := range lines {
		lines[i] = ansi.Truncate(l, maxWidth, "")
	}
	return strings.Join(lines, "\n")
}

// sanitizeForTerminal normalizes text so raw control characters don't break TUI
// redraw/layout in terminal output.
func sanitizeForTerminal(text string) string {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")
	text = strings.ReplaceAll(text, "\t", "    ")
	return strings.Map(func(r rune) rune {
		if r == '\n' {
			return r
		}
		if r < 0x20 || r == 0x7f {
			return -1
		}
		return r
	}, text)
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
