package nlp

import (
	"regexp"
	"strings"
)

// headingPattern matches ATX Markdown headings (up to 3 leading spaces).
var headingPattern = regexp.MustCompile(`^[ \t]{0,3}(#{1,6})[ \t]+(.*)$`)

// RawSection is a generic segment produced by Segment. Offsets are byte
// positions in the source text; Text is exactly source[StartOffset:EndOffset].
type RawSection struct {
	Title       string
	Text        string
	StartOffset int
	EndOffset   int
}

// Segment splits text into generic sections using Markdown headings. When the
// text has no heading, a single untitled section covers the whole input.
//
// This is deliberately structural, not semantic: the NLP core does not know
// which heading means "experience" or "education". Consumers may also supply
// their own sections to Analyze.
func Segment(text string) []RawSection {
	type heading struct {
		start int // byte offset of the heading line
		end   int // byte offset just after the heading line
		title string
	}
	var headings []heading

	for start := 0; start < len(text); {
		lineEnd := strings.IndexByte(text[start:], '\n')
		var end int
		if lineEnd < 0 {
			end = len(text)
		} else {
			end = start + lineEnd + 1
		}
		line := strings.TrimRight(text[start:end], "\r\n")
		if m := headingPattern.FindStringSubmatch(line); m != nil {
			headings = append(headings, heading{
				start: start,
				end:   end,
				title: strings.TrimSpace(m[2]),
			})
		}
		if lineEnd < 0 {
			break
		}
		start = end
	}

	if len(headings) == 0 {
		return []RawSection{{Title: "", Text: text, StartOffset: 0, EndOffset: len(text)}}
	}

	sections := make([]RawSection, 0, len(headings)+1)
	if first := headings[0].start; first > 0 {
		if strings.TrimSpace(text[:first]) != "" {
			sections = append(sections, RawSection{Title: "", Text: text[:first], StartOffset: 0, EndOffset: first})
		}
	}
	for i, h := range headings {
		start := h.end
		end := len(text)
		if i+1 < len(headings) {
			end = headings[i+1].start
		}
		sections = append(sections, RawSection{
			Title:       h.title,
			Text:        text[start:end],
			StartOffset: start,
			EndOffset:   end,
		})
	}
	return sections
}
