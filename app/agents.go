package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

// Claude Code writes each subagent's transcript under
// <config>/projects/<encoded repo>/<session>/subagents/. The documentation gives
// this location but not a stable format, so everything here tolerates what it
// cannot read: an agent whose transcript no longer parses stays listed, without
// its activity.

const (
	// A killed session never writes `finished`: past this silence, the agent is
	// considered interrupted rather than running.
	agentInterruptedAfter = time.Hour
	agentPollEvery        = 2 * time.Second
	eventLabelMax         = 240
	eventTextMax          = 2000
	coordinatorPrefix     = "The coordinator sent a message while you were working:"
)

type agentState string

const (
	agentRunning     agentState = "running"
	agentFinished    agentState = "finished"
	agentInterrupted agentState = "interrupted"
)

type agentMeta struct {
	AgentType     string `json:"agentType"`
	Description   string `json:"description"`
	ToolUseID     string `json:"toolUseId"`
	ParentAgentID string `json:"parentAgentId"`
	Model         string `json:"model"`
	Effort        string `json:"effort"`
	Finished      bool   `json:"finished"`
}

type agentRecord struct {
	id      string
	path    string
	meta    agentMeta
	modTime time.Time
	size    int64
	state   agentState
}

type agentStore struct {
	dir         string
	now         func() time.Time
	mu          sync.Mutex
	transcripts map[string]*transcript
}

func newAgentStore(repoRoot string) *agentStore {
	return newAgentStoreAt(claudeProjectDir(repoRoot))
}

func newAgentStoreAt(dir string) *agentStore {
	return &agentStore{dir: dir, now: time.Now, transcripts: map[string]*transcript{}}
}

var notAlphanumeric = regexp.MustCompile(`[^a-zA-Z0-9]`)

func claudeProjectDir(repoRoot string) string {
	config := os.Getenv("CLAUDE_CONFIG_DIR")
	if config == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return ""
		}
		config = filepath.Join(home, ".claude")
	}
	return filepath.Join(config, "projects", notAlphanumeric.ReplaceAllString(repoRoot, "-"))
}

// scan only returns what it found in the directories themselves, so an id that
// comes from a request can never become a path.
func (a *agentStore) scan() []agentRecord {
	if a.dir == "" {
		return nil
	}
	sessions, err := os.ReadDir(a.dir)
	if err != nil {
		return nil
	}
	now := a.now()
	var records []agentRecord
	for _, session := range sessions {
		if !session.IsDir() {
			continue
		}
		dir := filepath.Join(a.dir, session.Name(), "subagents")
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			name, isMeta := strings.CutSuffix(entry.Name(), ".meta.json")
			id, isAgent := strings.CutPrefix(name, "agent-")
			if !isMeta || !isAgent || !entry.Type().IsRegular() {
				continue
			}
			if record, ok := readAgentRecord(dir, id, now); ok {
				records = append(records, record)
			}
		}
	}
	return records
}

func readAgentRecord(dir, id string, now time.Time) (agentRecord, bool) {
	base := filepath.Join(dir, "agent-"+id)
	data, err := os.ReadFile(base + ".meta.json")
	if err != nil {
		return agentRecord{}, false
	}
	var meta agentMeta
	// A field of an unexpected type stays empty. Anything else is a file still
	// being written, which the next poll will see complete.
	if err := json.Unmarshal(data, &meta); err != nil {
		var typeError *json.UnmarshalTypeError
		if !errors.As(err, &typeError) {
			return agentRecord{}, false
		}
	}
	path := base + ".jsonl"
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() {
		return agentRecord{}, false
	}
	state := agentRunning
	switch {
	case meta.Finished:
		state = agentFinished
	case now.Sub(info.ModTime()) > agentInterruptedAfter:
		state = agentInterrupted
	}
	return agentRecord{id: id, path: path, meta: meta, modTime: info.ModTime(), size: info.Size(), state: state}, true
}

// signature changes whenever the agents' state or their transcripts do.
func (a *agentStore) signature() string {
	var b strings.Builder
	for _, record := range a.scan() {
		fmt.Fprintf(&b, "%s %s %d %d\n", record.id, record.state, record.modTime.UnixNano(), record.size)
	}
	return b.String()
}

