package cmd

import (
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var (
	flagTheme string
)

var rootCmd = &cobra.Command{
	Use:   "contx",
	Short: "Context management TUI + MCP server for Claude Code",
	Long: `contx manages reusable context snippets and exposes them as MCP resources
so you can @-mention them in Claude Code conversations.

Run without arguments to launch the TUI.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runTUI(cmd, args)
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&flagTheme, "theme", "", "theme name (tokyo-night|catppuccin|dracula|gruvbox)")

	rootCmd.AddCommand(serveCmd)
	rootCmd.AddCommand(tuiCmd)
	rootCmd.AddCommand(addCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(registerCmd)
}

// configTheme returns the theme name from --theme flag or config file.
func configTheme() string {
	if flagTheme != "" {
		return flagTheme
	}
	return loadConfigTheme()
}

type config struct {
	Theme string `yaml:"theme"`
}

func loadConfigTheme() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	data, err := os.ReadFile(filepath.Join(home, ".contx", "config.yaml"))
	if err != nil {
		return ""
	}
	var cfg config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return ""
	}
	return cfg.Theme
}
