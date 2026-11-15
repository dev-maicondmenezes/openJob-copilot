package score

import (
	"html"
	"regexp"
	"strings"
)

// Block-level tags whose closing implies a line break; replacing them with
// newlines keeps the visual structure of pasted HTML job descriptions.
var newlineTags = regexp.MustCompile(`(?i)<\s*(br|/p|/div|/tr|/h[1-6])\s*/?>`)

// List items become markdown-style bullets ("- ") so structure survives.
var (
	liOpen  = regexp.MustCompile(`(?i)<\s*li\s*>`)
	liClose = regexp.MustCompile(`(?i)<\s*/\s*li\s*>`)
)

// Script and style blocks carry no signal for scoring and pollute the prompt.
// (RE2 has no backreferences, so each tag gets its own pattern.)
var noiseBlocks = regexp.MustCompile(`(?is)<\s*(script|style)\b.*?<\s*/\s*(script|style)\s*>`)

// Any remaining tag (opening, closing or self-closing).
var anyTag = regexp.MustCompile(`<[^>]*>`)

// Collapse runs of whitespace (including newlines) — used to normalize text.
var spaceRuns = regexp.MustCompile(`[ \t\r\f\v]+`)

// Three or more consecutive newlines become one blank line.
var blankRuns = regexp.MustCompile(`\n{3,}`)

// CleanJD normalizes a raw job description before sending it to the LLM
// (spec §10.5): strips HTML tags (keeping line structure), unescapes entities
// and collapses whitespace. Plain-text input passes through almost unchanged.
func CleanJD(raw string) string {
	s := noiseBlocks.ReplaceAllString(raw, " ")
	s = newlineTags.ReplaceAllString(s, "\n")
	s = liOpen.ReplaceAllString(s, "\n- ")
	s = liClose.ReplaceAllString(s, " ")
	s = anyTag.ReplaceAllString(s, " ")
	s = html.UnescapeString(s)
	s = strings.ReplaceAll(s, "\u00a0", " ") // &nbsp; arrives as NBSP after unescape

	// Normalize line-by-line: single spaces inside lines, no trailing blanks.
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimSpace(spaceRuns.ReplaceAllString(line, " "))
	}
	s = strings.Join(lines, "\n")
	s = blankRuns.ReplaceAllString(s, "\n\n")

	return strings.TrimSpace(s)
}
