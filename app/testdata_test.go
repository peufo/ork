package main

import (
	"os"
	"path/filepath"
	"testing"
)

// newFixture writes files in a fresh repository, under paths relative to its root,
// and returns its store on doc/ and .ork/. A secret sits at the root, as a
// repository's .env would.
func newFixture(t *testing.T, files map[string]string) *docStore {
	t.Helper()
	root := t.TempDir()
	for _, dir := range []string{defaultDocsDir, orkDir} {
		if err := os.MkdirAll(filepath.Join(root, dir), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, ".env"), []byte("SECRET=1"), 0o644); err != nil {
		t.Fatal(err)
	}
	for rel, content := range files {
		full := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	store, err := newDocStore(root, defaultDocsDir)
	if err != nil {
		t.Fatal(err)
	}
	return store
}
