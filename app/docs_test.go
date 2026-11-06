package main

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestResolveConfinesToDoc(t *testing.T) {
	store := newFixture(t, map[string]string{
		"ok.md":             "x",
		"sub/ok.md":         "x",
		".obsidian/app.md":  "x",
		"sub/.hidden/no.md": "x",
	})
	secret := filepath.Join(filepath.Dir(store.root), ".env")
	if err := os.Symlink(secret, filepath.Join(store.root, "link.md")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Dir(store.root), filepath.Join(store.root, "up")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(store.root, "ok.md"), filepath.Join(store.root, "inner.md")); err != nil {
		t.Fatal(err)
	}

	for _, rel := range []string{"ok.md", "sub/ok.md", "inner.md", "up/doc/ok.md"} {
		if _, err := store.resolve(rel); err != nil {
			t.Errorf("%s refused: %v", rel, err)
		}
	}
	refused := []string{
		"", "..", "../.env", "sub/../../.env", "sub/../ok.md", "./ok.md", "/etc/passwd",
		"..\\.env", ".obsidian/app.md", "sub/.hidden/no.md", "link.md", "up/.env", "missing.md",
	}
	for _, rel := range refused {
		if abs, err := store.resolve(rel); err == nil {
			t.Errorf("%q resolved to %s", rel, abs)
		}
	}
}

func TestTree(t *testing.T) {
	store := newFixture(t, map[string]string{
		"ork.md":                   "x",
		"tickets/t10-b.md":         "x",
		"tickets/t9-a.md":          "x",
		"tickets/captures/a.png":   "x",
		"tickets/done/t1-z.md":     "x",
		".obsidian/plugin.md":      "x",
		"kanban.base":              "x",
		"empty-of-markdown/a.json": "x",
	})
	tree, err := store.tree()
	if err != nil {
		t.Fatal(err)
	}
	if got := names(tree); !reflect.DeepEqual(got, []string{"tickets", "ork.md"}) {
		t.Errorf("root = %v", got)
	}
	if got := names(tree.Children[0]); !reflect.DeepEqual(got, []string{"done", "t9-a.md", "t10-b.md"}) {
		t.Errorf("tickets = %v", got)
	}
}

func names(node *treeNode) []string {
	result := []string{}
	for _, child := range node.Children {
		result = append(result, child.Name)
	}
	return result
}
