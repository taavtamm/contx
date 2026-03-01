package store

import (
	"io/fs"
	"os"
	"path/filepath"
	"sort"
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

// skipDirs are directory names that are never walked for context files.
var skipDirs = map[string]bool{
	".git":         true,
	".contx":       true,
	".claude":      true,
	"node_modules": true,
	"vendor":       true,
	"dist":         true,
	"build":        true,
	"__pycache__":  true,
	".venv":        true,
	"venv":         true,
	"target":       true, // Rust
}

// FindUnmanagedFiles walks the entire project tree and returns all .md/.mdc
// files that have not already been imported as a contx context.
func FindUnmanagedFiles(projectRoot string, managedNames map[string]bool) []UnmanagedFile {
	if projectRoot == "" {
		return nil
	}
	var files []UnmanagedFile
	filepath.WalkDir(projectRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if skipDirs[d.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		ext := strings.ToLower(filepath.Ext(d.Name()))
		if ext != ".md" && ext != ".mdc" {
			return nil
		}
		rel, _ := filepath.Rel(projectRoot, path)
		name := deriveName(rel)
		if managedNames[name] {
			return nil
		}
		files = append(files, UnmanagedFile{
			Path:    path,
			RelPath: rel,
			Name:    name,
			Preview: readFilePreview(path, 30),
		})
		return nil
	})

	// Sort by depth (shallow first), then alphabetically within the same depth.
	sort.Slice(files, func(i, j int) bool {
		di := strings.Count(files[i].RelPath, string(filepath.Separator))
		dj := strings.Count(files[j].RelPath, string(filepath.Separator))
		if di != dj {
			return di < dj
		}
		return files[i].RelPath < files[j].RelPath
	})

	return files
}

// deriveName converts a relative path into a context name.
// "README.md" → "readme", "docs/ARCH.md" → "docs-arch",
// ".cursor/commands/foo.mdc" → "cursor-commands-foo"
func deriveName(relPath string) string {
	name := strings.TrimSuffix(relPath, filepath.Ext(relPath))
	name = strings.ReplaceAll(name, string(filepath.Separator), "-")
	name = strings.ReplaceAll(name, ".", "-")
	name = strings.ToLower(strings.Trim(name, "-"))
	return name
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
