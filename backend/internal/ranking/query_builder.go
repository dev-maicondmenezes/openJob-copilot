package ranking

import (
	"regexp"
	"sort"
	"strings"

	"github.com/maicondmenezes/openjob/internal/nlp"
)

// DefaultMaxQueryTerms bounds the FTS query to the most informative terms of
// the curriculum, keeping the query short and deterministic.
const DefaultMaxQueryTerms = 64

var ftsTermPattern = regexp.MustCompile(`[\p{L}\p{N}]+`)

// BuildQuery converts curriculum text into a safe FTS5 MATCH expression. The
// terms are stems produced by the NLP core, split into the alphanumeric tokens
// the FTS5 unicode61 tokenizer recognizes. Repeated terms are collapsed and
// ordered by frequency then alphabetically.
func BuildQuery(analyzer *nlp.Analyzer, text string, maxTerms int) (string, []string) {
	if analyzer == nil || strings.TrimSpace(text) == "" {
		return "", nil
	}
	if maxTerms <= 0 {
		maxTerms = DefaultMaxQueryTerms
	}
	lang := analyzer.DetectLanguage(text)
	tokens := nlp.TokenizeContent(text, lang, analyzer.Resources())

	freq := make(map[string]int)
	for _, token := range tokens {
		for _, piece := range ftsTermPattern.FindAllString(token.Term(), -1) {
			if len([]rune(piece)) < 2 {
				continue
			}
			freq[piece]++
		}
	}
	if len(freq) == 0 {
		return "", nil
	}

	terms := make([]string, 0, len(freq))
	for term := range freq {
		terms = append(terms, term)
	}
	sort.Slice(terms, func(i, j int) bool {
		if freq[terms[i]] != freq[terms[j]] {
			return freq[terms[i]] > freq[terms[j]]
		}
		return terms[i] < terms[j]
	})
	if len(terms) > maxTerms {
		terms = terms[:maxTerms]
	}

	quoted := make([]string, len(terms))
	for i, term := range terms {
		quoted[i] = `"` + strings.ReplaceAll(term, `"`, `""`) + `"`
	}
	return strings.Join(quoted, " OR "), terms
}
