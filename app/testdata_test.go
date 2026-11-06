package main

import (
	"os"
	"path/filepath"
	"testing"
)

// newFixture writes files under a fresh doc/ directory and returns its store.
// A secret sits next to doc/, as a repository's .env would.
func newFixture(t *testing.T, files map[string]string) *docStore {
	t.Helper()
	root := t.TempDir()
	doc := filepath.Join(root, "doc")
	if err := os.MkdirAll(doc, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".env"), []byte("SECRET=1"), 0o644); err != nil {
		t.Fatal(err)
	}
	for rel, content := range files {
		full := filepath.Join(doc, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	store, err := newDocStore(doc)
	if err != nil {
		t.Fatal(err)
	}
	return store
}
