package nlp

import (
	"strings"

	"golang.org/x/text/unicode/norm"
)

// Normalize applies Unicode NFKC compatibility normalization and lowercases
// the text. It never mutates the original document: callers keep both forms.
func Normalize(text string) string {
	return strings.ToLower(norm.NFKC.String(text))
}

// NormalizeWord normalizes a single token (NFKC + lowercase + trim).
func NormalizeWord(word string) string {
	return strings.TrimSpace(Normalize(word))
}
