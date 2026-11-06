package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const testPort = 4799

func newTestServer(t *testing.T, files map[string]string) (*server, *docStore) {
	t.Helper()
	store := newFixture(t, files)
	s, err := newServer(repoInfo{Root: filepath.Dir(store.root), DocDir: store.root}, store, newHub(), testPort)
	if err != nil {
		t.Fatal(err)
	}
	return s, store
}

func request(s *server, method, target, host string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, target, nil)
	r.Host = host
	w := httptest.NewRecorder()
	s.ServeHTTP(w, r)
	return w
}

const goodHost = "127.0.0.1:4799"

func TestHostCheck(t *testing.T) {
	s, _ := newTestServer(t, map[string]string{"a.md": "x"})
	for host, want := range map[string]int{
		goodHost:         http.StatusOK,
		"localhost:4799": http.StatusOK,
		"evil.test":      http.StatusForbidden,
		"evil.test:4799": http.StatusForbidden,
		"127.0.0.1":      http.StatusForbidden,
		"127.0.0.1:1":    http.StatusForbidden,
	} {
		if got := request(s, http.MethodGet, "/api/tree", host).Code; got != want {
			t.Errorf("Host %q: %d, want %d", host, got, want)
		}
	}
}

func TestReadOnly(t *testing.T) {
	s, _ := newTestServer(t, map[string]string{"a.md": "x"})
	for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete} {
		for _, target := range []string{"/", "/api/doc?path=a.md"} {
			w := request(s, method, target, goodHost)
			if w.Code != http.StatusMethodNotAllowed {
				t.Errorf("%s %s: %d", method, target, w.Code)
			}
		}
	}
	if got := request(s, http.MethodHead, "/api/tree", goodHost).Code; got != http.StatusOK {
		t.Errorf("HEAD: %d", got)
	}
}

func TestSecurityHeadersOnEveryResponse(t *testing.T) {
	s, _ := newTestServer(t, map[string]string{"a.md": "x"})
	responses := []*httptest.ResponseRecorder{
		request(s, http.MethodGet, "/", goodHost),
		request(s, http.MethodGet, "/api/tree", goodHost),
		request(s, http.MethodGet, "/raw/missing.png", goodHost),
		request(s, http.MethodPost, "/", goodHost),
		request(s, http.MethodGet, "/", "evil.test"),
	}
	for i, w := range responses {
		if got := w.Header().Get("Content-Security-Policy"); got != contentSecurityPolicy {
			t.Errorf("response %d: CSP = %q", i, got)
		}
		if got := w.Header().Get("X-Content-Type-Options"); got != "nosniff" {
			t.Errorf("response %d: nosniff = %q", i, got)
		}
	}
}

func TestDocRoute(t *testing.T) {
	s, _ := newTestServer(t, map[string]string{
		"tickets/t1-a.md": "---\nid: t1\nstatus: ready\n---\n# Titre\n",
		"plain.md":        "# Sans frontmatter\n",
		"kanban.base":     "secret",
	})
	w := request(s, http.MethodGet, "/api/doc?path=tickets/t1-a.md", goodHost)
	if w.Code != http.StatusOK {
		t.Fatalf("status %d", w.Code)
	}
	var doc docResponse
	if err := json.Unmarshal(w.Body.Bytes(), &doc); err != nil {
		t.Fatal(err)
	}
	if fieldValue(doc.Fields, "status") != "ready" || !strings.Contains(doc.HTML, "<h1") || strings.Contains(doc.HTML, "status:") {
		t.Errorf("doc = %+v", doc)
	}
	w = request(s, http.MethodGet, "/api/doc?path=plain.md", goodHost)
	if !strings.Contains(w.Body.String(), `"fields":[]`) {
		t.Errorf("fields must be an empty list: %s", w.Body.String())
	}
	for _, path := range []string{"missing.md", "kanban.base", "../.env", "../../etc/passwd", "", "%2e%2e/.env"} {
		if got := request(s, http.MethodGet, "/api/doc?path="+path, goodHost).Code; got != http.StatusNotFound {
			t.Errorf("path %q: %d", path, got)
		}
	}
}

