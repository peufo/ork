package main

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestParseFrontmatter(t *testing.T) {
	content := `---
id: t31
title: Outil MCP create_cards — ajouter : un lot
status: in-review
review:
depends_on: [t8, "t9"]
zones: [apps/web/src/lib/server/mcp, doc/specs/mcp.md]
owners:
  - a
  - b
# a comment
quoted: "x: y"
---

## Context
status: not-a-field
`
	fields, body, ok := parseFrontmatter(content)
	if !ok {
		t.Fatal("frontmatter not recognized")
	}
	if got := fieldValue(fields, "title"); got != "Outil MCP create_cards — ajouter : un lot" {
		t.Errorf("title = %q", got)
	}
	if got := fieldValue(fields, "review"); got != "" {
		t.Errorf("empty value = %q", got)
	}
	if got := fieldList(fields, "depends_on"); !reflect.DeepEqual(got, []string{"t8", "t9"}) {
		t.Errorf("inline list = %v", got)
	}
	if got := fieldList(fields, "owners"); !reflect.DeepEqual(got, []string{"a", "b"}) {
		t.Errorf("block list = %v", got)
	}
	if got := fieldValue(fields, "quoted"); got != "x: y" {
		t.Errorf("quoted = %q", got)
	}
	if got := fieldValue(fields, "status"); got != "in-review" {
		t.Errorf("status taken from the wrong place: %q", got)
	}
	if body != "\n## Context\nstatus: not-a-field\n" {
		t.Errorf("body = %q", body)
	}
}

func TestParseFrontmatterToleratesBrokenFiles(t *testing.T) {
	cases := map[string]string{
		"no frontmatter": "# Title\n",
		"never closed":   "---\nid: t1\nstatus: ready\n",
		"empty":          "",
	}
	for name, content := range cases {
		t.Run(name, func(t *testing.T) {
			fields, body, ok := parseFrontmatter(content)
			if ok || fields != nil || body != content {
				t.Errorf("got fields=%v body=%q ok=%v", fields, body, ok)
			}
		})
	}
}

func TestParseFrontmatterHandlesCRLFAndBOM(t *testing.T) {
	fields, _, ok := parseFrontmatter("\xef\xbb\xbf---\r\nstatus: ready\r\n---\r\nbody\r\n")
	if !ok || fieldValue(fields, "status") != "ready" {
		t.Errorf("fields=%v ok=%v", fields, ok)
	}
}

func columnIDs(b board, status string) []string {
	ids := []string{}
	for _, column := range b.Columns {
		if column.Status == status {
			for _, item := range column.Tickets {
				ids = append(ids, item.ID)
			}
		}
	}
	return ids
}

func hasColumn(b board, status string) bool {
	for _, column := range b.Columns {
		if column.Status == status {
			return true
		}
	}
	return false
}

func TestBuildBoardNeverHidesATicket(t *testing.T) {
	tickets := []ticket{
		newTicket("tickets/t1-a.md", "---\nid: t1\nstatus: ready\n---\n", false),
		newTicket("tickets/t2-b.md", "---\nid: t2\nstatus: wat\n---\n", false),
		newTicket("tickets/t3-c.md", "no frontmatter at all", false),
		newTicket("tickets/notes.md", "---\ntitle: Sans id\n---\n", false),
		newTicket("tickets/done/t4-d.md", "---\nid: t4\nstatus: in-review\n---\n", true),
	}
	b := buildBoard(tickets)
	if got := columnIDs(b, "other"); !reflect.DeepEqual(got, []string{"notes", "t2", "t3"}) {
		t.Errorf("other = %v", got)
	}
	if got := columnIDs(b, "done"); !reflect.DeepEqual(got, []string{"t4"}) {
		t.Errorf("a file under done/ belongs to done whatever its status: %v", got)
	}
	total := 0
	for _, column := range b.Columns {
		total += len(column.Tickets)
	}
	if total != len(tickets) {
		t.Errorf("%d tickets on the board, want %d", total, len(tickets))
	}
}

