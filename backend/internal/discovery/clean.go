package discovery

import (
	"strings"

	"golang.org/x/net/html"
)

var skipTags = map[string]bool{
	"script":   true,
	"style":    true,
	"noscript": true,
	"template": true,
}

var breakTags = map[string]bool{
	"p":          true,
	"div":        true,
	"tr":         true,
	"section":    true,
	"article":    true,
	"header":     true,
	"footer":     true,
	"aside":      true,
	"h1":         true,
	"h2":         true,
	"h3":         true,
	"h4":         true,
	"h5":         true,
	"h6":         true,
	"blockquote": true,
}

// CleanHTML extracts readable plain text with preserved line breaks from an HTML snippet.
func CleanHTML(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	doc, err := html.Parse(strings.NewReader(raw))
	if err != nil {
		return raw
	}

	var sb strings.Builder
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.TextNode {
			sb.WriteString(n.Data)
			return
		}
		if n.Type == html.ElementNode {
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
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)

	// Clean excessive whitespace while keeping paragraph structure
	lines := strings.Split(sb.String(), "\n")
	cleanLines := make([]string, 0, len(lines))
	emptyCount := 0
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			emptyCount++
			if emptyCount <= 1 {
				cleanLines = append(cleanLines, "")
			}
		} else {
			emptyCount = 0
			cleanLines = append(cleanLines, trimmed)
		}
	}
	return strings.TrimSpace(strings.Join(cleanLines, "\n"))
}
