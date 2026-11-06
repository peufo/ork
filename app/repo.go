package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// repoInfo locates the main repository, even when started from a worker's
// worktree: tickets only live there.
type repoInfo struct {
	Root      string
	CommonDir string
	DocDir    string
}

func findRepo(start string) (repoInfo, error) {
	out, err := exec.Command("git", "-C", start, "rev-parse", "--path-format=absolute", "--git-common-dir").Output()
	if err != nil {
		return repoInfo{}, fmt.Errorf("%s n'est pas dans un dépôt git", start)
	}
	common, err := filepath.EvalSymlinks(strings.TrimSpace(string(out)))
	if err != nil {
		return repoInfo{}, fmt.Errorf("dépôt git illisible : %w", err)
	}
	root := filepath.Dir(common)
	doc := filepath.Join(root, "doc")
	info, err := os.Stat(doc)
	if err != nil || !info.IsDir() {
		return repoInfo{}, errors.New("pas de dossier doc/ dans " + root + " : lance /ork:init dans ce projet")
	}
	return repoInfo{Root: root, CommonDir: common, DocDir: doc}, nil
}