func TestRawRoute(t *testing.T) {
	s, store := newTestServer(t, map[string]string{
		"a.png": "png",
		"a.md":  "md",
		"a.js":  "js",
	})
	if err := os.Symlink(filepath.Join(filepath.Dir(store.root), ".env"), filepath.Join(store.root, "leak.png")); err != nil {
		t.Fatal(err)
	}
	w := request(s, http.MethodGet, "/raw/a.png", goodHost)
	if w.Code != http.StatusOK || w.Header().Get("Content-Type") != "image/png" {
		t.Errorf("image: %d %s", w.Code, w.Header().Get("Content-Type"))
	}
	for _, target := range []string{"/raw/a.md", "/raw/a.js", "/raw/leak.png", "/raw/missing.png", "/raw/..%2f.env", "/raw/%2e%2e/.env"} {
		if got := request(s, http.MethodGet, target, goodHost); got.Code == http.StatusOK {
			t.Errorf("%s served: %s", target, got.Body.String())
		}
	}
}

func TestTicketsRoute(t *testing.T) {
	s, _ := newTestServer(t, map[string]string{
		"tickets/t1-a.md":         "---\nid: t1\nstatus: ready\n---\n",
		"tickets/done/t2-b.md":    "---\nid: t2\nstatus: done\n---\n",
		"tickets/annexes/note.md": "---\nid: t3\nstatus: ready\n---\n",
	})
	var b board
	if err := json.Unmarshal(request(s, http.MethodGet, "/api/tickets", goodHost).Body.Bytes(), &b); err != nil {
		t.Fatal(err)
	}
	if got := columnIDs(b, "ready"); len(got) != 1 || got[0] != "t1" {
		t.Errorf("ready = %v, annexes/ must not count", got)
	}
	if got := columnIDs(b, "done"); len(got) != 1 || got[0] != "t2" {
		t.Errorf("done = %v", got)
	}
}

func TestTicketsRouteWithoutTicketsFolder(t *testing.T) {
	s, _ := newTestServer(t, map[string]string{"a.md": "x"})
	w := request(s, http.MethodGet, "/api/tickets", goodHost)
	if w.Code != http.StatusOK {
		t.Fatalf("status %d", w.Code)
	}
	var b board
	if err := json.Unmarshal(w.Body.Bytes(), &b); err != nil {
		t.Fatal(err)
	}
	if len(b.Columns) != len(statusOrder) {
		t.Errorf("%d columns", len(b.Columns))
	}
}

func TestWatchAnnouncesChangesInNewFolders(t *testing.T) {
	store := newFixture(t, map[string]string{"tickets/t1.md": "x"})
	h := newHub()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := watchDocs(ctx, store.root, h); err != nil {
		t.Fatal(err)
	}
	changes := h.subscribe()
	expect := func(what string) {
		t.Helper()
		select {
		case <-changes:
		case <-time.After(3 * time.Second):
			t.Fatalf("no event after %s", what)
		}
	}
	if err := os.WriteFile(filepath.Join(store.root, "tickets/t1.md"), []byte("y"), 0o644); err != nil {
		t.Fatal(err)
	}
	expect("a file change")

	newDir := filepath.Join(store.root, "story")
	if err := os.Mkdir(newDir, 0o755); err != nil {
		t.Fatal(err)
	}
	expect("a new folder")
	time.Sleep(50 * time.Millisecond)
	if err := os.WriteFile(filepath.Join(newDir, "a.md"), []byte("y"), 0o644); err != nil {
		t.Fatal(err)
	}
	expect("a file in the new folder")
}

func TestEventsStream(t *testing.T) {
	s, _ := newTestServer(t, map[string]string{"a.md": "x"})
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Host = goodHost
		s.ServeHTTP(w, r)
	}))
	defer ts.Close()
	defer s.hub.close()

	response, err := http.Get(ts.URL + "/api/events")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = response.Body.Close() }()
	if got := response.Header.Get("Content-Type"); got != "text/event-stream" {
		t.Errorf("Content-Type = %q", got)
	}
	s.hub.broadcast()
	buffer := make([]byte, 64)
	n, err := response.Body.Read(buffer)
	if err != nil && !errors.Is(err, io.EOF) {
		t.Fatal(err)
	}
	if !strings.Contains(string(buffer[:n]), "event: changed") {
		t.Errorf("got %q", buffer[:n])
	}
}
