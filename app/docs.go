package main

import (
	"errors"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"unicode"
)

var errOutsideStore = errors.New("chemin hors de .ork/ et du dossier de documents")

// imageTypes lists the only non-Markdown files the app serves.
var imageTypes = map[string]string{
	".png":  "image/png",
	".jpg":  "image/jpeg",
	".jpeg": "image/jpeg",
	".gif":  "image/gif",
	".webp": "image/webp",
	".avif": "image/avif",
	".svg":  "image/svg+xml",
}

// docStore gives confined access to the project's documents folder and to .ork/.
// Paths are slash-separated and relative to the repository root, like the ones
// the lead writes in tickets: doc/story/a.md, .ork/tickets/t1-a.md.
type docStore struct {
	root   string
	mounts []string
}

func newDocStore(root, docsDir string) (*docStore, error) {
	resolved, err := filepath.EvalSymlinks(root)
	if err != nil {
		return nil, err
	}
	return &docStore{root: resolved, mounts: []string{docsDir, orkDir}}, nil
}

// roots returns the absolute folders to watch. One of them may not exist yet.
func (d *docStore) roots() []string {
	roots := make([]string, 0, len(d.mounts))
	for _, mount := range d.mounts {
		abs := filepath.Join(d.root, filepath.FromSlash(mount))
		if resolved, err := filepath.EvalSymlinks(abs); err == nil {
			abs = resolved
		}
		roots = append(roots, abs)
	}
	return roots
}

// mountOf splits rel into its mount and what follows it.
func (d *docStore) mountOf(rel string) (mount, inner string, ok bool) {
	for _, mount := range d.mounts {
		if rel == mount {
			return mount, "", true
		}
		if inner, found := strings.CutPrefix(rel, mount+"/"); found {
			return mount, inner, true
		}
	}
	return "", "", false
}

// resolve turns a slash-separated path into an absolute one. It refuses what is
// outside the mounts, hidden segments inside them, `..` and symlinks that lead
// out of their mount.
func (d *docStore) resolve(rel string) (string, error) {
	if rel == "" || !filepath.IsLocal(filepath.FromSlash(rel)) || strings.Contains(rel, "\\") {
		return "", errOutsideStore
	}
	if path.Clean(rel) != rel {
		return "", errOutsideStore
	}
	mount, inner, ok := d.mountOf(rel)
	if !ok {
		return "", errOutsideStore
	}
	for _, segment := range strings.Split(inner, "/") {
		if strings.HasPrefix(segment, ".") {
			return "", errOutsideStore
		}
	}
	mountRoot, err := filepath.EvalSymlinks(filepath.Join(d.root, filepath.FromSlash(mount)))
	if err != nil {
		return "", err
	}
	abs, err := filepath.EvalSymlinks(filepath.Join(mountRoot, filepath.FromSlash(inner)))
	if err != nil {
		return "", err
	}
	inside, err := filepath.Rel(mountRoot, abs)
	if err != nil || !filepath.IsLocal(inside) {
		return "", errOutsideStore
	}
	return abs, nil
}

// isFile reports whether rel is a regular file inside a mount.
func (d *docStore) isFile(rel string) bool {
	abs, err := d.resolve(rel)
	if err != nil {
		return false
	}
	info, err := os.Stat(abs)
	return err == nil && info.Mode().IsRegular()
}

func isMarkdown(name string) bool {
	return strings.EqualFold(path.Ext(name), ".md")
}

func imageType(name string) (string, bool) {
	contentType, ok := imageTypes[strings.ToLower(path.Ext(name))]
	return contentType, ok
}

type treeNode struct {
	Name     string      `json:"name"`
	Path     string      `json:"path"`
	Dir      bool        `json:"dir"`
	Children []*treeNode `json:"children,omitempty"`
}

// tree lists the Markdown files of each mount. Directories without any Markdown
// are left out, and symlinked directories are not followed.
func (d *docStore) tree() (*treeNode, error) {
	top := &treeNode{Dir: true}
	for _, mount := range d.mounts {
		sub, err := d.walk(mount)
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, err
		}
		if len(sub.Children) > 0 {
			sub.Name = mount
			top.Children = append(top.Children, sub)
		}
	}
	return top, nil
}

func (d *docStore) walk(rel string) (*treeNode, error) {
	node := &treeNode{Name: path.Base(rel), Path: rel, Dir: true}
	entries, err := os.ReadDir(filepath.Join(d.root, filepath.FromSlash(rel)))
	if err != nil {
		return nil, err
	}
	for _, entry := range entries {
		name := entry.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}
		child := path.Join(rel, name)
		switch {
		case entry.IsDir():
			sub, err := d.walk(child)
			if err != nil {
				return nil, err
			}
			if len(sub.Children) > 0 {
				node.Children = append(node.Children, sub)
			}
		case isMarkdown(name) && d.isFile(child):
			node.Children = append(node.Children, &treeNode{Name: name, Path: child})
		}
	}
	sort.SliceStable(node.Children, func(i, j int) bool {
		a, b := node.Children[i], node.Children[j]
		if a.Dir != b.Dir {
			return a.Dir
		}
		return naturalLess(a.Name, b.Name)
	})
	return node, nil
}

// naturalLess orders names so that t9 comes before t10.
func naturalLess(a, b string) bool {
	ra, rb := []rune(strings.ToLower(a)), []rune(strings.ToLower(b))
	i, j := 0, 0
	for i < len(ra) && j < len(rb) {
		if unicode.IsDigit(ra[i]) && unicode.IsDigit(rb[j]) {
			startA, startB := i, j
			for i < len(ra) && unicode.IsDigit(ra[i]) {
				i++
			}
			for j < len(rb) && unicode.IsDigit(rb[j]) {
				j++
			}
			numA := strings.TrimLeft(string(ra[startA:i]), "0")
			numB := strings.TrimLeft(string(rb[startB:j]), "0")
			if len(numA) != len(numB) {
				return len(numA) < len(numB)
			}
			if numA != numB {
				return numA < numB
			}
			continue
		}
		if ra[i] != rb[j] {
			return ra[i] < rb[j]
		}
		i++
		j++
	}
	return len(ra)-i < len(rb)-j
}
