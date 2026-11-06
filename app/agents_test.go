package main

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

var agentNow = time.Date(2026, 11, 6, 12, 0, 0, 0, time.UTC)

// These lines copy the shape of real transcripts (Claude Code 2.1.314).
const (
	linePrompt  = `{"type":"user","timestamp":"2026-11-06T11:00:00Z","cwd":"/w","message":{"role":"user","content":"PLAN MODE\n\nid: t38"}}`
	lineReminds = `{"type":"user","timestamp":"2026-11-06T11:00:00Z","message":{"content":"<system-reminder>injected</system-reminder>"}}`
	lineNoise   = `{"type":"api-request-blob","timestamp":"2026-11-06T11:00:01Z","message":{"content":[{"type":"text","text":"PROMPT_COPY"}]}}`
	lineBash    = `{"type":"assistant","timestamp":"2026-11-06T11:00:02Z","cwd":"/w","message":{"content":[{"type":"thinking","thinking":"PRIVATE_THOUGHT"},{"type":"tool_use","id":"tool1","name":"Bash","input":{"command":"cat .env","description":"Show SVG assets"}}]}}`
	lineResult  = `{"type":"user","timestamp":"2026-11-06T11:00:03Z","message":{"content":[{"type":"tool_result","tool_use_id":"tool1","content":"SECRET=hunter2"}]}}`
	lineEdit    = `{"type":"assistant","timestamp":"2026-11-06T11:00:04Z","cwd":"/w","message":{"content":[{"type":"tool_use","id":"tool2","name":"Edit","input":{"file_path":"/w/src/a.ts","old_string":"OLD_CONTENT","new_string":"NEW_CONTENT"}}]}}`
)

type agentFile struct {
	id, session string
	meta        string
	lines       []string
	age         time.Duration
}