// agentEvent is one line of the activity feed. A tool's output never reaches it:
// the transcript holds everything the agent read, and the feed has no need for it.
type agentEvent struct {
	Kind   string `json:"kind"`
	Tool   string `json:"tool,omitempty"`
	Label  string `json:"label"`
	At     string `json:"at"`
	Done   bool   `json:"done,omitempty"`
	Failed bool   `json:"failed,omitempty"`
	Agent  string `json:"agent,omitempty"`
	toolID string
}

const (
	eventTool    = "tool"
	eventText    = "text"
	eventMessage = "message"
)

// transcript is read incrementally: transcripts reach several megabytes and only
// ever grow.
type transcript struct {
	offset  int64
	started string
	events  []agentEvent
	tools   map[string]int
}

func newTranscript() *transcript {
	return &transcript{tools: map[string]int{}}
}

func (t *transcript) advance(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }()
	info, err := file.Stat()
	if err != nil {
		return err
	}
	if info.Size() < t.offset {
		*t = *newTranscript()
	}
	if _, err := file.Seek(t.offset, io.SeekStart); err != nil {
		return err
	}
	reader := bufio.NewReader(file)
	for {
		line, err := reader.ReadBytes('\n')
		if errors.Is(err, io.EOF) {
			// A line still being written is read again, whole, next time.
			return nil
		}
		if err != nil {
			return err
		}
		t.offset += int64(len(line))
		t.consume(line)
	}
}

type contentItem struct {
	Type      string          `json:"type"`
	Text      string          `json:"text"`
	Name      string          `json:"name"`
	ID        string          `json:"id"`
	Input     json.RawMessage `json:"input"`
	ToolUseID string          `json:"tool_use_id"`
	IsError   bool            `json:"is_error"`
}

func (t *transcript) consume(raw []byte) {
	var line struct {
		Type      string `json:"type"`
		Timestamp string `json:"timestamp"`
		Cwd       string `json:"cwd"`
		Message   struct {
			Content json.RawMessage `json:"content"`
		} `json:"message"`
	}
	if err := json.Unmarshal(raw, &line); err != nil {
		return
	}
	if t.started == "" {
		t.started = line.Timestamp
	}
	if line.Type != "assistant" && line.Type != "user" {
		return
	}
	var text string
	if err := json.Unmarshal(line.Message.Content, &text); err == nil {
		if line.Type == "user" {
			t.consumePrompt(text, line.Timestamp)
		}
		return
	}
	var items []contentItem
	if err := json.Unmarshal(line.Message.Content, &items); err != nil {
		return
	}
	for _, item := range items {
		switch item.Type {
		case "tool_use":
			t.consumeToolUse(item, line.Cwd, line.Timestamp)
		case "tool_result":
			if index, ok := t.tools[item.ToolUseID]; ok {
				t.events[index].Done = true
				t.events[index].Failed = item.IsError
			}
		case "text":
			if line.Type == "assistant" {
				t.add(agentEvent{Kind: eventText, Label: agentWords(item.Text), At: line.Timestamp})
			}
		}
	}
}

// consumePrompt keeps what the orchestrator asked. System injections start with a
// tag and are not part of the conversation.
func (t *transcript) consumePrompt(text, at string) {
	if strings.HasPrefix(text, "<") {
		return
	}
	text = strings.TrimPrefix(text, coordinatorPrefix)
	t.add(agentEvent{Kind: eventMessage, Label: shorten(text), At: at})
}

func (t *transcript) consumeToolUse(item contentItem, cwd, at string) {
	// The worker's final report goes through this tool: it reads as text.
	if item.Name == "SubagentHandback" {
		var input struct {
			Message string `json:"message"`
		}
		_ = json.Unmarshal(item.Input, &input)
		t.add(agentEvent{Kind: eventText, Label: agentWords(input.Message), At: at})
		return
	}
	t.tools[item.ID] = len(t.events)
	t.add(agentEvent{Kind: eventTool, Tool: shortToolName(item.Name), Label: toolLabel(item.Input, cwd), At: at, toolID: item.ID})
}

func (t *transcript) add(event agentEvent) {
	t.events = append(t.events, event)
}

// shortToolName turns mcp__server__click into click.
func shortToolName(name string) string {
	if index := strings.LastIndex(name, "__"); index >= 0 && index+2 < len(name) {
		return name[index+2:]
	}
	return name
}

