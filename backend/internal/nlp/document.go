// Package nlp implements the generic, deterministic text-processing core
// shared by the keyword, ranking and filtering engines (Fase 5 / B11.1).
//
// The package deliberately knows nothing about curricula, vacancies,
// professional experience or evidence mapping. Documents are generic text
// with sections, tokens and provenance.
package nlp

// TextDocument is a generic document with an optional section structure.
//
// Offsets are byte positions in OriginalText using UTF-8, following Go's
// native string indexing. The original text is always preserved so future
// consumers (tailoring, document intelligence) can trace a token or section
// back to its source without re-processing.
type TextDocument struct {
	ID              string
	OriginalText    string
	Language        string
	Sections        []TextSection
	Metadata        DocumentMetadata
	ResourceVersion string
}

// TextSection is a contiguous, offset-addressed region of a document.
type TextSection struct {
	ID          string
	Title       string
	Text        string
	StartOffset int
	EndOffset   int
	Tokens      []Token
}

// Token preserves the original surface form, the normalized form, the stem
// and the byte offsets. Token offsets are relative to the parent
// TextSection.Text, while TextSection offsets are relative to
// TextDocument.OriginalText; adding the section StartOffset yields absolute
// document offsets.
type Token struct {
	Original    string
	Normalized  string
	Stem        string
	StartOffset int
	EndOffset   int
	Position    int
}

// DocumentMetadata carries non-linguistic provenance for a document.
type DocumentMetadata struct {
	SourceType string
	SourceID   string
}

// Term returns the matching form of the token (its stem when available).
func (t Token) Term() string {
	if t.Stem != "" {
		return t.Stem
	}
	return t.Normalized
}

// Terms returns the matching forms of the tokens, preserving order.
func Terms(tokens []Token) []string {
	terms := make([]string, 0, len(tokens))
	for _, token := range tokens {
		terms = append(terms, token.Term())
	}
	return terms
}
