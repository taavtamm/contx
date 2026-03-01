package cmd

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/taavtamm/contx/internal/store"
)

var (
	addGlobal      bool
	addDescription string
	addTags        string
)

var addCmd = &cobra.Command{
	Use:   "add <name>",
	Short: "Quick-add a context (reads body from stdin)",
	Args:  cobra.ExactArgs(1),
	Example: `  echo "Acme Corp info" | contx add company-info --global --desc "Company overview"
  contx add api-docs < api.md`,
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		if strings.ContainsAny(name, " /\\") {
			return fmt.Errorf("name must not contain spaces or slashes")
		}

		ms, err := buildMultiStore()
		if err != nil {
			return err
		}

		// Read body from stdin
		var bodyBytes []byte
		stat, _ := os.Stdin.Stat()
		if (stat.Mode() & os.ModeCharDevice) == 0 {
			bodyBytes, err = io.ReadAll(os.Stdin)
			if err != nil {
				return fmt.Errorf("read stdin: %w", err)
			}
		}

		var tagList []string
		for _, t := range strings.Split(addTags, ",") {
			t = strings.TrimSpace(t)
			if t != "" {
				tagList = append(tagList, t)
			}
		}

		scope := store.ScopeProject
		if addGlobal {
			scope = store.ScopeGlobal
		}

		now := time.Now().UTC()
		c := &store.Context{
			Name:        name,
			Description: addDescription,
			Tags:        tagList,
			Scope:       scope,
			Body:        string(bodyBytes),
			CreatedAt:   now,
			UpdatedAt:   now,
		}

		if err := ms.Save(c); err != nil {
			if errors.Is(err, store.ErrNoProjectRoot) {
				return fmt.Errorf("no project root found; use --global or run from a project directory")
			}
			return fmt.Errorf("save: %w", err)
		}

		fmt.Printf("Created context %q  (%s)\n", name, c.URI())
		return nil
	},
}

func init() {
	addCmd.Flags().BoolVar(&addGlobal, "global", false, "save to global scope (~/.contx/)")
	addCmd.Flags().StringVar(&addDescription, "desc", "", "short description")
	addCmd.Flags().StringVar(&addTags, "tags", "", "comma-separated tags")
}
