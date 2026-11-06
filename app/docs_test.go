package main

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestResolveConfinesToMounts(t *testing.T) {
	store := newFixture(t, map[string]string{
		"doc/ok.md":             "x",
		"doc/sub/ok.md":         "x",
		"doc/.obsidian/app.md":  "x",
		"doc/sub/.hidden/no.md": "x",
		".ork/config.md":        "x",
		".ork/tickets/t1-a.md":  "x",
		".ork/.secret/no.md":    "x",
		"src/foo.md":            "x",
		"document/near.md":      "x",
	})
	if err := os.Symlink(filepath.Join(store.root, ".env"), filepath.Join(store.root, "doc/link.md")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(store.root, filepath.Join(store.root, "doc/up")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(store.root, "doc/ok.md"), filepath.Join(store.root, "doc/inner.md")); err != nil {
		t.Fatal(err)
	}

	for _, rel := range []string{"doc/ok.md", "doc/sub/ok.md", "doc/inner.md", "doc/up/doc/ok.md", ".ork/config.md", ".ork/tickets/t1-a.md"} {
		if _, err := store.resolve(rel); err != nil {
			t.Errorf("%s refused: %v", rel, err)
		}
	}
	refused := []string{
		"", "..", "../.env", ".env", "src/foo.md", "document/near.md", "doc/../.env", "doc/sub/../ok.md", "./doc/ok.md",
		"/etc/passwd", "..\\.env", "doc/.obsidian/app.md", "doc/sub/.hidden/no.md", "doc/link.md", "doc/up/.env",
		"doc/up/src/foo.md", ".ork/.secret/no.md", "doc/missing.md", ".git/config",
	}
	for _, rel := range refused {
		if abs, err := store.resolve(rel); err == nil {
			t.Errorf("%q resolved to %s", rel, abs)
		}
	}
}

func TestTree(t *testing.T) {
	store := newFixture(t, map[string]string{
		"doc/vision.md":                "x",
		"doc/.obsidian/plugin.md":      "x",
		"doc/kanban.base":              "x",
		"doc/empty-of-markdown/a.json": "x",
		".ork/config.md":               "x",
		".ork/tickets/t10-b.md":        "x",
		".ork/tickets/t9-a.md":         "x",
		".ork/tickets/captures/a.png":  "x",
		".ork/tickets/done/t1-z.md":    "x",
		"src/not-a-document.md":        "x",
	})
	tree, err := store.tree()
	if err != nil {
		t.Fatal(err)
	}
	if got := names(tree); !reflect.DeepEqual(got, []string{"doc", ".ork"}) {
		t.Fatalf("root = %v", got)
	}
	if got := names(tree.Children[0]); !reflect.DeepEqual(got, []string{"vision.md"}) {
		t.Errorf("doc = %v", got)
	}
	ork := tree.Children[1]
	if ork.Path != ".ork" || !reflect.DeepEqual(names(ork), []string{"tickets", "config.md"}) {
		t.Errorf(".ork = %q %v", ork.Path, names(ork))
	}
	if got := names(ork.Children[0]); !reflect.DeepEqual(got, []string{"done", "t9-a.md", "t10-b.md"}) {
		t.Errorf("tickets = %v", got)
	}
	if got := ork.Children[1].Path; got != ".ork/config.md" {
		t.Errorf("config path = %q", got)
	}
}

func TestTreeWithCustomOrMissingDocsDir(t *testing.T) {
	store := newFixture(t, map[string]string{"site/guide/a.md": "x", ".ork/config.md": "x"})

	custom, err := newDocStore(store.root, "site/guide")
	if err != nil {
		t.Fatal(err)
	}
	tree, err := custom.tree()
	if err != nil {
		t.Fatal(err)
	}
	if got := names(tree); !reflect.DeepEqual(got, []string{"site/guide", ".ork"}) {
		t.Errorf("custom = %v", got)
	}
	if _, err := custom.resolve("site/guide/a.md"); err != nil {
		t.Errorf("custom docs_dir refused: %v", err)
	}

	missing, err := newDocStore(store.root, "nope")
	if err != nil {
		t.Fatal(err)
	}
	tree, err = missing.tree()
	if err != nil {
		t.Fatal(err)
	}
	if got := names(tree); !reflect.DeepEqual(got, []string{".ork"}) {
		t.Errorf("missing = %v", got)
	}
}

func names(node *treeNode) []string {
	result := []string{}
	for _, child := range node.Children {
		result = append(result, child.Name)
	}
	return result
}
