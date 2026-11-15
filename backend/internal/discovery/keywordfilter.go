package discovery

import (
	"regexp"
	"strings"
)

// MatchKeyword reports whether keyword appears as a distinct word in text.
// Uses word boundary matching so that "go" matches "Go Developer" but not "Google" or "algorithm".
// Handles special terms like "c++", "c#", ".net".
func MatchKeyword(text, keyword string) bool {
	kw := strings.TrimSpace(keyword)
	if kw == "" || text == "" {
		return false
	}

	// Escape regexp special characters except when we want literal matching
	pattern := regexp.QuoteMeta(strings.ToLower(kw))
	// Standard word boundaries \b don't always work cleanly when kw ends with # or +
	// Precede with non-alphanumeric or start of string, follow with non-alphanumeric or end of string
	reStr := `(?i)(?:^|[^a-zA-Z0-9_#+])` + pattern + `(?:$|[^a-zA-Z0-9_#+])`
	re, err := regexp.Compile(reStr)
	if err != nil {
		// Fallback to case-insensitive substring if regexp compilation fails
		return strings.Contains(strings.ToLower(text), strings.ToLower(kw))
	}
	return re.MatchString(text)
}

// FilterKeywords evaluates whether a vacancy's title and description match the user's keywords.
// Returns (matched, matchedKeywords).
// If keywords is empty, matches everything with an empty matched slice.
func FilterKeywords(title, description string, keywords []string, matchMode string) (bool, []string) {
	if len(keywords) == 0 {
		return true, nil
	}

	text := title + "\n" + description
	matched := make([]string, 0, len(keywords))

	for _, kw := range keywords {
		kw = strings.TrimSpace(kw)
		if kw == "" {
			continue
		}
		if MatchKeyword(text, kw) {
			matched = append(matched, kw)
		}
	}

	if strings.EqualFold(matchMode, "all") {
		// All non-empty keywords must match
		nonEmpty := 0
		for _, kw := range keywords {
			if strings.TrimSpace(kw) != "" {
				nonEmpty++
			}
		}
		return len(matched) == nonEmpty, matched
	}

	// Default: "any" (at least 1 match)
	return len(matched) > 0, matched
}