// toolLabel never reads the fields that carry file contents (Edit, Write).
func toolLabel(raw json.RawMessage, cwd string) string {
	var input struct {
		Description string `json:"description"`
		FilePath    string `json:"file_path"`
		Pattern     string `json:"pattern"`
		Query       string `json:"query"`
		URL         string `json:"url"`
		Skill       string `json:"skill"`
		Command     string `json:"command"`
	}
	_ = json.Unmarshal(raw, &input)
	if input.FilePath != "" && cwd != "" {
		input.FilePath = strings.TrimPrefix(input.FilePath, strings.TrimSuffix(cwd, "/")+"/")
	}
	for _, candidate := range []string{input.Description, input.FilePath, input.Pattern, input.Query, input.URL, input.Skill, input.Command} {
		if candidate != "" {
			return shorten(candidate)
		}
	}
	return ""
}

func shorten(text string) string {
	return truncate(strings.Join(strings.Fields(text), " "), eventLabelMax)
}

// agentWords keeps the line breaks of what the agent wrote: its report is the
// most useful line of the feed.
func agentWords(text string) string {
	return truncate(strings.TrimSpace(text), eventTextMax)
}

func truncate(text string, limit int) string {
	runes := []rune(text)
	if len(runes) > limit {
		return string(runes[:limit]) + "…"
	}
	return text
}

// current is what the agent is doing, or else the last thing it did.
func (t *transcript) current() (event agentEvent, ok bool) {
	var last *agentEvent
	for i := len(t.events) - 1; i >= 0; i-- {
		candidate := &t.events[i]
		if candidate.Kind != eventTool {
			continue
		}
		if !candidate.Done {
			return *candidate, true
		}
		if last == nil {
			last = candidate
		}
	}
	if last == nil {
		return agentEvent{}, false
	}
	return *last, true
}

type agentSummary struct {
	ID           string     `json:"id"`
	Type         string     `json:"type"`
	Description  string     `json:"description"`
	Model        string     `json:"model,omitempty"`
	Effort       string     `json:"effort,omitempty"`
	Ticket       string     `json:"ticket,omitempty"`
	State        agentState `json:"state"`
	Started      string     `json:"started,omitempty"`
	LastActivity string     `json:"lastActivity"`
	Activity     string     `json:"activity,omitempty"`
	ToolActive   bool       `json:"toolActive,omitempty"`
}

type agentsResponse struct {
	Agents []agentSummary `json:"agents"`
}

type agentDetail struct {
	agentSummary
	Parent string       `json:"parent,omitempty"`
	Events []agentEvent `json:"events"`
}

// agentGraph holds one scan and its links between agents and tickets.
type agentGraph struct {
	records  map[string]agentRecord
	children map[string][]string
	byField  map[string]string
	ticketOf map[string]string
	open     map[string]bool
}

func newAgentGraph(records []agentRecord, tickets []ticket) *agentGraph {
	g := &agentGraph{
		records:  make(map[string]agentRecord, len(records)),
		children: map[string][]string{},
		byField:  map[string]string{},
		ticketOf: make(map[string]string, len(tickets)),
		open:     map[string]bool{},
	}
	for _, t := range tickets {
		g.ticketOf[strings.ToLower(t.ID)] = t.ID
		if t.column != statusDone {
			g.open[t.ID] = true
		}
		// A ticket holds its current agent; older ones are found by description.
		if t.Agent != "" {
			g.byField[t.Agent] = t.ID
		}
	}
	for _, record := range records {
		g.records[record.id] = record
		if parent := record.meta.ParentAgentID; parent != "" {
			g.children[parent] = append(g.children[parent], record.id)
		}
	}
	return g
}

func (g *agentGraph) isRoot(record agentRecord) bool {
	_, hasParent := g.records[record.meta.ParentAgentID]
	return !hasParent
}

// rootOf climbs to the agent the lead launched. Sub-agents belong to its ticket.
func (g *agentGraph) rootOf(record agentRecord) agentRecord {
	seen := map[string]bool{record.id: true}
	for {
		parent, ok := g.records[record.meta.ParentAgentID]
		if !ok || seen[parent.id] {
			return record
		}
		seen[parent.id] = true
		record = parent
	}
}

// isLive is true while the agent or one of its sub-agents still works.
func (g *agentGraph) isLive(id string, seen map[string]bool) bool {
	if seen[id] {
		return false
	}
	seen[id] = true
	if g.records[id].state == agentRunning {
		return true
	}
	for _, child := range g.children[id] {
		if g.isLive(child, seen) {
			return true
		}
	}
	return false
}

var descriptionToken = regexp.MustCompile(`[\p{L}\p{N}_-]+`)

