package keywords

import (
	"math"
	"sort"
	"strings"

	"github.com/maicondmenezes/openjob/internal/nlp"
)

// yakeMaxNgram is the largest candidate expression considered (YAKE's
// default window for short documents).
const yakeMaxNgram = 3

type yakeCandidate struct {
	text       string
	normalized string
	tf         int
	firstPos   int
	context    float64
	length     int
	score      float64
}

// ExtractYAKE extracts ranked keywords from a single section using a
// deterministic, simplified YAKE scoring function. Lower scores are better.
//
// The implementation keeps YAKE's core idea — unsupervised scoring from term
// frequency, position and contextual diversity — without external corpora or
// professional vocabularies. It is deliberately small and reproducible.
func ExtractYAKE(section nlp.TextSection) []Keyword {
	tokens := section.Tokens
	if len(tokens) == 0 {
		return nil
	}

	candidates := make(map[string]*yakeCandidate)
	occurrences := make(map[string][]int)

	for i := 0; i < len(tokens); i++ {
		for n := 1; n <= yakeMaxNgram && i+n <= len(tokens); n++ {
			window := tokens[i : i+n]
			terms := make([]string, n)
			originals := make([]string, n)
			valid := true
			for k, token := range window {
				if k > 0 && !adjacent(section.Text, window[k-1], token) {
					valid = false
					break
				}
				terms[k] = token.Term()
				originals[k] = token.Original
			}
			if !valid {
				continue
			}
			key := strings.Join(terms, " ")
			c, ok := candidates[key]
			if !ok {
				c = &yakeCandidate{
					text:       strings.Join(originals, " "),
					normalized: key,
					firstPos:   tokens[i].Position,
					length:     n,
				}
				candidates[key] = c
			}
			c.tf++
			occurrences[key] = append(occurrences[key], i)
		}
	}

	if len(candidates) == 0 {
		return nil
	}

	maxTF := 1
	for _, c := range candidates {
		if c.tf > maxTF {
			maxTF = c.tf
		}
	}
	total := len(tokens)

	for key, c := range candidates {
		neighbors := make(map[string]struct{})
		for _, at := range occurrences[key] {
			if at > 0 {
				neighbors[tokens[at-1].Term()] = struct{}{}
			}
			if end := at + c.length; end < len(tokens) {
				neighbors[tokens[end].Term()] = struct{}{}
			}
		}
		contextTotal := 2 * c.tf
		if contextTotal == 0 {
			contextTotal = 1
		}
		c.context = float64(len(neighbors)) / float64(contextTotal)

		tfNorm := float64(c.tf) / float64(maxTF)
		posNorm := 1 - float64(c.firstPos)/float64(total)
		lenNorm := float64(c.length) / float64(yakeMaxNgram)
		c.score = 0.40*(1-tfNorm) + 0.30*(1-posNorm) + 0.20*(1-c.context) + 0.10*(1-lenNorm)
	}

	ordered := make([]*yakeCandidate, 0, len(candidates))
	for _, c := range candidates {
		ordered = append(ordered, c)
	}
	sort.Slice(ordered, func(i, j int) bool {
		if ordered[i].score != ordered[j].score {
			return ordered[i].score < ordered[j].score
		}
		if ordered[i].tf != ordered[j].tf {
			return ordered[i].tf > ordered[j].tf
		}
		return ordered[i].normalized < ordered[j].normalized
	})

	sectionID := sectionIdentifier(section)
	out := make([]Keyword, 0, len(ordered))
	for i, c := range ordered {
		out = append(out, Keyword{
			Text:       c.text,
			Normalized: c.normalized,
			Section:    sectionID,
			Source:     SourceYAKE,
			Rank:       i + 1,
			Score:      math.Round(c.score*1e6) / 1e6,
		})
	}
	return out
}

func adjacent(text string, left, right nlp.Token) bool {
	if right.Position != left.Position+1 {
		return false
	}
	if left.EndOffset < 0 || right.StartOffset > len(text) || left.EndOffset > right.StartOffset {
		return true
	}
	gap := text[left.EndOffset:right.StartOffset]
	return !strings.ContainsAny(gap, "\n.!?;:")
}

func sectionIdentifier(section nlp.TextSection) string {
	if section.ID != "" {
		return section.ID
	}
	if slug := nlp.Slug(section.Title); slug != "" {
		return slug
	}
	return "document"
}