func TestBuildBoardColumns(t *testing.T) {
	b := buildBoard(nil)
	if hasColumn(b, "other") {
		t.Error("other column shown while empty")
	}
	if len(b.Columns) != len(statusOrder) {
		t.Errorf("%d columns, want %d", len(b.Columns), len(statusOrder))
	}
	for i, column := range b.Columns {
		if column.Status != statusOrder[i] {
			t.Errorf("column %d = %s", i, column.Status)
		}
		if _, attention := attentionStatuses[column.Status]; attention != column.Attention {
			t.Errorf("%s attention = %v", column.Status, column.Attention)
		}
	}
}

func TestBuildBoardOrder(t *testing.T) {
	open := func(id, priority, created string) ticket {
		return newTicket("tickets/"+id+".md", "---\nid: "+id+"\nstatus: ready\npriority: "+priority+"\ncreated: "+created+"\n---\n", false)
	}
	finished := func(id, updated string) ticket {
		return newTicket("tickets/done/"+id+".md", "---\nid: "+id+"\nupdated: "+updated+"\n---\n", true)
	}
	b := buildBoard([]ticket{
		open("t10", "normal", "2026-01-02"),
		open("t9", "normal", "2026-01-02"),
		open("t8", "low", "2026-01-01"),
		open("t7", "high", "2026-01-05"),
		open("t6", "normal", "2026-01-01"),
		finished("t1", "2026-01-01"),
		finished("t2", "2026-03-01"),
		finished("t3", "2026-02-01"),
	})
	if got := columnIDs(b, "ready"); !reflect.DeepEqual(got, []string{"t7", "t6", "t9", "t10", "t8"}) {
		t.Errorf("queue order = %v", got)
	}
	if got := columnIDs(b, "done"); !reflect.DeepEqual(got, []string{"t2", "t3", "t1"}) {
		t.Errorf("done order = %v", got)
	}
}

func TestBuildBoardWaitingDependencies(t *testing.T) {
	b := buildBoard([]ticket{
		newTicket("tickets/t3-c.md", "---\nid: t3\nstatus: ready\ndepends_on: [T1, t2, t99]\n---\n", false),
		newTicket("tickets/t2-b.md", "---\nid: t2\nstatus: ready\n---\n", false),
		newTicket("tickets/done/t1-a.md", "---\nid: t1\n---\n", true),
	})
	for _, column := range b.Columns {
		for _, item := range column.Tickets {
			if item.ID == "t3" && !reflect.DeepEqual(item.Waiting, []string{"t2", "t99"}) {
				t.Errorf("waiting = %v", item.Waiting)
			}
		}
	}
}

func TestNaturalLess(t *testing.T) {
	if !naturalLess("t9-a.md", "t10-a.md") || naturalLess("t10", "t9") {
		t.Error("numbers must compare as numbers")
	}
	if naturalLess("a", "a") {
		t.Error("not strict")
	}
	if !naturalLess("a", "ab") || !naturalLess("A", "b") {
		t.Error("prefix or case order")
	}
}

func TestDoneTicketsOfTheSameDayFollowTheirFileTime(t *testing.T) {
	store := newFixture(t, map[string]string{
		"tickets/done/t38-a.md": "---\nid: t38\nupdated: 2026-11-06\n---\n",
		"tickets/done/t26-b.md": "---\nid: t26\nupdated: 2026-11-06\n---\n",
		"tickets/done/t40-c.md": "---\nid: t40\nupdated: 2026-11-05\n---\n",
	})
	now := time.Date(2026, 11, 6, 12, 0, 0, 0, time.UTC)
	for name, at := range map[string]time.Time{
		"t38-a.md": now.Add(-time.Hour),
		"t26-b.md": now,
		"t40-c.md": now.Add(time.Hour),
	} {
		if err := os.Chtimes(filepath.Join(store.root, "tickets/done", name), at, at); err != nil {
			t.Fatal(err)
		}
	}
	got := strings.Join(columnIDs(buildBoard(store.loadTickets()), statusDone), " ")
	if want := "t26 t38 t40"; got != want {
		t.Errorf("done = %s, want %s: the date comes first, then the file time", got, want)
	}
}
