package store

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ErrNotFound is returned when a context does not exist.
var ErrNotFound = errors.New("context not found")

// FileStore implements Store backed by a directory of markdown files.
type FileStore struct {
	dir   string
	scope Scope
}

// NewFileStore creates a FileStore rooted at dir.
// The dir is created if it does not exist.
func NewFileStore(dir string, scope Scope) (*FileStore, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create store dir %s: %w", dir, err)
	}
	return &FileStore{dir: dir, scope: scope}, nil
}

func (fs *FileStore) Scope() Scope { return fs.scope }

func (fs *FileStore) List() ([]*Context, error) {
	entries, err := os.ReadDir(fs.dir)
	if err != nil {
		return nil, fmt.Errorf("list contexts: %w", err)
	}

	var contexts []*Context
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		name := strings.TrimSuffix(e.Name(), ".md")
		c, err := fs.Get(name)
		if err != nil {
			continue // skip malformed files
		}
		contexts = append(contexts, c)
	}
	return contexts, nil
}

func (fs *FileStore) Get(name string) (*Context, error) {
	path := fs.path(name)
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("read context %s: %w", name, err)
	}

	c, err := Unmarshal(data)
	if err != nil {
		return nil, fmt.Errorf("parse context %s: %w", name, err)
	}
	c.Scope = fs.scope
	c.FilePath = path
	return c, nil
}

func (fs *FileStore) Save(c *Context) error {
	data, err := c.Marshal()
	if err != nil {
		return err
	}
	if err := os.WriteFile(fs.path(c.Name), data, 0o644); err != nil {
		return fmt.Errorf("write context %s: %w", c.Name, err)
	}
	return nil
}

func (fs *FileStore) Delete(name string) error {
	err := os.Remove(fs.path(name))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return ErrNotFound
		}
		return fmt.Errorf("delete context %s: %w", name, err)
	}
	return nil
}

func (fs *FileStore) path(name string) string {
	return filepath.Join(fs.dir, name+".md")
}
