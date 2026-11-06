package main

import (
	"os"
	"path"
	"regexp"
	"sort"
	"strings"
	"time"
)

// field is one frontmatter property, in file order.
type field struct {
	Key    string   `json:"key"`
	Value  string   `json:"value"`
	List   []string `json:"list,omitempty"`
	IsList bool     `json:"isList,omitempty"`
}

var fieldLine = regexp.MustCompile(`^([A-Za-z_][\w-]*):\s*(.*)$`)

// parseFrontmatter reads the subset of YAML that tickets use. It never fails:
// a file without a closed frontmatter is returned whole as its body, with
// ok=false, and an unquoted colon in a title stays part of the value.
func parseFrontmatter(content string) (fields []field, body string, ok bool) {
	content = strings.TrimPrefix(content, "\xef\xbb\xbf")
	normalized := strings.ReplaceAll(content, "\r\n", "\n")
	lines := strings.Split(normalized, "\n")
	if lines[0] != "---" {
		return nil, normalized, false
	}
	end := -1
	for i := 1; i < len(lines); i++ {
		if lines[i] == "---" {
			end = i
			break
		}
	}
	if end < 0 {
		return nil, normalized, false
	}
	for _, line := range lines[1:end] {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if item, isItem := strings.CutPrefix(trimmed, "- "); isItem && len(fields) > 0 {
			last := &fields[len(fields)-1]
			last.IsList = true
			last.List = append(last.List, unquote(strings.TrimSpace(item)))
			continue
		}
		match := fieldLine.FindStringSubmatch(line)
		if match == nil {
			continue
		}
		f := field{Key: match[1]}
		value := strings.TrimSpace(match[2])
		if strings.HasPrefix(value, "[") && strings.HasSuffix(value, "]") {
			f.IsList = true
			f.List = splitInlineList(value[1 : len(value)-1])
		} else {
			f.Value = unquote(value)
		}
		fields = append(fields, f)
	}
	return fields, strings.Join(lines[end+1:], "\n"), true
}

func unquote(value string) string {
	if len(value) >= 2 {
		first, last := value[0], value[len(value)-1]
		if first == last && (first == '"' || first == '\'') {
			return value[1 : len(value)-1]
		}
	}
	return value
}

func splitInlineList(inner string) []string {
	items := []string{}
	var current strings.Builder
	var quote rune
	flush := func() {
		if item := strings.TrimSpace(current.String()); item != "" {
			items = append(items, unquote(item))
		}
		current.Reset()
	}
	for _, r := range inner {
		switch {
		case quote != 0:
			if r == quote {
				quote = 0
			}
			current.WriteRune(r)
		case r == '"' || r == '\'':
			quote = r
			current.WriteRune(r)
		case r == ',':
			flush()
		default:
			current.WriteRune(r)
		}
	}
	flush()
	return items
}

func fieldValue(fields []field, key string) string {
	for _, f := range fields {
		if f.Key == key {
			return f.Value
		}
	}
	return ""
}

func fieldList(fields []field, key string) []string {
	for _, f := range fields {
		if f.Key == key {
			return f.List
		}
	}
	return nil
}

const (
	statusDone  = "done"
	statusOther = "other"
)

// statusOrder is the ticket cycle from skills/ticket, in board order.
var statusOrder = []string{"idea", "ready", "planning", "plan-review", "in-progress", "blocked", "in-review", statusDone}

// attentionStatuses wait for the developer, with the status line's icons.
var attentionStatuses = map[string]string{"blocked": "⏸", "plan-review": "📋", "in-review": "👀"}

var priorityRank = map[string]int{"high": 0, "normal": 1, "low": 2}

var ticketID = regexp.MustCompile(`^[tT]\d+`)

type ticket struct {
	ID        string   `json:"id"`
	Title     string   `json:"title"`
	Path      string   `json:"path"`
	Status    string   `json:"status"`
	Priority  string   `json:"priority"`
	Review    string   `json:"review"`
	Story     string   `json:"story"`
	Created   string   `json:"created"`
	Updated   string   `json:"updated"`
	DependsOn []string `json:"dependsOn"`
	Waiting   []string `json:"waiting"`
	Malformed bool     `json:"malformed"`
	Agent     string   `json:"-"`
	column    string
	modTime   time.Time
}