func writeAgentFiles(t *testing.T, dir string, files ...agentFile) {
	t.Helper()
	for _, file := range files {
		session := file.session
		if session == "" {
			session = "session1"
		}
		sub := filepath.Join(dir, session, "subagents")
		if err := os.MkdirAll(sub, 0o755); err != nil {
			t.Fatal(err)
		}
		base := filepath.Join(sub, "agent-"+file.id)
		if err := os.WriteFile(base+".meta.json", []byte(file.meta), 0o644); err != nil {
			t.Fatal(err)
		}
		body := strings.Join(file.lines, "\n")
		if body != "" {
			body += "\n"
		}
		if err := os.WriteFile(base+".jsonl", []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
		at := agentNow.Add(-file.age)
		if err := os.Chtimes(base+".jsonl", at, at); err != nil {
			t.Fatal(err)
		}
	}
}

func newTestAgents(t *testing.T, files ...agentFile) *agentStore {
	t.Helper()
	dir := t.TempDir()
	writeAgentFiles(t, dir, files...)
	a := newAgentStoreAt(dir)
	a.now = func() time.Time { return agentNow }
	return a
}

func openTicket(id, agent string) ticket {
	return ticket{ID: id, Agent: agent, column: "in-progress"}
}

func agentByID(t *testing.T, response agentsResponse, id string) agentSummary {
	t.Helper()
	for _, agent := range response.Agents {
		if agent.ID == id {
			return agent
		}
	}
	t.Fatalf("agent %s missing from %+v", id, response.Agents)
	return agentSummary{}
}

func TestClaudeProjectDirEncodesThePath(t *testing.T) {
	t.Setenv("CLAUDE_CONFIG_DIR", "/config")
	got := claudeProjectDir("/Users/me/dev/my_repo.x")
	if want := filepath.Join("/config", "projects", "-Users-me-dev-my-repo-x"); got != want {
		t.Errorf("got %s, want %s", got, want)
	}
}

func TestAgentStates(t *testing.T) {
	a := newTestAgents(t,
		agentFile{id: "working", meta: `{"description":"t1 a"}`, age: time.Minute},
		agentFile{id: "silent", meta: `{"description":"t2 a"}`, age: 50 * time.Minute},
		agentFile{id: "orphan", meta: `{"description":"t3 a"}`, age: 2 * time.Hour},
		agentFile{id: "ended", meta: `{"description":"t4 a","finished":true}`, age: time.Minute},
	)
	response := a.list([]ticket{openTicket("t1", ""), openTicket("t2", ""), openTicket("t3", ""), openTicket("t4", "")})
	for id, want := range map[string]agentState{
		"working": agentRunning,
		"silent":  agentRunning,
		"orphan":  agentInterrupted,
		"ended":   agentFinished,
	} {
		if got := agentByID(t, response, id).State; got != want {
			t.Errorf("%s: %s, want %s", id, got, want)
		}
	}
}

func TestAgentLinkToTicket(t *testing.T) {
	a := newTestAgents(t,
		agentFile{id: "byfield", meta: `{"description":"sans identifiant"}`},
		agentFile{id: "bytoken", meta: `{"description":"Plan t38 identité, voir aussi t1"}`},
		agentFile{id: "byslug", meta: `{"description":"Reprendre t2-bouton-envoi"}`},
		agentFile{id: "legacy", meta: `{"description":"T-0003 plan decks"}`},
		agentFile{id: "nolink", meta: `{"description":"t100 n'est pas t3000"}`},
	)
	response := a.list([]ticket{openTicket("t1", "byfield"), openTicket("t38", ""), openTicket("t2", ""), openTicket("T-0003", ""), openTicket("t10", "")})
	for id, want := range map[string]string{"byfield": "t1", "bytoken": "t38", "byslug": "t2", "legacy": "T-0003", "nolink": ""} {
		if got := agentByID(t, response, id).Ticket; got != want {
			t.Errorf("%s: ticket %q, want %q", id, got, want)
		}
	}
}

func TestAgentListKeepsActiveAndLastRun(t *testing.T) {
	a := newTestAgents(t,
		agentFile{id: "older", meta: `{"description":"t1 plan","finished":true}`, age: 3 * time.Hour},
		agentFile{id: "newer", meta: `{"description":"t1 exec","finished":true}`, age: time.Hour},
		agentFile{id: "busy", meta: `{"description":"t2 exec"}`, age: time.Minute},
		agentFile{id: "pastbusy", meta: `{"description":"t2 plan","finished":true}`, age: time.Hour},
		agentFile{id: "closed", meta: `{"description":"t3 exec","finished":true}`, age: time.Hour},
		agentFile{id: "scout", meta: `{"description":"Chercher le bouton"}`, age: time.Minute},
		agentFile{id: "oldscout", meta: `{"description":"Chercher ailleurs","finished":true}`, age: time.Minute},
		agentFile{id: "child", meta: `{"description":"t1 revue","parentAgentId":"newer"}`, age: 2 * time.Hour},
	)
	done := ticket{ID: "t3", column: statusDone}
	response := a.list([]ticket{openTicket("t1", ""), openTicket("t2", ""), done})
	got := map[string]bool{}
	for _, agent := range response.Agents {
		got[agent.ID] = true
	}
	want := map[string]bool{"newer": true, "busy": true, "scout": true}
	if len(got) != len(want) {
		t.Errorf("got %v, want %v", got, want)
	}
	for id := range want {
		if !got[id] {
			t.Errorf("%s missing from %v", id, got)
		}
	}
}

func TestAgentWithWorkingChildIsLive(t *testing.T) {
	a := newTestAgents(t,
		agentFile{id: "worker", meta: `{"description":"t1 exec","finished":true}`, age: time.Minute},
		agentFile{id: "explore", meta: `{"description":"revue","parentAgentId":"worker","toolUseId":"tool9"}`, age: time.Minute},
	)
	response := a.list([]ticket{openTicket("t1", "")})
	if len(response.Agents) != 1 || response.Agents[0].ID != "worker" || response.Agents[0].State != agentRunning {
		t.Errorf("agents = %+v", response.Agents)
	}
	detail, ok := a.detail("explore", []ticket{openTicket("t1", "")})
	if !ok || detail.Parent != "worker" || detail.Ticket != "t1" {
		t.Errorf("sub-agent inherits its parent's ticket: %+v", detail)
	}
}

func TestAgentActivityFollowsTools(t *testing.T) {
	a := newTestAgents(t, agentFile{id: "a", meta: `{"description":"t1 a"}`, lines: []string{linePrompt, lineReminds, lineNoise, lineBash}, age: time.Minute})
	tickets := []ticket{openTicket("t1", "")}
	agent := agentByID(t, a.list(tickets), "a")
	if agent.Activity != "Show SVG assets" || !agent.ToolActive {
		t.Errorf("tool running: %+v", agent)
	}
	if agent.Started != "2026-11-06T11:00:00Z" {
		t.Errorf("started = %q", agent.Started)
	}
	appendLines(t, a, "a", lineResult)
	agent = agentByID(t, a.list(tickets), "a")
	if agent.Activity != "Show SVG assets" || agent.ToolActive {
		t.Errorf("tool ended: %+v", agent)
	}
	appendLines(t, a, "a", lineEdit)
	agent = agentByID(t, a.list(tickets), "a")
	if agent.Activity != "src/a.ts" || !agent.ToolActive {
		t.Errorf("path relative to the worktree: %+v", agent)
	}
}

func appendLines(t *testing.T, a *agentStore, id string, lines ...string) {
	t.Helper()
	path := filepath.Join(a.dir, "session1", "subagents", "agent-"+id+".jsonl")
	file, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = file.Close() }()
	if _, err := file.WriteString(strings.Join(lines, "\n") + "\n"); err != nil {
		t.Fatal(err)
	}
}

