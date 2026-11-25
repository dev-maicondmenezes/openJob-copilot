package keywords

import (
	"math"
	"regexp"
	"sort"
	"strings"

	"github.com/maicondmenezes/openjob/internal/nlp"
)

// rakeSplit splits a section into sentence-ish fragments. Stopwords and
// punctuation delimit candidate phrases inside each fragment.
var rakeSplit = regexp.MustCompile(`[.,;:!?()\[\]{}\n\r\t/|]+`)

// ExtractRAKE extracts ranked keywords from a single section using RAKE.
// Higher scores are better. It is used only when YAKE yields no candidate.
func ExtractRAKE(section nlp.TextSection, analyzer *nlp.Analyzer, lang nlp.Language) []Keyword {
	if analyzer == nil || len(section.Tokens) == 0 {
		return nil
	}
	res := analyzer.Resources()

	type phrase struct {
		text       string
		normalized string
		words      []string
	}
	var phrases []phrase

	for _, fragment := range rakeSplit.Split(section.Text, -1) {
		tokens := nlp.TokenizeAll(fragment, lang)
		current := phrase{}
		flush := func() {
			if len(current.words) > 0 {
				phrases = append(phrases, current)
			}
			current = phrase{}
		}
		for _, token := range tokens {
			if res.IsStopword(lang, token.Normalized) {
				flush()
				continue
			}
			if len([]rune(token.Normalized)) < 2 {
				continue
			}
			if current.text == "" {
				current.text = token.Original
			} else {
				current.text += " " + token.Original
			}
			current.normalized = strings.TrimSpace(current.normalized + " " + token.Term())
			current.words = append(current.words, token.Term())
		}
		flush()
	}
	if len(phrases) == 0 {
		return nil
	}

	freq := make(map[string]int)
	degree := make(map[string]int)
	for _, p := range phrases {
		unique := make(map[string]struct{}, len(p.words))
		for _, word := range p.words {
			freq[word]++
			unique[word] = struct{}{}
		}
		k := len(unique)
		for word := range unique {
			if k > 1 {
				degree[word] += k - 1
			}
		}
	}

	type scored struct {
		phrase
		score float64
	}
	results := make([]scored, 0, len(phrases))
	for _, p := range phrases {
		var score float64
		unique := make(map[string]struct{}, len(p.words))
		for _, word := range p.words {
			if _, seen := unique[word]; seen {
				continue
			}
			unique[word] = struct{}{}
			if freq[word] > 0 {
				score += float64(degree[word]) / float64(freq[word])
			}
		}
		results = append(results, scored{phrase: p, score: score})
	}

	sort.Slice(results, func(i, j int) bool {
		if results[i].score != results[j].score {
			return results[i].score > results[j].score
		}
		return results[i].normalized < results[j].normalized
	})

	sectionID := sectionIdentifier(section)
	out := make([]Keyword, 0, len(results))
	for i, r := range results {
		out = append(out, Keyword{
			Text:       r.text,
			Normalized: r.normalized,
			Section:    sectionID,
			Source:     SourceRAKE,
			Rank:       i + 1,
			Score:      math.Round(r.score*1e6) / 1e6,
		})
	}
	return out
}
