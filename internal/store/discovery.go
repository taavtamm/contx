package store

import (
	"os"
	"path/filepath"
	"strings"
)

// FindProjectRoot walks up from dir looking for a .contx directory or a
// common project marker (.git, go.mod, package.json).
// Returns the directory containing .contx, or "" if none found.
// The home directory is excluded from the .contx check because ~/.contx
// is the global config dir, not a project root.
func FindProjectRoot(startDir string) string {
	home, _ := os.UserHomeDir()
	dir := startDir
	for {
		// If .contx exists here and it's not the home dir, it's a project root.
		// (~/.contx is the global config — not a project marker.)
		if dir != home {
			if _, err := os.Stat(filepath.Join(dir, ".contx")); err == nil {
				return dir
			}
		}

		// Stop at filesystem root.
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}

		// If we hit a project marker, treat this directory as the root
		// (even if .contx doesn't exist yet — it will be created on first save).
		for _, marker := range []string{".git", "go.mod", "package.json", "Cargo.toml", "pyproject.toml"} {
			if _, err := os.Stat(filepath.Join(dir, marker)); err == nil {
				return dir
			}
		}

		dir = parent
	}
	return ""
}

// GlobalDir returns ~/.contx/contexts.
func GlobalDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".contx", "contexts"), nil
}

// ProjectDir returns <root>/.contx/contexts for the given project root.
func ProjectDir(root string) string {
	return filepath.Join(root, ".contx", "contexts")
}

// UnmanagedFile is a project file that exists in the repo but has not yet been
// imported as a contx context.
type UnmanagedFile struct {
	Path    string // absolute path
	RelPath string // relative to project root (for display)
	Name    string // suggested context name derived from the filename
	Preview string // first 30 lines of content, for the TUI preview pane
}

// contextFilePaths lists the relative paths (from project root) that contx
// considers candidate context files.
var contextFilePaths = []string{
	"README.md",
	"AGENTS.md",
	"CLAUDE.md",
	"CONTRIBUTING.md",
	"CHANGELOG.md",
	"ARCHITECTURE.md",
	"DESIGN.md",
	"DEVELOPMENT.md",
	"CONVENTIONS.md",
	"NOTES.md",
	"HACKING.md",
	".github/CONTRIBUTING.md",
	".github/PULL_REQUEST_TEMPLATE.md",
	".cursor/rules",
	"docs/README.md",
	"docs/ARCHITECTURE.md",
	"docs/CONTRIBUTING.md",
}

// FindUnmanagedFiles returns project files that exist but are not already
// tracked as contx contexts (i.e. their derived name is not in managedNames).
// projectRoot must be a non-empty absolute path; callers should fall back to
// os.Getwd() when no project root is detected.
func FindUnmanagedFiles(projectRoot string, managedNames map[string]bool) []UnmanagedFile {
	if projectRoot == "" {
		return nil // safety guard; callers should provide a fallback
	}
	var files []UnmanagedFile
	for _, rel := range contextFilePaths {
		abs := filepath.Join(projectRoot, rel)
		if _, err := os.Stat(abs); err != nil {
			continue
		}
		name := deriveName(rel)
		if managedNames[name] {
			continue
		}
		files = append(files, UnmanagedFile{
			Path:    abs,
			RelPath: rel,
			Name:    name,
			Preview: readFilePreview(abs, 30),
		})
	}
	return files
}

// deriveName converts a relative path like "AGENTS.md" or ".github/CONTRIBUTING.md"
// into a context name like "agents" or "contributing".
func deriveName(relPath string) string {
	base := filepath.Base(relPath)
	name := strings.TrimSuffix(base, filepath.Ext(base))
	return strings.ToLower(name)
}

func readFilePreview(path string, maxLines int) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	lines := strings.SplitN(string(data), "\n", maxLines+1)
	if len(lines) > maxLines {
		lines = lines[:maxLines]
	}
	return strings.Join(lines, "\n")
}
