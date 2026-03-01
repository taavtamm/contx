package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var registerCmd = &cobra.Command{
	Use:   "register",
	Short: "Print the claude mcp add command for registering contx",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Run the following command to register contx with Claude Code:")
		fmt.Println()
		fmt.Println("  claude mcp add contx $(which contx) serve")
		fmt.Println()
		fmt.Println("Or for user-scoped registration (all projects):")
		fmt.Println()
		fmt.Println("  claude mcp add --scope user contx $(which contx) serve")
		fmt.Println()
		fmt.Println("After registration, restart Claude Code and type @contx:// in a conversation to use contexts.")
	},
}
