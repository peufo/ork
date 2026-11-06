package main

import (
	"strings"
	"testing"
)

func renderFixture(t *testing.T, rel, body string) string {
	t.Helper()
	store := newFixture(t, map[string]string{
		"doc/decisions.md":              "x",
		"doc/specs/mcp.md":              "x",
		"doc/story/a.md":                "x",
		"doc/kanban.base":               "x",
		".ork/tickets/t1-a.md":          "x",
		".ork/tickets/captures/a.png":   "x",
		".ork/tickets/captures/b c.png": "x",
	})
	html, err := newRenderer(store).render(rel, body)
	if err != nil {
		t.Fatal(err)
	}
	return html
}

func TestRenderLinks(t *testing.T) {
	cases := []struct {
		name, from, source, want, absent string
	}{
		{name: "relative", from: "doc/decisions.md", source: "[m](specs/mcp.md)", want: `href="#/doc/doc/specs/mcp.md"`},
		{name: "parent", from: "doc/story/a.md", source: "[d](../decisions.md)", want: `href="#/doc/doc/decisions.md"`},
		{name: "from repo root", from: "doc/story/a.md", source: "[t](.ork/tickets/t1-a.md)", want: `href="#/doc/.ork/tickets/t1-a.md"`},
		{name: "fragment", from: "doc/decisions.md", source: "[m](specs/mcp.md#outils)", want: `href="#/doc/doc/specs/mcp.md?a=outils"`},
		{name: "anchor", from: "doc/decisions.md", source: "[v](#vérification-après)", want: `href="#/doc/doc/decisions.md?a=v%C3%A9rification-apr%C3%A8s"`},
		{name: "outside doc", from: "doc/story/a.md", source: "[f](../../src/foo.ts)", want: ">f<", absent: "<a "},
		{name: "repo file", from: "doc/decisions.md", source: "[f](src/foo.ts)", want: ">f<", absent: "<a "},
		{name: "with line", from: "doc/decisions.md", source: "[f](schema.ts:40)", want: ">f<", absent: "<a "},
		{name: "missing", from: "doc/decisions.md", source: "[f](nope.md)", want: ">f<", absent: "<a "},
		{name: "not markdown", from: "doc/decisions.md", source: "[f](kanban.base)", want: ">f<", absent: "<a "},
		{name: "secret", from: "doc/decisions.md", source: "[f](../../.env)", want: ">f<", absent: "<a "},
		{name: "external", from: "doc/decisions.md", source: "[e](https://example.com)", want: `target="_blank"`},
		{name: "external noopener", from: "doc/decisions.md", source: "[e](https://example.com)", want: `noopener`},
		{name: "image", from: ".ork/tickets/t1-a.md", source: "![c](captures/a.png)", want: `src="/raw/.ork/tickets/captures/a.png"`},
		{name: "image with space", from: ".ork/tickets/t1-a.md", source: "![c](<captures/b c.png>)", want: `src="/raw/.ork/tickets/captures/b%20c.png"`},
		{name: "link to image", from: ".ork/tickets/t1-a.md", source: "[c](captures/a.png)", want: `href="/raw/.ork/tickets/captures/a.png"`},
		{name: "missing image", from: "doc/decisions.md", source: "![alt](nope.png)", want: "alt", absent: "<img"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := renderFixture(t, c.from, c.source)
			if !strings.Contains(got, c.want) {
				t.Errorf("want %q in %s", c.want, got)
			}
			if c.absent != "" && strings.Contains(got, c.absent) {
				t.Errorf("did not want %q in %s", c.absent, got)
			}
		})
	}
}

func TestRenderLinksCitedPaths(t *testing.T) {
	cases := []struct {
		name, from, source, want, absent string
	}{
		{name: "capture", from: ".ork/tickets/t1-a.md", source: "capture `.ork/tickets/captures/a.png` ici", want: `<a href="/raw/.ork/tickets/captures/a.png"><code>.ork/tickets/captures/a.png</code></a>`},
		{name: "story", from: ".ork/tickets/t1-a.md", source: "`doc/story/a.md`", want: `<a href="#/doc/doc/story/a.md"><code>doc/story/a.md</code></a>`},
		{name: "ticket from a story", from: "doc/story/a.md", source: "`.ork/tickets/t1-a.md`", want: `<a href="#/doc/.ork/tickets/t1-a.md"><code>.ork/tickets/t1-a.md</code></a>`},
		{name: "relative", from: "doc/decisions.md", source: "`specs/mcp.md`", want: `href="#/doc/doc/specs/mcp.md"`},
		{name: "plain code", from: "doc/decisions.md", source: "`npm test`", want: "<code>npm test</code>", absent: "<a "},
		{name: "missing", from: "doc/decisions.md", source: "`doc/nope.png`", want: "<code>doc/nope.png</code>", absent: "<a "},
		{name: "other file", from: "doc/decisions.md", source: "`doc/kanban.base`", want: "<code>doc/kanban.base</code>", absent: "<a "},
		{name: "outside doc", from: "doc/decisions.md", source: "`../.env`", want: "<code>../.env</code>", absent: "<a "},
		{name: "repo root secret", from: "doc/decisions.md", source: "`.env`", want: "<code>.env</code>", absent: "<a "},
		{name: "inside a link", from: "doc/decisions.md", source: "[`specs/mcp.md`](https://example.com)", want: "https://example.com", absent: "#/doc/"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := renderFixture(t, c.from, c.source)
			if !strings.Contains(got, c.want) {
				t.Errorf("want %q in %s", c.want, got)
			}
			if c.absent != "" && strings.Contains(got, c.absent) {
				t.Errorf("did not want %q in %s", c.absent, got)
			}
		})
	}
}

func TestRenderHeadingIDs(t *testing.T) {
	got := renderFixture(t, "doc/decisions.md", "# Points propres à Poj\n\n## DNS\n\n## DNS\n\n## `ORIGIN` et proxy\n")
	for _, want := range []string{`id="points-propres-à-poj"`, `id="dns"`, `id="dns-1"`, `id="origin-et-proxy"`} {
		if !strings.Contains(got, want) {
			t.Errorf("want %s in %s", want, got)
		}
	}
}

func TestRenderSanitizes(t *testing.T) {
	cases := []string{
		"<script>alert(1)</script>",
		`<img src="x" onerror="alert(1)">`,
		`<a href="javascript:alert(1)">x</a>`,
		"[x](javascript:alert(1))",
		"[x](data:text/html;base64,PHNjcmlwdD4=)",
		`<iframe src="https://example.com"></iframe>`,
		`<svg onload="alert(1)"></svg>`,
		`<div style="position:fixed">x</div>`,
		`<form action="https://example.com"><input name="a"></form>`,
	}
	forbidden := []string{"<script", "onerror", "javascript:", "data:text", "<iframe", "onload", "<svg", "style=", "<form", "name="}
	for _, source := range cases {
		got := renderFixture(t, "doc/decisions.md", source)
		for _, word := range forbidden {
			if strings.Contains(got, word) {
				t.Errorf("%q left %q in %s", source, word, got)
			}
		}
	}
}

func TestRenderKeepsUsefulMarkup(t *testing.T) {
	got := renderFixture(t, "doc/decisions.md", "- [x] fait\n- [ ] à faire\n\n| a |\n|---|\n| b |\n\n<details><summary>s</summary>d</details>\n")
	for _, want := range []string{`type="checkbox"`, `checked`, `disabled`, "<table>", "<details>", "<summary>"} {
		if !strings.Contains(got, want) {
			t.Errorf("want %s in %s", want, got)
		}
	}
}
