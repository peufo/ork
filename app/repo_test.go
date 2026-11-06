package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func writeRepoFile(t *testing.T, root, rel, content string) {
	t.Helper()
	full := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func newGitRepo(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	if out, err := exec.Command("git", "-C", root, "init", "-q").CombinedOutput(); err != nil {
		t.Fatalf("git init: %v %s", err, out)
	}
	return root
}

func TestReadDocsDir(t *testing.T) {
	cases := []struct {
		name, config, want string
		invalid            bool
	}{
		{name: "no config", want: "doc"},
		{name: "no key", config: "---\nmax_parallel: 1\n---\n", want: "doc"},
		{name: "empty", config: "---\ndocs_dir:\n---\n", want: "doc"},
		{name: "no frontmatter", config: "# Notes\n", want: "doc"},
		{name: "trailing slash", config: "---\ndocs_dir: doc/\n---\n", want: "doc"},
		{name: "other", config: "---\ndocs_dir: docs\n---\n", want: "docs"},
		{name: "nested", config: "---\ndocs_dir: site/guide/\n---\n", want: "site/guide"},
		{name: "repository", config: "---\ndocs_dir: .\n---\n", invalid: true},
		{name: "parent", config: "---\ndocs_dir: ../other\n---\n", invalid: true},
		{name: "escaping", config: "---\ndocs_dir: doc/../../other\n---\n", invalid: true},
		{name: "absolute", config: "---\ndocs_dir: /etc\n---\n", invalid: true},
		{name: "hidden", config: "---\ndocs_dir: .git\n---\n", invalid: true},
		{name: "hidden inside", config: "---\ndocs_dir: doc/.private\n---\n", invalid: true},
		{name: "backslash", config: "---\ndocs_dir: doc\\sub\n---\n", invalid: true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			root := t.TempDir()
			if c.config != "" {
				writeRepoFile(t, root, configPath, c.config)
			}
			got, err := readDocsDir(root)
			if c.invalid {
				if err == nil {
					t.Errorf("accepted as %q", got)
				}
				return
			}
			if err != nil || got != c.want {
				t.Errorf("got %q, %v; want %q", got, err, c.want)
			}
		})
	}
}

func TestFindRepo(t *testing.T) {
	t.Run("not initialized", func(t *testing.T) {
		_, err := findRepo(newGitRepo(t))
		if err == nil || !strings.Contains(err.Error(), "/ork:init") {
			t.Errorf("err = %v", err)
		}
	})
	t.Run("old layout asks to migrate", func(t *testing.T) {
		root := newGitRepo(t)
		writeRepoFile(t, root, legacyConfig, "---\n---\n")
		_, err := findRepo(root)
		if err == nil || !strings.Contains(err.Error(), "migrer") {
			t.Errorf("err = %v", err)
		}
	})
	t.Run("initialized, from a subfolder", func(t *testing.T) {
		root := newGitRepo(t)
		writeRepoFile(t, root, configPath, "---\ndocs_dir: docs/\n---\n")
		writeRepoFile(t, root, "src/a.txt", "x")
		repo, err := findRepo(filepath.Join(root, "src"))
		if err != nil {
			t.Fatal(err)
		}
		if repo.DocsDir != "docs" {
			t.Errorf("DocsDir = %q", repo.DocsDir)
		}
	})
	t.Run("invalid docs_dir stops the app", func(t *testing.T) {
		root := newGitRepo(t)
		writeRepoFile(t, root, configPath, "---\ndocs_dir: ../other\n---\n")
		if _, err := findRepo(root); err == nil || !strings.Contains(err.Error(), "docs_dir") {
			t.Errorf("err = %v", err)
		}
	})
}
