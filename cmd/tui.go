package cmd

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"github.com/taavtamm/contx/internal/store"
	"github.com/taavtamm/contx/internal/tui"
)

var tuiCmd = &cobra.Command{
	Use:   "tui",
	Short: "Launch the TUI (default when no subcommand given)",
	RunE:  runTUI,
}

func runTUI(_ *cobra.Command, _ []string) error {
	ms, err := buildMultiStore()
	if err != nil {
		return err
	}

	theme := tui.ThemeByName(configTheme())
	app, err := tui.New(ms, theme)
	if err != nil {
		return fmt.Errorf("init TUI: %w", err)
	}

	p := tea.NewProgram(app, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "TUI error: %v\n", err)
		os.Exit(1)
	}
	return nil
}

// buildMultiStore constructs the global + project multi-store.
func buildMultiStore() (*store.MultiStore, error) {
	globalDir, err := store.GlobalDir()
	if err != nil {
		return nil, fmt.Errorf("find home dir: %w", err)
	}

	globalStore, err := store.NewFileStore(globalDir, store.ScopeGlobal)
	if err != nil {
		return nil, fmt.Errorf("init global store: %w", err)
	}

	wd, _ := os.Getwd()
	projectRoot := store.FindProjectRoot(wd)

	var projectStore store.Store
	if projectRoot != "" {
		projectStore, err = store.NewFileStore(store.ProjectDir(projectRoot), store.ScopeProject)
		if err != nil {
			return nil, fmt.Errorf("init project store: %w", err)
		}
	}

	return store.NewMultiStore(globalStore, projectStore, projectRoot), nil
}
