package main

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
)

// Paths are slash-separated and relative to the repository root.
const (
	orkDir         = ".ork"
	configPath     = orkDir + "/config.md"
	ticketsPath    = orkDir + "/tickets"
	legacyConfig   = "doc/ork.md"
	defaultDocsDir = "doc"
)

// repoInfo locates the main repository, even when started from a worker's
// worktree: tickets only live there.
type repoInfo struct {
	Root      string
	CommonDir string
	DocsDir   string
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
	info, err := os.Stat(filepath.Join(root, orkDir))
	if err != nil || !info.IsDir() {
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(legacyConfig))); err == nil {
			return repoInfo{}, errors.New("ce projet utilise encore " + legacyConfig + " : lance /ork:init pour le migrer vers " + orkDir + "/")
		}
		return repoInfo{}, errors.New("pas de dossier " + orkDir + "/ dans " + root + " : lance /ork:init dans ce projet")
	}
	docsDir, err := readDocsDir(root)
	if err != nil {
		return repoInfo{}, err
	}
	return repoInfo{Root: root, CommonDir: common, DocsDir: docsDir}, nil
}

// readDocsDir reads `docs_dir` from the project's config. The server only looks
// at it when it starts.
func readDocsDir(root string) (string, error) {
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(configPath)))
	if errors.Is(err, fs.ErrNotExist) {
		return defaultDocsDir, nil
	}
	if err != nil {
		return "", fmt.Errorf("%s illisible : %w", configPath, err)
	}
	fields, _, _ := parseFrontmatter(string(data))
	value := fieldValue(fields, "docs_dir")
	if value == "" {
		return defaultDocsDir, nil
	}
	return cleanDocsDir(value)
}

// cleanDocsDir refuses what would serve more than a documents folder: a path out
// of the repository, the repository itself, or a hidden folder like .git.
func cleanDocsDir(value string) (string, error) {
	invalid := fmt.Errorf("docs_dir invalide dans %s : %q (attendu : un sous-dossier du dépôt, non caché)", configPath, value)
	clean := path.Clean(value)
	if clean == "." || strings.Contains(value, "\\") || !filepath.IsLocal(filepath.FromSlash(clean)) {
		return "", invalid
	}
	for _, segment := range strings.Split(clean, "/") {
		if strings.HasPrefix(segment, ".") {
			return "", invalid
		}
	}
	return clean, nil
}
