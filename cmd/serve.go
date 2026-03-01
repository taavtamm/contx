package cmd

import (
	"github.com/mark3labs/mcp-go/server"
	"github.com/spf13/cobra"
	contxmcp "github.com/taavtamm/contx/internal/mcp"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Run the MCP stdio server (used by Claude Code)",
	Long: `Starts contx as an MCP server over stdio.
Register with Claude Code via:
  claude mcp add contx $(which contx) serve`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ms, err := buildMultiStore()
		if err != nil {
			return err
		}

		s := contxmcp.NewServer(ms)
		return server.ServeStdio(s)
	},
}