// The transcript holds what the agent read and wrote: only labels may leave.
func TestAgentDetailNeverExposesContents(t *testing.T) {
	a := newTestAgents(t, agentFile{id: "a", meta: `{"description":"t1 a"}`, lines: []string{linePrompt, lineReminds, lineNoise, lineBash, lineResult, lineEdit}})
	detail, ok := a.detail("a", []ticket{openTicket("t1", "")})
	if !ok {
		t.Fatal("agent not found")
	}
	data, err := json.Marshal(detail)
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"hunter2", "OLD_CONTENT", "NEW_CONTENT", "PRIVATE_THOUGHT", "PROMPT_COPY", "injected", "cat .env"} {
		if strings.Contains(string(data), forbidden) {
			t.Errorf("%q leaked: %s", forbidden, data)
		}
	}
	kinds := []string{}
	for _, event := range detail.Events {
		kinds = append(kinds, event.Kind)
	}
	if got := strings.Join(kinds, ","); got != "message,tool,tool" {
		t.Errorf("events = %s", got)
	}
	if !detail.Events[1].Done || detail.Events[2].Done {
		t.Errorf("done flags: %+v", detail.Events)
	}
}

func TestAgentDetailLinksSubAgents(t *testing.T) {
	spawn := `{"type":"assistant","timestamp":"2026-11-06T11:00:02Z","message":{"content":[{"type":"tool_use","id":"tool9","name":"Agent","input":{"description":"Revue du diff","prompt":"PROMPT"}}]}}`
	a := newTestAgents(t,
		agentFile{id: "worker", meta: `{"description":"t1 exec"}`, lines: []string{spawn}},
		agentFile{id: "explore", meta: `{"parentAgentId":"worker","toolUseId":"tool9"}`},
	)
	detail, ok := a.detail("worker", nil)
	if !ok || len(detail.Events) != 1 {
		t.Fatalf("detail = %+v", detail)
	}
	if event := detail.Events[0]; event.Agent != "explore" || event.Label != "Revue du diff" {
		t.Errorf("event = %+v", event)
	}
}

