package main

import (
	"bytes"
	"net/url"
	"path"
	"regexp"
	"strconv"
	"strings"
	"unicode"

	"github.com/microcosm-cc/bluemonday"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/text"
)

// renderer turns Markdown into safe HTML. Raw HTML goes through goldmark so that
// <details> or <kbd> keep working, because bluemonday cleans the whole output.
type renderer struct {
	store  *docStore
	md     goldmark.Markdown
	policy *bluemonday.Policy
}

func newRenderer(store *docStore) *renderer {
	return &renderer{
		store: store,
		md: goldmark.New(
			goldmark.WithExtensions(extension.GFM),
			goldmark.WithParserOptions(parser.WithAutoHeadingID()),
			goldmark.WithRendererOptions(html.WithUnsafe()),
		),
		policy: newPolicy(),
	}
}

func newPolicy() *bluemonday.Policy {
	policy := bluemonday.UGCPolicy()
	policy.RequireNoFollowOnLinks(false)
	policy.AddTargetBlankToFullyQualifiedLinks(true)
	// bluemonday only allows ASCII ids by default, and headings like
	// « Vérification » are linked to with their accents.
	policy.AllowAttrs("id").Matching(regexp.MustCompile(`^[\p{L}\p{N}_-]+$`)).OnElements("h1", "h2", "h3", "h4", "h5", "h6")
	// Task lists only: nothing else needs a form element.
	policy.AllowElements("input")
	policy.AllowAttrs("type").Matching(regexp.MustCompile(`^checkbox$`)).OnElements("input")
	policy.AllowAttrs("checked", "disabled").OnElements("input")
	return policy
}

// render returns the safe HTML of a Markdown body located at rel inside doc/.
func (r *renderer) render(rel, body string) (string, error) {
	source := []byte(body)
	ctx := parser.NewContext(parser.WithIDs(newHeadingIDs()))
	document := r.md.Parser().Parse(text.NewReader(source), parser.WithContext(ctx))
	r.rewriteLinks(document, rel)
	r.linkCitedPaths(document, source, rel)
	var out bytes.Buffer
	if err := r.md.Renderer().Render(&out, source, document); err != nil {
		return "", err
	}
	return r.policy.Sanitize(out.String()), nil
}

func (r *renderer) rewriteLinks(document ast.Node, rel string) {
	var nodes []ast.Node
	_ = ast.Walk(document, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if entering && (n.Kind() == ast.KindLink || n.Kind() == ast.KindImage) {
			nodes = append(nodes, n)
		}
		return ast.WalkContinue, nil
	})
	for _, n := range nodes {
		switch node := n.(type) {
		case *ast.Link:
			if dest, ok := r.linkTarget(string(node.Destination), rel, false); ok {
				node.Destination = []byte(dest)
			} else {
				unwrap(node)
			}
		case *ast.Image:
			if dest, ok := r.linkTarget(string(node.Destination), rel, true); ok {
				node.Destination = []byte(dest)
			} else {
				unwrap(node)
			}
		}
	}
}

// linkCitedPaths makes a code span clickable when it names a document or an image
// of doc/: the lead cites stories and screenshots as `doc/tickets/captures/x.png`,
// not as Markdown links.
func (r *renderer) linkCitedPaths(document ast.Node, source []byte, rel string) {
	var spans []*ast.CodeSpan
	_ = ast.Walk(document, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		if n.Kind() == ast.KindLink {
			return ast.WalkSkipChildren, nil
		}
		if span, ok := n.(*ast.CodeSpan); ok {
			spans = append(spans, span)
		}
		return ast.WalkContinue, nil
	})
	for _, span := range spans {
		located, ok := r.locate(codeSpanText(span, source), rel)
		if !ok {
			continue
		}
		link := ast.NewLink()
		if _, isImage := imageType(located); isImage {
			link.Destination = []byte(rawRoute(located))
		} else if isMarkdown(located) {
			link.Destination = []byte(docRoute(located, ""))
		} else {
			continue
		}
		parent := span.Parent()
		parent.ReplaceChild(parent, span, link)
		link.AppendChild(link, span)
	}
}

func codeSpanText(span *ast.CodeSpan, source []byte) string {
	var content strings.Builder
	for child := span.FirstChild(); child != nil; child = child.NextSibling() {
		if node, ok := child.(*ast.Text); ok {
			content.Write(node.Segment.Value(source))
		}
	}
	return strings.TrimSpace(content.String())
}

