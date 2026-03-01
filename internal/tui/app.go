package tui

import (
	"fmt"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/taavtamm/contx/internal/store"
)

type screen int

const (
	screenList screen = iota
	screenForm
	screenConfirm
)

// App is the root Bubble Tea model.
type App struct {
	styles     *Styles
	themeIndex int
	ms         *store.MultiStore
	screen     screen
	list       ListModel
	form       FormModel
	confirm    ConfirmModel
	width      int
	height     int
}

// New creates the root TUI application.
func New(ms *store.MultiStore, theme Theme) (*App, error) {
	styles := NewStyles(theme)

	l, err := NewList(styles, ms, 80, 24)
	if err != nil {
		return nil, err
	}

	return &App{
		styles:     styles,
		themeIndex: ThemeIndex(theme),
		ms:         ms,
		screen:     screenList,
		list:       l,
		width:      80,
		height:     24,
	}, nil
}

func (a App) Init() tea.Cmd {
	return a.list.Init()
}

func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height

	case NewContextMsg:
		a.form = NewForm(a.styles, a.ms, nil, a.width, a.height)
		a.screen = screenForm
		return a, a.form.Init()

	case EditContextMsg:
		a.form = NewForm(a.styles, a.ms, msg.Ctx, a.width, a.height)
		a.screen = screenForm
		return a, a.form.Init()

	case ImportFileMsg:
		content, _ := os.ReadFile(msg.File.Path)
		a.form = NewImportForm(a.styles, a.ms, msg.File.Name, string(content), a.width, a.height)
		a.screen = screenForm
		return a, a.form.Init()

	case DeleteContextMsg:
		a.confirm = NewConfirm(a.styles, msg.Ctx.Name, msg.Ctx.Scope, a.width, a.height)
		a.screen = screenConfirm
		return a, a.confirm.Init()

	case FormSavedMsg:
		a.screen = screenList
		a.list.Reload()
		return a, nil

	case FormCancelledMsg:
		a.screen = screenList
		return a, nil

	case ConfirmedMsg:
		a.deleteContext(msg.Name, msg.Scope)
		a.screen = screenList
		a.list.Reload()
		return a, nil

	case ConfirmCancelledMsg:
		a.screen = screenList
		return a, nil

	case CycleThemeMsg:
		a.themeIndex = (a.themeIndex + 1) % len(AllThemes)
		newTheme := AllThemes[a.themeIndex]
		a.styles = NewStyles(newTheme)
		l, err := NewList(a.styles, a.ms, a.width, a.height)
		if err == nil {
			a.list = l
		}
		saveThemeConfig(newTheme.Name)
		return a, nil
	}

	switch a.screen {
	case screenList:
		var cmd tea.Cmd
		a.list, cmd = a.list.Update(msg)
		return a, cmd
	case screenForm:
		var cmd tea.Cmd
		a.form, cmd = a.form.Update(msg)
		return a, cmd
	case screenConfirm:
		var cmd tea.Cmd
		a.confirm, cmd = a.confirm.Update(msg)
		return a, cmd
	}

	return a, nil
}

func (a App) View() string {
	switch a.screen {
	case screenForm:
		return a.form.View()
	case screenConfirm:
		return a.confirm.View()
	default:
		return a.list.View()
	}
}

func saveThemeConfig(name string) {
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}
	dir := filepath.Join(home, ".contx")
	os.MkdirAll(dir, 0o755)
	os.WriteFile(filepath.Join(dir, "config.yaml"), []byte(fmt.Sprintf("theme: %s\n", name)), 0o644)
}

func (a *App) deleteContext(name string, scope store.Scope) {
	a.ms.Delete(scope, name) //nolint:errcheck
}
