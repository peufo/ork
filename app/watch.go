package main

import (
	"context"
	"errors"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

const watchDebounce = 150 * time.Millisecond

// hub tells every open page that the documents changed.
type hub struct {
	mu          sync.Mutex
	subscribers map[chan struct{}]struct{}
	closed      chan struct{}
	closeOnce   sync.Once
}

func newHub() *hub {
	return &hub{subscribers: map[chan struct{}]struct{}{}, closed: make(chan struct{})}
}

func (h *hub) subscribe() chan struct{} {
	ch := make(chan struct{}, 1)
	h.mu.Lock()
	defer h.mu.Unlock()
	h.subscribers[ch] = struct{}{}
	return ch
}

func (h *hub) unsubscribe(ch chan struct{}) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.subscribers, ch)
}

// broadcast never blocks: a page that has a pending event will refetch anyway.
func (h *hub) broadcast() {
	h.mu.Lock()
	defer h.mu.Unlock()
	for ch := range h.subscribers {
		select {
		case ch <- struct{}{}:
		default:
		}
	}
}

func (h *hub) subscriberCount() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return len(h.subscribers)
}

func (h *hub) close() {
	h.closeOnce.Do(func() { close(h.closed) })
}

// watchDocs watches the given folders until ctx ends. fsnotify is not recursive,
// so each directory is added, including the ones that appear later. A folder that
// does not exist yet is awaited from its parent: a project's first story creates
// doc/ while the app is open.
func watchDocs(ctx context.Context, roots []string, h *hub) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	for _, root := range roots {
		if _, err := os.Stat(root); errors.Is(err, fs.ErrNotExist) {
			if err := watcher.Add(filepath.Dir(root)); err != nil {
				log.Printf("%s ne sera pas surveillé : %v", root, err)
			}
			continue
		}
		if err := addDirs(watcher, root); err != nil {
			_ = watcher.Close()
			return err
		}
	}
	go func() {
		defer func() { _ = watcher.Close() }()
		var debounce <-chan time.Time
		for {
			select {
			case <-ctx.Done():
				return
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				if strings.HasPrefix(filepath.Base(event.Name), ".") || !isWithin(roots, event.Name) {
					continue
				}
				if event.Has(fsnotify.Create) {
					if info, err := os.Lstat(event.Name); err == nil && info.IsDir() {
						if err := addDirs(watcher, event.Name); err != nil {
							log.Printf("surveillance de %s : %v", event.Name, err)
						}
					}
				}
				if debounce == nil {
					debounce = time.After(watchDebounce)
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Printf("surveillance des documents : %v", err)
			case <-debounce:
				debounce = nil
				h.broadcast()
			}
		}
	}()
	return nil
}

// isWithin ignores what a watched parent reports about its other children.
func isWithin(roots []string, name string) bool {
	for _, root := range roots {
		if name == root || strings.HasPrefix(name, root+string(filepath.Separator)) {
			return true
		}
	}
	return false
}

func addDirs(watcher *fsnotify.Watcher, root string) error {
	return filepath.WalkDir(root, func(p string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() {
			return nil
		}
		if p != root && strings.HasPrefix(entry.Name(), ".") {
			return filepath.SkipDir
		}
		return watcher.Add(p)
	})
}