// unwrap replaces a link or an image by its text.
func unwrap(n ast.Node) {
	parent := n.Parent()
	for child := n.FirstChild(); child != nil; {
		next := child.NextSibling()
		n.RemoveChild(n, child)
		parent.InsertBefore(parent, n, child)
		child = next
	}
	parent.RemoveChild(parent, n)
}

var urlScheme = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9+.-]*:`)

// linkTarget maps a destination found in the document at rel to what the page
// can open. ok=false means the target is out of reach (a file outside doc/, a
// missing one, an unknown scheme) and only the text should remain.
func (r *renderer) linkTarget(dest, rel string, image bool) (string, bool) {
	dest = strings.TrimSpace(dest)
	switch {
	case dest == "", strings.HasPrefix(dest, "//"):
		return "", false
	case strings.HasPrefix(dest, "#"):
		return docRoute(rel, decoded(dest[1:])), !image
	case urlScheme.MatchString(dest):
		scheme := strings.ToLower(dest[:strings.IndexByte(dest, ':')])
		return dest, scheme == "http" || scheme == "https" || (scheme == "mailto" && !image)
	}
	target, fragment, _ := strings.Cut(dest, "#")
	target, _, _ = strings.Cut(target, "?")
	located, ok := r.locate(decoded(target), rel)
	if !ok {
		return "", false
	}
	if _, isImage := imageType(located); isImage {
		return rawRoute(located), true
	}
	if isMarkdown(located) && !image {
		return docRoute(located, decoded(fragment)), true
	}
	return "", false
}

// locate finds a link's file inside doc/. The lead writes links from the
// repository root (doc/tickets/t1.md); people write them relative to the file.
func (r *renderer) locate(target, rel string) (string, bool) {
	var candidates []string
	if rooted, isRooted := strings.CutPrefix(path.Clean("/"+target), "/doc/"); isRooted {
		candidates = append(candidates, rooted)
	}
	if !strings.HasPrefix(target, "/") {
		if inDoc, isInDoc := strings.CutPrefix(path.Join("doc", path.Dir(rel), target), "doc/"); isInDoc {
			candidates = append(candidates, inDoc)
		}
	}
	for _, candidate := range candidates {
		if r.store.isFile(candidate) {
			return candidate, true
		}
	}
	return "", false
}

func decoded(value string) string {
	if unescaped, err := url.PathUnescape(value); err == nil {
		return unescaped
	}
	return value
}

func escapePath(rel string) string {
	segments := strings.Split(rel, "/")
	for i, segment := range segments {
		segments[i] = url.PathEscape(segment)
	}
	return strings.Join(segments, "/")
}

// docRoute builds a page route. The anchor travels as `?a=`: a second `#` would
// not survive sanitizing, which encodes it as part of the fragment.
func docRoute(rel, anchor string) string {
	route := "#/doc/" + escapePath(rel)
	if anchor != "" {
		route += "?a=" + url.PathEscape(anchor)
	}
	return route
}

func rawRoute(rel string) string {
	return "/raw/" + escapePath(rel)
}

// headingIDs keeps accents in heading ids, like GitHub and Obsidian: goldmark's
// default drops every non-ASCII letter, which would break `#vérification`.
type headingIDs struct {
	used map[string]int
}

func newHeadingIDs() *headingIDs {
	return &headingIDs{used: map[string]int{}}
}

func (h *headingIDs) Generate(value []byte, _ ast.NodeKind) []byte {
	id := slugify(string(value))
	if id == "" {
		id = "section"
	}
	count, seen := h.used[id]
	h.used[id] = count + 1
	if seen {
		id += "-" + strconv.Itoa(count)
		h.used[id]++
	}
	return []byte(id)
}

func (h *headingIDs) Put(value []byte) {
	h.used[string(value)]++
}

func slugify(title string) string {
	var slug strings.Builder
	for _, r := range strings.ToLower(strings.TrimSpace(title)) {
		switch {
		case unicode.IsLetter(r), unicode.IsDigit(r), r == '-', r == '_':
			slug.WriteRune(r)
		case unicode.IsSpace(r):
			slug.WriteRune('-')
		}
	}
	return slug.String()
}
