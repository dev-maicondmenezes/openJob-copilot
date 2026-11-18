package discovery

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"golang.org/x/net/html"
)

// maxScrapeBody caps a single HTML/JSON page read (sources are server-rendered
// listings; anything larger is an error page or abuse).
const maxScrapeBody = 8 << 20

// scrapeUserAgent identifies openJob explicitly. Scraping sources never try to
// impersonate a browser (CP14 §7).
const scrapeUserAgent = "openJob/1.0 (+https://github.com/maicondmenezes/openjob; respectful crawler)"

// fetchPage performs one HTTP GET honoring opt.Wait when set (the source rate
// limiter) and returns the parsed document plus the final URL.
func fetchPage(ctx context.Context, client *http.Client, rawURL string, wait RequestWaiter) (*html.Node, string, error) {
	body, finalURL, err := fetchBytes(ctx, client, rawURL, wait, nil)
	if err != nil {
		return nil, finalURL, err
	}
	doc, err := html.Parse(strings.NewReader(string(body)))
	if err != nil {
		return nil, finalURL, fmt.Errorf("parse html: %w", err)
	}
	return doc, finalURL, nil
}

// fetchBytes is the single HTTP entry point of every scraping adapter: it
// applies the source rate limiter, a conservative User-Agent, a body ceiling
// and maps 429/403 to BlockedError so the orchestrator can back off.
func fetchBytes(ctx context.Context, client *http.Client, rawURL string, wait RequestWaiter, headers map[string]string) ([]byte, string, error) {
	if wait != nil {
		if err := wait.Wait(ctx); err != nil {
			return nil, rawURL, err
		}
	}
	if client == nil {
		client = http.DefaultClient
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, rawURL, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("User-Agent", scrapeUserAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, rawURL, fmt.Errorf("http request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode == http.StatusForbidden {
		return nil, rawURL, &BlockedError{StatusCode: resp.StatusCode}
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, rawURL, fmt.Errorf("unexpected status HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxScrapeBody))
	if err != nil {
		return nil, rawURL, fmt.Errorf("read body: %w", err)
	}
	return body, resp.Request.URL.String(), nil
}

// textNode flattens a subtree into normalized plain text.
func textNode(n *html.Node) string {
	if n == nil {
		return ""
	}
	var sb strings.Builder
	var walk func(*html.Node)
	walk = func(cur *html.Node) {
		if cur.Type == html.TextNode {
			sb.WriteString(cur.Data)
			sb.WriteByte(' ')
			return
		}
		if cur.Type == html.ElementNode && skipTags[cur.Data] {
			return
		}
		for c := cur.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return strings.Join(strings.Fields(sb.String()), " ")
}

// attrVal returns an attribute value ("" when absent).
func attrVal(n *html.Node, key string) string {
	if n == nil {
		return ""
	}
	for _, a := range n.Attr {
		if strings.EqualFold(a.Key, key) {
			return a.Val
		}
	}
	return ""
}

// hasClass reports whether an element carries one of the given classes.
func hasClass(n *html.Node, classes ...string) bool {
	if n == nil || n.Type != html.ElementNode {
		return false
	}
	raw := " " + attrVal(n, "class") + " "
	for _, c := range classes {
		if strings.Contains(raw, " "+c+" ") {
			return true
		}
	}
	return false
}

// walkElements visits every element node in document order.
func walkElements(root *html.Node, fn func(*html.Node)) {
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode {
			fn(n)
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(root)
}

// allByClass returns every element carrying the class.
func allByClass(root *html.Node, class string) []*html.Node {
	var out []*html.Node
	walkElements(root, func(n *html.Node) {
		if hasClass(n, class) {
			out = append(out, n)
		}
	})
	return out
}

// firstByClass returns the first element carrying the class.
func firstByClass(root *html.Node, class string) *html.Node {
	var found *html.Node
	walkElements(root, func(n *html.Node) {
		if found == nil && hasClass(n, class) {
			found = n
		}
	})
	return found
}

// firstByTag returns the first element with the tag name.
func firstByTag(root *html.Node, tag string) *html.Node {
	var found *html.Node
	walkElements(root, func(n *html.Node) {
		if found == nil && n.Data == tag {
			found = n
		}
	})
	return found
}

// allByTag returns every element with the tag name.
func allByTag(root *html.Node, tag string) []*html.Node {
	var out []*html.Node
	walkElements(root, func(n *html.Node) {
		if n.Data == tag {
			out = append(out, n)
		}
	})
	return out
}

// firstByAttr returns the first element with the attribute value.
func firstByAttr(root *html.Node, key, value string) *html.Node {
	var found *html.Node
	walkElements(root, func(n *html.Node) {
		if found == nil && strings.EqualFold(attrVal(n, key), value) {
			found = n
		}
	})
	return found
}

// firstLink returns the first anchor href inside the subtree.
func firstLink(root *html.Node) string {
	var href string
	walkElements(root, func(n *html.Node) {
		if href == "" && n.Data == "a" {
			href = attrVal(n, "href")
		}
	})
	return href
}

// absoluteURL resolves href against base. Invalid URLs are returned as-is so
// the URL dedup layer can still decide.
func absoluteURL(base, href string) string {
	href = strings.TrimSpace(href)
	if href == "" {
		return ""
	}
	b, err := url.Parse(base)
	if err != nil {
		return href
	}
	u, err := url.Parse(href)
	if err != nil {
		return href
	}
	return b.ResolveReference(u).String()
}

// textOr returns the trimmed text of n, falling back to alt candidates.
func textOr(n *html.Node, fallbacks ...string) string {
	if text := strings.TrimSpace(textNode(n)); text != "" {
		return text
	}
	for _, f := range fallbacks {
		if f = strings.TrimSpace(f); f != "" {
			return f
		}
	}
	return ""
}

// extractExternalID extracts a stable id from a URL path (last non-empty
// segment), used when a listing card does not expose a slug/id attribute.
func extractExternalID(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	for i := len(parts) - 1; i >= 0; i-- {
		if p := strings.TrimSuffix(parts[i], ".html"); p != "" {
			return p
		}
	}
	return ""
}
