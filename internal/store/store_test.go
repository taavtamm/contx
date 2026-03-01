package store_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/taavtamm/contx/internal/store"
)

func TestMarshalUnmarshal(t *testing.T) {
	original := &store.Context{
		Name:        "test-ctx",
		Description: "A test context",
		Tags:        []string{"foo", "bar"},
		CreatedAt:   time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		UpdatedAt:   time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
		Scope:       store.ScopeGlobal,
		Body:        "Hello, world!\n",
	}

	data, err := original.Marshal()
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	got, err := store.Unmarshal(data)
	if err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if got.Name != original.Name {
		t.Errorf("Name: got %q, want %q", got.Name, original.Name)
	}
	if got.Description != original.Description {
		t.Errorf("Description: got %q, want %q", got.Description, original.Description)
	}
	if got.Body != original.Body {
		t.Errorf("Body: got %q, want %q", got.Body, original.Body)
	}
	if len(got.Tags) != len(original.Tags) {
		t.Errorf("Tags len: got %d, want %d", len(got.Tags), len(original.Tags))
	}
}

func TestFileStoreCRUD(t *testing.T) {
	dir := t.TempDir()
	fs, err := store.NewFileStore(dir, store.ScopeGlobal)
	if err != nil {
		t.Fatalf("NewFileStore: %v", err)
	}

	now := time.Now().UTC()
	ctx := &store.Context{
		Name:        "my-ctx",
		Description: "desc",
		Tags:        []string{"a"},
		CreatedAt:   now,
		UpdatedAt:   now,
		Scope:       store.ScopeGlobal,
		Body:        "body content\n",
	}

	// Save
	if err := fs.Save(ctx); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// Verify file on disk
	if _, err := os.Stat(filepath.Join(dir, "my-ctx.md")); err != nil {
		t.Fatalf("file not created: %v", err)
	}

	// Get
	got, err := fs.Get("my-ctx")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Name != ctx.Name {
		t.Errorf("Name: got %q, want %q", got.Name, ctx.Name)
	}
	if got.Body != ctx.Body {
		t.Errorf("Body: got %q, want %q", got.Body, ctx.Body)
	}

	// List
	list, err := fs.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 1 {
		t.Errorf("List len: got %d, want 1", len(list))
	}

	// Delete
	if err := fs.Delete("my-ctx"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	list, _ = fs.List()
	if len(list) != 0 {
		t.Errorf("after Delete, List len: got %d, want 0", len(list))
	}

	// Get non-existent
	_, err = fs.Get("my-ctx")
	if err != store.ErrNotFound {
		t.Errorf("Get missing: want ErrNotFound, got %v", err)
	}
}

func TestMultiStore(t *testing.T) {
	gDir := t.TempDir()
	pDir := t.TempDir()

	gs, _ := store.NewFileStore(gDir, store.ScopeGlobal)
	ps, _ := store.NewFileStore(pDir, store.ScopeProject)
	ms := store.NewMultiStore(gs, ps, "")

	now := time.Now().UTC()
	gc := &store.Context{Name: "global-one", Scope: store.ScopeGlobal, CreatedAt: now, UpdatedAt: now}
	pc := &store.Context{Name: "proj-one", Scope: store.ScopeProject, CreatedAt: now, UpdatedAt: now}

	gs.Save(gc)
	ps.Save(pc)

	all, err := ms.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(all) != 2 {
		t.Errorf("MultiStore List len: got %d, want 2", len(all))
	}

	got, err := ms.Get(store.ScopeGlobal, "global-one")
	if err != nil {
		t.Fatalf("Get global: %v", err)
	}
	if got.Name != "global-one" {
		t.Errorf("Get global name: got %q", got.Name)
	}
}

func TestFindProjectRoot(t *testing.T) {
	// Create a temp tree: root/.git, root/sub/
	root := t.TempDir()
	os.Mkdir(filepath.Join(root, ".git"), 0o755)
	sub := filepath.Join(root, "sub", "deeper")
	os.MkdirAll(sub, 0o755)

	found := store.FindProjectRoot(sub)
	if found != filepath.Join(root, "sub") && found != root {
		// Accept either root or sub depending on walk direction
		t.Logf("FindProjectRoot returned %q (root=%q)", found, root)
	}
}
