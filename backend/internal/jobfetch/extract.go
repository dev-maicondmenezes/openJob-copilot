package jobfetch

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"golang.org/x/net/html"

	"github.com/maicondmenezes/openjob/internal/score"
)

// extractor walks a parsed page and pulls title, company and description
// using layered heuristics (JSON-LD → meta tags → largest text block).
type extractor struct {
	title       string
	company     string
	description string

	titleTag string
	metas    map[string]string // lower-cased name/property → content
	jsonld   []string          // raw application/ld+json payloads
	blocks   []*blockCandidate
}

// blockCandidate is a potential description container.
type blockCandidate struct {
	node *html.Node
	text string
	len  int
}

// Block elements considered as description containers.
var blockTags = map[string]bool{"article": true, "main": true, "section": true, "div": true}

// Tags that force a line break when flattening text.
var breakTags = map[string]bool{
	"p": true, "div": true, "tr": true, "section": true, "article": true,
	"header": true, "footer": true, "aside": true, "h1": true, "h2": true,
	"h3": true, "h4": true, "h5": true, "h6": true, "blockquote": true,
}

func (e *extractor) extract(root *html.Node) {
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode {
			switch n.Data {
			case "title":
				e.titleTag = strings.TrimSpace(nodeText(n))
			case "meta":
				e.collectMeta(n)
			case "script":
				if hasAttr(n, "type", "application/ld+json") {
					e.jsonld = append(e.jsonld, rawText(n))
				}
			case "body":
				e.blocks = append(e.blocks, newCandidate(n))
			case "h1":
				if e.title == "" {
					e.title = strings.TrimSpace(nodeText(n))
				}
			}
			if blockTags[n.Data] && n.Data != "body" {
				e.blocks = append(e.blocks, newCandidate(n))
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(root)

	if err := e.extractJSONLD(); err != nil || e.description == "" {
		e.pickLargestBlock()
	}
	if e.title == "" {
		e.title = e.metaAny("og:title", "twitter:title")
	}
	if e.title == "" {
		e.title = e.titleTag
	}
	if e.company == "" {
		e.company = e.metaAny("og:site_name", "application-name")
	}
	// Page <title> chrome often carries the company as a suffix:
	// "Pleno DevOps — Gama" / "Job | Careers Site".
	if e.company == "" && e.titleTag != "" {
		if _, company := splitTitleCompany(e.titleTag); company != "" {
			e.company = company
		}
	}
	e.description = normalizeDescription(e.description)
	e.title = flatten(e.title)
	e.company = flatten(e.company)
}

func newCandidate(n *html.Node) *blockCandidate {
	text := nodeText(n)
	return &blockCandidate{node: n, text: text, len: len(text)}
}

// collectMeta records <meta name|property="…" content="…"> pairs.
func (e *extractor) collectMeta(n *html.Node) {
	key := strings.ToLower(getAttr(n, "name"))
	if key == "" {
		key = strings.ToLower(getAttr(n, "property"))
	}
	if key == "" {
		return
	}
	if e.metas == nil {
		e.metas = map[string]string{}
	}
	if _, ok := e.metas[key]; !ok { // first occurrence wins
		e.metas[key] = getAttr(n, "content")
	}
}

func (e *extractor) metaAny(keys ...string) string {
	for _, k := range keys {
		if v := strings.TrimSpace(e.metas[k]); v != "" {
			return v
		}
	}
	return ""
}

// extractJSONLD scans the page's ld+json payloads for a schema.org JobPosting.
func (e *extractor) extractJSONLD() error {
	for _, raw := range e.jsonld {
		var v any
		if err := json.Unmarshal([]byte(raw), &v); err != nil {
			continue
		}
		if found := scanJSONLD(v); found != nil {
			if found.title != "" {
				e.title = found.title
			}
			if found.company != "" {
				e.company = found.company
			}
			if found.description != "" {
				e.description = found.description
			}
			return nil
		}
	}
	return fmt.Errorf("no JobPosting")
}

type posting struct {
	title       string
	company     string
	description string
}

// scanJSONLD searches any decoded JSON value for a JobPosting object
// (objects, arrays and @graph containers included).
func scanJSONLD(v any) *posting {
	switch t := v.(type) {
	case map[string]any:
		if isJobPosting(t) {
			p := &posting{
				title:       asString(t["title"], t["name"]),
				description: asString(t["description"]),
			}
			if org, ok := t["hiringOrganization"].(map[string]any); ok {
				p.company = asString(org["name"])
			}
			if p.title != "" || p.description != "" {
				return p
			}
			return nil
		}
		for _, child := range []string{"@graph", "itemListElement", "mainEntity"} {
			if nested, ok := t[child]; ok {
				if p := scanJSONLD(nested); p != nil {
					return p
				}
			}
		}
		for _, child := range t {
			if p := scanJSONLD(child); p != nil {
				return p
			}
		}
	case []any:
		for _, item := range t {
			if p := scanJSONLD(item); p != nil {
				return p
			}
		}
	}
	return nil
}

func isJobPosting(m map[string]any) bool {
	at := m["@type"]
	switch v := at.(type) {
	case string:
		return strings.EqualFold(v, "JobPosting")
	case []any:
		for _, s := range v {
			if str, ok := s.(string); ok && strings.EqualFold(str, "JobPosting") {
				return true
			}
		}
	}
	return false
}

func asString(vals ...any) string {
	for _, v := range vals {
		if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
			return s
		}
	}
	return ""
}

// minJDChars: below this a block is page chrome, not a description — the
// caller/UI should fall back to manual paste (JS-rendered ATS pages).
const minJDChars = 120

// pickLargestBlock chooses the description from the DOM heuristically:
// semantic containers (article/main/section) beat generic div/body, and a
// candidate that merely wraps an equally big block is skipped (avoids the
// body/nav trap — the description lives in the leaf with the real content).
func (e *extractor) pickLargestBlock() {
	var best *blockCandidate
	for _, c := range e.blocks {
		if c.len == 0 || wrapsAnotherBigBlock(c, e.blocks) {
			continue
		}
		if best == nil || preferBlock(c, best) {
			best = c
		}
	}
	if best != nil && best.len >= minJDChars {
		e.description = best.text
	}
	// Sem bloco grande o suficiente → descrição vazia (página JS-renderizada).
}

// wrapsAnotherBigBlock reports whether c contains another candidate with at
// least 80% of c's text (meaning c is just a wrapper around it).
func wrapsAnotherBigBlock(c *blockCandidate, all []*blockCandidate) bool {
	for _, other := range all {
		if other == c || other.len == 0 {
			continue
		}
		if isDescendant(other.node, c.node) && other.len*5 >= c.len*4 {
			return true
		}
	}
	return false
}

// titleSuffixBlocklist: segments that look like page chrome, not a company.
var titleSuffixBlocklist = map[string]bool{
	"remote": true, "jobs": true, "job": true, "career": true, "careers": true,
	"vaga": true, "vagas": true, "vagas remotas": true, "join us": true,
	"join us!": true, "we're hiring": true, "apply now": true,
	"job application": true, "open role": true,
}

// splitTitleCompany splits a page <title> like "Pleno DevOps — Gama" into
// (title, company) using the common separators of job-site chrome. Returns
// company = "" when there is no plausible suffix.
func splitTitleCompany(title string) (string, string) {
	seps := []string{" | ", " — ", " – ", " - ", " ｜ "}
	for _, sep := range seps {
		if idx := strings.LastIndex(title, sep); idx > 0 {
			company := strings.TrimSpace(title[idx+len(sep):])
			head := strings.TrimSpace(title[:idx])
			lower := strings.ToLower(company)
			if head != "" && company != "" && len(company) <= 40 && !titleSuffixBlocklist[lower] {
				return head, company
			}
		}
	}
	return title, ""
}

// preferBlock breaks ties: semantic tags beat div; longer text wins among
// equal-quality containers (the wrapper rule already filtered the parents).
func preferBlock(c, best *blockCandidate) bool {
	rank := func(s string) int {
		switch s {
		case "article":
			return 2
		case "main":
			return 2
		case "section":
			return 1
		default:
			return 0
		}
	}
	if r1, r2 := rank(c.node.Data), rank(best.node.Data); r1 != r2 {
		return r1 > r2
	}
	return c.len > best.len
}

func isDescendant(child, ancestor *html.Node) bool {
	for n := child.Parent; n != nil; n = n.Parent {
		if n == ancestor {
			return true
		}
	}
	return false
}

// Tags whose entire subtree must be ignored when flattening text (page
// chrome and JS-rendered apps embed huge blobs that pollute the JD).
var skipTags = map[string]bool{"script": true, "style": true, "noscript": true, "template": true}

// nodeText flattens a node's subtree into text with basic line structure:
// block tags produce newlines, list items become "- " bullets, <br> breaks.
func nodeText(n *html.Node) string {
	var sb strings.Builder
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		switch n.Type {
		case html.TextNode:
			sb.WriteString(n.Data)
		case html.ElementNode:
			if skipTags[n.Data] {
				return
			}
			switch n.Data {
			case "br":
				sb.WriteString("\n")
			case "li":
				sb.WriteString("\n- ")
			}
			if breakTags[n.Data] && sb.Len() > 0 {
				sb.WriteString("\n")
			}
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				walk(c)
			}
		}
	}
	walk(n)
	return sb.String()
}

// rawText concatenates only the text nodes of a subtree (no newline rules,
// no tag skipping) — used to read JSON-LD payloads inside <script>.
func rawText(n *html.Node) string {
	var sb strings.Builder
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.TextNode {
			sb.WriteString(n.Data)
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return sb.String()
}

// getAttr returns the value of an attribute, or "" when absent.
func getAttr(n *html.Node, key string) string {
	for _, a := range n.Attr {
		if strings.EqualFold(a.Key, key) {
			return a.Val
		}
	}
	return ""
}

func hasAttr(n *html.Node, key, value string) bool {
	return strings.EqualFold(getAttr(n, key), value)
}

// parseHTML parses an HTML document from a reader (kept here so tests can
// share the entry point with Fetch).
func parseHTML(r io.Reader) (*html.Node, error) {
	return html.Parse(r)
}

// flatten normalizes a one-line field (title/company): no newlines, no runs.
func flatten(s string) string {
	s = strings.Join(strings.Fields(s), " ")
	return strings.TrimSpace(s)
}

// normalizeDescription reuses the JD cleaner from the scoring package so
// what reaches the form looks exactly like what reaches the LLM.
func normalizeDescription(s string) string { return score.CleanJD(s) }
