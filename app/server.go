package main

import (
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

//go:embed web
var webFiles embed.FS

const contentSecurityPolicy = "default-src 'none'; script-src 'self'; style-src 'self'; img-src 'self'; " +
	"connect-src 'self'; base-uri 'none'; form-action 'none'; frame-ancestors 'none'"

const eventsHeartbeat = 25 * time.Second

// server is read-only: it has no route that writes, and the lead stays the only
// writer of .ork/tickets/.
type server struct {
	repo     repoInfo
	store    *docStore
	agents   *agentStore
	render   *renderer
	hub      *hub
	agentHub *hub
	port     int
	mux      *http.ServeMux
}

func newServer(repo repoInfo, store *docStore, agents *agentStore, h, agentHub *hub, port int) (*server, error) {
	web, err := fs.Sub(webFiles, "web")
	if err != nil {
		return nil, err
	}
	s := &server{
		repo: repo, store: store, agents: agents, render: newRenderer(store),
		hub: h, agentHub: agentHub, port: port, mux: http.NewServeMux(),
	}
	s.mux.HandleFunc("GET /api/health", s.handleHealth)
	s.mux.HandleFunc("GET /api/tree", s.handleTree)
	s.mux.HandleFunc("GET /api/doc", s.handleDoc)
	s.mux.HandleFunc("GET /api/tickets", s.handleTickets)
	s.mux.HandleFunc("GET /api/agents", s.handleAgents)
	s.mux.HandleFunc("GET /api/agents/{id}", s.handleAgent)
	s.mux.HandleFunc("GET /api/events", s.handleEvents)
	s.mux.HandleFunc("GET /raw/{path...}", s.handleRaw)
	s.mux.Handle("GET /", http.FileServerFS(web))
	return s, nil
}

func (s *server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	header := w.Header()
	header.Set("Content-Security-Policy", contentSecurityPolicy)
	header.Set("X-Content-Type-Options", "nosniff")
	header.Set("Cross-Origin-Resource-Policy", "same-origin")
	header.Set("Referrer-Policy", "no-referrer")
	header.Set("Cache-Control", "no-store")

	// Blocks DNS rebinding: a foreign site pointing its domain at 127.0.0.1 would
	// otherwise read these private documents.
	if !s.allowedHost(r.Host) {
		http.Error(w, "hôte refusé", http.StatusForbidden)
		return
	}
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		header.Set("Allow", "GET, HEAD")
		http.Error(w, "lecture seule", http.StatusMethodNotAllowed)
		return
	}
	s.mux.ServeHTTP(w, r)
}

func (s *server) allowedHost(host string) bool {
	port := strconv.Itoa(s.port)
	return host == "127.0.0.1:"+port || host == "localhost:"+port
}

type healthResponse struct {
	Repo    string `json:"repo"`
	Version string `json:"version"`
	PID     int    `json:"pid"`
}

func (s *server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, healthResponse{Repo: s.repo.Root, Version: version, PID: os.Getpid()})
}

func (s *server) handleTree(w http.ResponseWriter, _ *http.Request) {
	tree, err := s.store.tree()
	if err != nil {
		log.Printf("arbre des documents : %v", err)
		http.Error(w, "documents illisibles", http.StatusInternalServerError)
		return
	}
	writeJSON(w, tree)
}

type docResponse struct {
	Path   string  `json:"path"`
	Fields []field `json:"fields"`
	HTML   string  `json:"html"`
}

func (s *server) handleDoc(w http.ResponseWriter, r *http.Request) {
	rel := r.URL.Query().Get("path")
	if !isMarkdown(rel) || !s.store.isFile(rel) {
		http.NotFound(w, r)
		return
	}
	content, err := s.store.read(rel)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	fields, body, _ := parseFrontmatter(content)
	rendered, err := s.render.render(rel, body)
	if err != nil {
		log.Printf("rendu de %q : %v", rel, err) //nolint:gosec // %q escapes control characters
		http.Error(w, "rendu impossible", http.StatusInternalServerError)
		return
	}
	if fields == nil {
		fields = []field{}
	}
	writeJSON(w, docResponse{Path: rel, Fields: fields, HTML: rendered})
}

func (s *server) handleTickets(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, buildBoard(s.store.loadTickets()))
}

func (s *server) handleAgents(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, s.agents.list(s.store.loadTickets()))
}

func (s *server) handleAgent(w http.ResponseWriter, r *http.Request) {
	detail, ok := s.agents.detail(r.PathValue("id"), s.store.loadTickets())
	if !ok {
		http.NotFound(w, r)
		return
	}
	writeJSON(w, detail)
}

func (s *server) handleRaw(w http.ResponseWriter, r *http.Request) {
	rel := r.PathValue("path")
	contentType, ok := imageType(rel)
	if !ok || !s.store.isFile(rel) {
		http.NotFound(w, r)
		return
	}
	abs, err := s.store.resolve(rel)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	file, err := os.Open(abs)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer func() { _ = file.Close() }()
	info, err := file.Stat()
	if err != nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", contentType)
	http.ServeContent(w, r, "", info.ModTime(), file)
}

func (s *server) handleEvents(w http.ResponseWriter, r *http.Request) {
	controller := http.NewResponseController(w)
	w.Header().Set("Content-Type", "text/event-stream")
	w.WriteHeader(http.StatusOK)
	if err := controller.Flush(); err != nil {
		return
	}
	changes := s.hub.subscribe()
	defer s.hub.unsubscribe(changes)
	agentChanges := s.agentHub.subscribe()
	defer s.agentHub.unsubscribe(agentChanges)
	heartbeat := time.NewTicker(eventsHeartbeat)
	defer heartbeat.Stop()
	for {
		var message string
		select {
		case <-r.Context().Done():
			return
		case <-s.hub.closed:
			return
		case <-changes:
			message = "event: changed\ndata: {}\n\n"
		case <-agentChanges:
			message = "event: agents\ndata: {}\n\n"
		case <-heartbeat.C:
			message = ": heartbeat\n\n"
		}
		if _, err := fmt.Fprint(w, message); err != nil {
			return
		}
		if err := controller.Flush(); err != nil {
			return
		}
	}
}

func writeJSON(w http.ResponseWriter, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if err := json.NewEncoder(w).Encode(value); err != nil && !errors.Is(err, http.ErrHandlerTimeout) && !strings.Contains(err.Error(), "broken pipe") {
		log.Printf("réponse JSON : %v", err)
	}
}
