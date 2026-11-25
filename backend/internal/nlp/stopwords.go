package nlp

// negationTerms are never discarded as ordinary stopwords: removing them
// changes the meaning of a requirement ("sem experiência" ≠ "experiência").
var negationTerms = map[string]struct{}{
	"não": {}, "nao": {}, "sem": {}, "nunca": {}, "jamais": {},
	"no": {}, "nor": {}, "not": {}, "without": {}, "never": {},
	"sin": {},
}

// IsStopword reports whether the normalized word is a stopword of lang.
func (r *Resources) IsStopword(lang Language, word string) bool {
	if _, neg := negationTerms[word]; neg {
		return false
	}
	_, ok := r.Stopwords(lang)[word]
	return ok
}

// Negations exposes the negation vocabulary (read-only).
func Negations() map[string]struct{} { return negationTerms }
