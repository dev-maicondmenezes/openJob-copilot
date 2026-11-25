package nlp

import "strings"

// Slug converts a section title to a stable, lowercase, hyphen-separated id.
// It is a generic helper; mapping a slug to a domain section (experience,
// education...) belongs to the consumer.
func Slug(title string) string {
	var b strings.Builder
	lastDash := false
	for _, r := range Normalize(title) {
		switch {
		case isLetter(r) || isDigit(r):
			b.WriteRune(r)
			lastDash = false
		case !lastDash && b.Len() > 0:
			b.WriteByte('-')
			lastDash = true
		}
	}
	return strings.Trim(b.String(), "-")
}
