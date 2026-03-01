package store

import (
	"errors"
	"fmt"
)

// ErrNoProjectRoot is returned when a project-scoped operation is attempted
// but no project root directory was found.
var ErrNoProjectRoot = errors.New("no project root found")

// Store is the interface for reading and writing contexts.
type Store interface {
	List() ([]*Context, error)
	Get(name string) (*Context, error)
	Save(c *Context) error
	Delete(name string) error
	Scope() Scope
}

// MultiStore combines global and optional project stores into a single List/Get view.
type MultiStore struct {
	Global      Store
	Project     Store  // may be nil if no project root found
	ProjectRoot string // absolute path of the project root, or ""
}

func NewMultiStore(global Store, project Store, projectRoot string) *MultiStore {
	return &MultiStore{Global: global, Project: project, ProjectRoot: projectRoot}
}

// List returns all contexts from both stores (global first, then project).
func (m *MultiStore) List() ([]*Context, error) {
	all, err := m.Global.List()
	if err != nil {
		return nil, err
	}
	if m.Project != nil {
		proj, err := m.Project.List()
		if err != nil {
			return nil, err
		}
		all = append(all, proj...)
	}
	return all, nil
}

// Get looks up a context by scope and name.
func (m *MultiStore) Get(scope Scope, name string) (*Context, error) {
	switch scope {
	case ScopeGlobal:
		return m.Global.Get(name)
	case ScopeProject:
		if m.Project == nil {
			return nil, ErrNotFound
		}
		return m.Project.Get(name)
	default:
		return nil, ErrNotFound
	}
}

// Save writes the context to the store matching c.Scope.
func (m *MultiStore) Save(c *Context) error {
	switch c.Scope {
	case ScopeGlobal:
		return m.Global.Save(c)
	case ScopeProject:
		if m.Project == nil {
			return ErrNoProjectRoot
		}
		return m.Project.Save(c)
	default:
		return fmt.Errorf("unknown scope: %s", c.Scope)
	}
}

// Delete removes the named context from the store matching scope.
func (m *MultiStore) Delete(scope Scope, name string) error {
	switch scope {
	case ScopeGlobal:
		return m.Global.Delete(name)
	case ScopeProject:
		if m.Project == nil {
			return ErrNotFound
		}
		return m.Project.Delete(name)
	default:
		return ErrNotFound
	}
}