// newTicket never rejects a file: what it cannot read sends the ticket to the
// `other` column rather than hiding it.
func newTicket(rel, content string, inDone bool) ticket {
	fields, _, ok := parseFrontmatter(content)
	base := strings.TrimSuffix(path.Base(rel), path.Ext(rel))
	t := ticket{
		ID:        fieldValue(fields, "id"),
		Title:     fieldValue(fields, "title"),
		Path:      rel,
		Status:    fieldValue(fields, "status"),
		Priority:  fieldValue(fields, "priority"),
		Review:    fieldValue(fields, "review"),
		Story:     fieldValue(fields, "story"),
		Created:   fieldValue(fields, "created"),
		Updated:   fieldValue(fields, "updated"),
		DependsOn: fieldList(fields, "depends_on"),
		Waiting:   []string{},
		Malformed: !ok,
		Agent:     fieldValue(fields, "agent"),
	}
	if t.DependsOn == nil {
		t.DependsOn = []string{}
	}
	if t.Title == "" {
		t.Title = base
	}
	if t.ID == "" {
		t.ID = ticketID.FindString(base)
	}
	if t.ID == "" {
		t.ID = base
	}
	switch {
	case inDone:
		t.column = statusDone
	case isKnownStatus(t.Status):
		t.column = t.Status
	default:
		t.column = statusOther
	}
	return t
}

func isKnownStatus(status string) bool {
	for _, known := range statusOrder {
		if status == known {
			return true
		}
	}
	return false
}

type boardColumn struct {
	Status    string   `json:"status"`
	Icon      string   `json:"icon,omitempty"`
	Attention bool     `json:"attention,omitempty"`
	Tickets   []ticket `json:"tickets"`
}

type board struct {
	Columns []boardColumn `json:"columns"`
}

// buildBoard groups tickets by column. The `other` column only exists when a
// ticket needs it. Open tickets follow the lead's queue order (priority, then
// age); finished ones are newest first.
func buildBoard(tickets []ticket) board {
	done := map[string]bool{}
	for _, t := range tickets {
		if t.column == statusDone {
			done[strings.ToLower(t.ID)] = true
		}
	}
	grouped := map[string][]ticket{}
	for _, t := range tickets {
		for _, dep := range t.DependsOn {
			if !done[strings.ToLower(dep)] {
				t.Waiting = append(t.Waiting, dep)
			}
		}
		grouped[t.column] = append(grouped[t.column], t)
	}
	statuses := statusOrder
	if len(grouped[statusOther]) > 0 {
		statuses = append(append([]string{}, statusOrder...), statusOther)
	}
	columns := make([]boardColumn, 0, len(statuses))
	for _, status := range statuses {
		list := grouped[status]
		if list == nil {
			list = []ticket{}
		}
		if status == statusDone {
			sort.SliceStable(list, func(i, j int) bool { return finishedBefore(list[i], list[j]) })
		} else {
			sort.SliceStable(list, func(i, j int) bool { return queuedBefore(list[i], list[j]) })
		}
		icon, attention := attentionStatuses[status]
		columns = append(columns, boardColumn{Status: status, Icon: icon, Attention: attention, Tickets: list})
	}
	return board{Columns: columns}
}

func queuedBefore(a, b ticket) bool {
	if ra, rb := rank(a.Priority), rank(b.Priority); ra != rb {
		return ra < rb
	}
	if a.Created != b.Created {
		return a.Created < b.Created
	}
	return naturalLess(a.ID, b.ID)
}

// finishedBefore relies on the file's time within a day: `updated` has no hour.
func finishedBefore(a, b ticket) bool {
	if a.Updated != b.Updated {
		return a.Updated > b.Updated
	}
	if !a.modTime.Equal(b.modTime) {
		return a.modTime.After(b.modTime)
	}
	return naturalLess(b.ID, a.ID)
}

func rank(priority string) int {
	if r, ok := priorityRank[priority]; ok {
		return r
	}
	return priorityRank["normal"]
}

// loadTickets reads .ork/tickets/*.md and .ork/tickets/done/*.md, nothing deeper:
// annexes/ and captures/ are not tickets.
func (d *docStore) loadTickets() []ticket {
	tickets := d.loadTicketDir(ticketsPath, false)
	return append(tickets, d.loadTicketDir(ticketsPath+"/done", true)...)
}

func (d *docStore) loadTicketDir(rel string, inDone bool) []ticket {
	abs, err := d.resolve(rel)
	if err != nil {
		return nil
	}
	entries, err := os.ReadDir(abs)
	if err != nil {
		return nil
	}
	tickets := []ticket{}
	for _, entry := range entries {
		name := entry.Name()
		child := path.Join(rel, name)
		if strings.HasPrefix(name, ".") || !isMarkdown(name) || !d.isFile(child) {
			continue
		}
		content, err := d.read(child)
		t := newTicket(child, content, inDone)
		t.Malformed = t.Malformed || err != nil
		if info, err := entry.Info(); err == nil {
			t.modTime = info.ModTime()
		}
		tickets = append(tickets, t)
	}
	return tickets
}

func (d *docStore) read(rel string) (string, error) {
	abs, err := d.resolve(rel)
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(abs)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