func (g *agentGraph) ticketFor(record agentRecord) string {
	root := g.rootOf(record)
	if id, ok := g.byField[root.id]; ok {
		return id
	}
	for _, token := range descriptionToken.FindAllString(root.meta.Description, -1) {
		lower := strings.ToLower(token)
		if id, ok := g.ticketOf[lower]; ok {
			return id
		}
		if id, ok := g.ticketOf[strings.ToLower(ticketID.FindString(token))]; ok {
			return id
		}
	}
	return ""
}

// list returns the agents at work, and for each open ticket with none, its last
// run. Sub-agents are reached through their parent's feed.
func (a *agentStore) list(tickets []ticket) agentsResponse {
	g := newAgentGraph(a.scan(), tickets)
	live := []agentRecord{}
	lastRun := map[string]agentRecord{}
	for _, record := range g.records {
		if !g.isRoot(record) {
			continue
		}
		if g.isLive(record.id, map[string]bool{}) {
			record.state = agentRunning
			live = append(live, record)
			continue
		}
		id := g.ticketFor(record)
		if !g.open[id] {
			continue
		}
		if previous, ok := lastRun[id]; !ok || record.modTime.After(previous.modTime) {
			lastRun[id] = record
		}
	}
	busy := map[string]bool{}
	for _, record := range live {
		busy[g.ticketFor(record)] = true
	}
	selected := live
	for id, record := range lastRun {
		if !busy[id] {
			selected = append(selected, record)
		}
	}
	sort.Slice(selected, func(i, j int) bool { return selected[i].modTime.After(selected[j].modTime) })

	a.mu.Lock()
	defer a.mu.Unlock()
	agents := make([]agentSummary, 0, len(selected))
	for _, record := range selected {
		agents = append(agents, a.summarize(record, g))
	}
	return agentsResponse{Agents: agents}
}

// summarize must be called with the lock held.
func (a *agentStore) summarize(record agentRecord, g *agentGraph) agentSummary {
	summary := agentSummary{
		ID:           record.id,
		Type:         record.meta.AgentType,
		Description:  record.meta.Description,
		Model:        record.meta.Model,
		Effort:       record.meta.Effort,
		Ticket:       g.ticketFor(record),
		State:        record.state,
		LastActivity: record.modTime.UTC().Format(time.RFC3339),
	}
	t := a.load(record)
	summary.Started = t.started
	if event, ok := t.current(); ok {
		summary.Activity = event.Label
		if summary.Activity == "" {
			summary.Activity = event.Tool
		}
		summary.ToolActive = !event.Done
	}
	return summary
}

// load must be called with the lock held. A read error keeps what is known.
func (a *agentStore) load(record agentRecord) *transcript {
	t, ok := a.transcripts[record.id]
	if !ok {
		t = newTranscript()
		a.transcripts[record.id] = t
	}
	_ = t.advance(record.path)
	return t
}

// detail returns one agent's feed, with its tool calls linked to the sub-agents
// they started.
func (a *agentStore) detail(id string, tickets []ticket) (agentDetail, bool) {
	g := newAgentGraph(a.scan(), tickets)
	record, ok := g.records[id]
	if !ok {
		return agentDetail{}, false
	}
	if record.state != agentRunning && g.isLive(id, map[string]bool{}) {
		record.state = agentRunning
	}
	childByTool := map[string]string{}
	for _, child := range g.children[id] {
		childByTool[g.records[child].meta.ToolUseID] = child
	}

	a.mu.Lock()
	defer a.mu.Unlock()
	detail := agentDetail{agentSummary: a.summarize(record, g), Events: []agentEvent{}}
	if !g.isRoot(record) {
		detail.Parent = record.meta.ParentAgentID
	}
	for _, event := range a.load(record).events {
		event.Agent = childByTool[event.toolID]
		detail.Events = append(detail.Events, event)
	}
	return detail, true
}

// watchAgents polls rather than using fsnotify: the watched tree also holds the
// main session's transcript, written constantly, and on macOS fsnotify would hold
// one descriptor per file. It only looks at the disk while a page is listening.
func watchAgents(ctx context.Context, a *agentStore, h *hub) {
	ticker := time.NewTicker(agentPollEvery)
	defer ticker.Stop()
	last := ""
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if h.subscriberCount() == 0 {
				continue
			}
			if current := a.signature(); current != last {
				last = current
				h.broadcast()
			}
		}
	}
}
