package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/taavtamm/contx/internal/store"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all contexts",
	RunE: func(cmd *cobra.Command, args []string) error {
		ms, err := buildMultiStore()
		if err != nil {
			return err
		}

		contexts, err := ms.List()
		if err != nil {
			return err
		}

		if len(contexts) == 0 {
			fmt.Println("No contexts found. Run `contx` to open the TUI and create one.")
			return nil
		}

		var global, project []*store.Context
		for _, c := range contexts {
			if c.Scope == store.ScopeGlobal {
				global = append(global, c)
			} else {
				project = append(project, c)
			}
		}

		if len(global) > 0 {
			fmt.Println("GLOBAL")
			for _, c := range global {
				printContext(c)
			}
		}
		if len(project) > 0 {
			if len(global) > 0 {
				fmt.Println()
			}
			fmt.Println("PROJECT")
			for _, c := range project {
				printContext(c)
			}
		}
		return nil
	},
}

func printContext(c *store.Context) {
	fmt.Printf("  %-30s  %s\n", c.Name, c.Description)
	fmt.Printf("  %s\n", c.URI())
}