func TestTranscriptReadsOnlyCompleteLines(t *testing.T) {
	path := filepath.Join(t.TempDir(), "a.jsonl")
	if err := os.WriteFile(path, []byte(linePrompt+"\n"+lineBash[:40]), 0o600); err != nil {
		t.Fatal(err)
	}
	transcript := newTranscript()
	if err := transcript.advance(path); err != nil {
		t.Fatal(err)
	}
	if len(transcript.events) != 1 {
		t.Fatalf("partial line consumed: %+v", transcript.events)
	}
	if err := os.WriteFile(path, []byte(linePrompt+"\n"+lineBash+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := transcript.advance(path); err != nil {
		t.Fatal(err)
	}
	if len(transcript.events) != 2 || transcript.events[1].Label != "Show SVG assets" {
		t.Errorf("events = %+v", transcript.events)
	}
	if err := os.WriteFile(path, []byte(lineEdit+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := transcript.advance(path); err != nil {
		t.Fatal(err)
	}
	if len(transcript.events) != 1 || transcript.events[0].Tool != "Edit" {
		t.Errorf("a shorter file restarts the reading: %+v", transcript.events)
	}
}

// A future Claude Code format must degrade the page, not break it.
func TestAgentToleratesUnknownFormats(t *testing.T) {
	a := newTestAgents(t,
		agentFile{id: "drift", meta: `{"description":"t1 a","effort":3,"finished":"soon"}`, lines: []string{"not json", `{"type":"assistant","message":{"content":42}}`, `{"type":"assistant","message":{"content":[{"type":"tool_use","name":"Bash","input":"weird"}]}}`}},
		agentFile{id: "partial", meta: `{"descrip`},
	)
	response := a.list([]ticket{openTicket("t1", "")})
	if agent := agentByID(t, response, "drift"); agent.Description != "t1 a" || agent.Activity != "Bash" {
		t.Errorf("agent = %+v", agent)
	}
	if len(response.Agents) != 1 {
		t.Errorf("a meta still being written is skipped: %+v", response.Agents)
	}
}

func TestShortToolNameAndLabels(t *testing.T) {
	if got := shortToolName("mcp__plugin_x__click"); got != "click" {
		t.Errorf("got %q", got)
	}
	if got := shortToolName("Bash"); got != "Bash" {
		t.Errorf("got %q", got)
	}
	long := strings.Repeat("é", eventLabelMax+10)
	if got := []rune(shorten("  a\n\n b  " + long)); len(got) != eventLabelMax+1 || string(got[:3]) != "a b" {
		t.Errorf("shorten = %d runes, %q", len(got), string(got[:3]))
	}
	for raw, want := range map[string]string{
		`{"command":"bun test\n--watch"}`:           "bun test --watch",
		`{"pattern":"foo.*"}`:                       "foo.*",
		`{"skill":"impeccable"}`:                    "impeccable",
		`{"description":"Doit primer","url":"x.y"}`: "Doit primer",
		`{"pageId":1}`:                              "",
	} {
		if got := toolLabel(json.RawMessage(raw), ""); got != want {
			t.Errorf("%s: %q, want %q", raw, got, want)
		}
	}
}

func TestAgentReportKeepsItsLines(t *testing.T) {
	report := `{"type":"assistant","timestamp":"2026-11-06T11:00:02Z","message":{"content":[{"type":"tool_use","id":"tool3","name":"SubagentHandback","input":{"message":"TICKET: t1\nSTATUS: DONE"}}]}}`
	a := newTestAgents(t, agentFile{id: "a", meta: `{"finished":true}`, lines: []string{report}})
	detail, ok := a.detail("a", nil)
	if !ok || len(detail.Events) != 1 {
		t.Fatalf("detail = %+v", detail)
	}
	if event := detail.Events[0]; event.Kind != eventText || event.Label != "TICKET: t1\nSTATUS: DONE" {
		t.Errorf("event = %+v", event)
	}
}

func TestAgentRoutes(t *testing.T) {
	s, _ := newTestServer(t, map[string]string{".ork/tickets/t1-a.md": "---\nid: t1\nstatus: in-progress\nagent: a1\n---\n"})
	dir := t.TempDir()
	writeAgentFiles(t, dir, agentFile{id: "a1", meta: `{"description":"x","agentType":"ork:worker"}`, lines: []string{linePrompt, lineBash, lineResult}})
	secret := filepath.Join(filepath.Dir(dir), "secret.jsonl")
	s.agents = newAgentStoreAt(dir)

	w := request(s, http.MethodGet, "/api/agents", goodHost)
	var list agentsResponse
	if err := json.Unmarshal(w.Body.Bytes(), &list); err != nil {
		t.Fatal(err)
	}
	if len(list.Agents) != 1 || list.Agents[0].Ticket != "t1" {
		t.Errorf("list = %+v", list)
	}
	if body := request(s, http.MethodGet, "/api/agents/a1", goodHost).Body.String(); strings.Contains(body, "hunter2") || !strings.Contains(body, "Show SVG assets") {
		t.Errorf("detail = %s", body)
	}
	for _, id := range []string{"missing", "..%2f..%2fsecret", "%2e%2e", secret} {
		if got := request(s, http.MethodGet, "/api/agents/"+id, goodHost).Code; got == http.StatusOK {
			t.Errorf("id %q answered 200", id)
		}
	}
	if got := request(s, http.MethodGet, "/api/agents", "evil.test").Code; got != http.StatusForbidden {
		t.Errorf("Host check: %d", got)
	}
}

func TestAgentRouteWithoutSessions(t *testing.T) {
	s, _ := newTestServer(t, map[string]string{"a.md": "x"})
	w := request(s, http.MethodGet, "/api/agents", goodHost)
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"agents":[]`) {
		t.Errorf("%d %s", w.Code, w.Body.String())
	}
}

func TestSignatureChangesWithActivity(t *testing.T) {
	a := newTestAgents(t, agentFile{id: "a", meta: `{"description":"t1"}`, lines: []string{linePrompt}, age: time.Minute})
	before := a.signature()
	if a.signature() != before {
		t.Error("signature must be stable")
	}
	appendLines(t, a, "a", lineBash)
	if a.signature() == before {
		t.Error("signature must follow the transcript")
	}
}
