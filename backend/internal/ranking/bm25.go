package ranking

import (
	"context"
	"math"
	"sort"
	"strings"

	"github.com/maicondmenezes/openjob/internal/nlp"
	"github.com/maicondmenezes/openjob/internal/search"
)

const (
	bm25K1 = 1.2
	bm25B  = 0.75
)

func bm25IDF(totalDocs, docFreq int) float64 {
	if totalDocs <= 0 {
		return 0
	}
	if docFreq < 0 {
		docFreq = 0
	}
	if docFreq > totalDocs {
		docFreq = totalDocs
	}
	return math.Log(1 + (float64(totalDocs-docFreq)+0.5)/(float64(docFreq)+0.5))
}

func bm25TF(tf, docLen int, avgDocLen float64) float64 {
	if tf <= 0 || docLen <= 0 || avgDocLen <= 0 {
		return 0
	}
	lengthNorm := 1 - bm25B + bm25B*float64(docLen)/avgDocLen
	f := float64(tf)
	return f * (bm25K1 + 1) / (f + bm25K1*lengthNorm)
}

type fieldData struct {
	weight float64
	tokens []string
	freq   map[string]int
}

// scoreText computes weighted BM25 for one resume/job pair. It uses global
// corpus statistics when an index is available and falls back to the job's
// own fields as a local corpus otherwise.
func (s *Service) scoreText(resume string, fields JobFields) FitResult {
	if strings.TrimSpace(resume) == "" {
		return FitResult{
			Reason:           "Aderência de 0/100 — currículo vazio.",
			MechanismVersion: MechanismVersion,
			ResourceVersion:  s.analyzer.Version(),
			IndexVersion:     search.IndexVersion,
			Weights:          s.weights,
		}
	}

	lang := s.analyzer.DetectLanguage(resume + "\n" + fields.Description)
	res := s.analyzer.Resources()
	resumeTokens := nlp.TokenizeContent(resume, lang, res)
	if len(resumeTokens) == 0 {
		return FitResult{
			Reason:           "Aderência de 0/100 — nenhum termo relevante no currículo.",
			MechanismVersion: MechanismVersion,
			ResourceVersion:  s.analyzer.Version(),
			IndexVersion:     search.IndexVersion,
			Weights:          s.weights,
		}
	}

	fieldInputs := []struct {
		weight float64
		text   string
	}{
		{s.weights.Title, fields.Title},
		{s.weights.Requirements, fields.Requirements},
		{s.weights.Responsibilities, fields.Responsibilities},
		{s.weights.Description, fields.Description},
	}
	fieldsData := make([]fieldData, 0, len(fieldInputs))
	for _, input := range fieldInputs {
		tokens := nlp.Terms(nlp.TokenizeContent(input.text, lang, res))
		freq := make(map[string]int, len(tokens))
		for _, term := range tokens {
			freq[term]++
		}
		fieldsData = append(fieldsData, fieldData{weight: input.weight, tokens: tokens, freq: freq})
	}

	queryFreq := make(map[string]int)
	originals := make(map[string]string)
	for _, token := range resumeTokens {
		term := token.Term()
		queryFreq[term]++
		if _, ok := originals[term]; !ok {
			originals[term] = token.Original
		}
	}
	queryTerms := make([]string, 0, len(queryFreq))
	for term := range queryFreq {
		queryTerms = append(queryTerms, term)
	}
	sort.Strings(queryTerms)

	totalDocs := len(fieldsData)
	avgLens := make([]float64, len(fieldsData))
	for i, field := range fieldsData {
		avgLens[i] = float64(len(field.tokens))
		if avgLens[i] <= 0 {
			avgLens[i] = 1
		}
	}

	if s.index != nil {
		if stats, err := s.index.Stats(context.Background()); err == nil && stats.Documents > 0 {
			totalDocs = stats.Documents
			if stats.AvgTitleLen > 0 {
				avgLens[0] = stats.AvgTitleLen
			}
			if stats.AvgRequirementsLen > 0 {
				avgLens[1] = stats.AvgRequirementsLen
			}
			if stats.AvgResponsibilitiesLen > 0 {
				avgLens[2] = stats.AvgResponsibilitiesLen
			}
			if stats.AvgDescriptionLen > 0 {
				avgLens[3] = stats.AvgDescriptionLen
			}
		}
	}

	maxWeight := 0.0
	for _, field := range fieldsData {
		if field.weight > maxWeight {
			maxWeight = field.weight
		}
	}

	var raw, ideal float64
	contributions := make(map[string]float64, len(queryTerms))
	present := make(map[string]bool, len(queryTerms))

	for _, term := range queryTerms {
		df := 0
		if s.index != nil && totalDocs > 0 {
			if n, err := s.index.TermDocFreq(context.Background(), term); err == nil {
				df = n
			}
		}
		if df == 0 {
			for _, field := range fieldsData {
				if field.freq[term] > 0 {
					df++
				}
			}
		}
		idf := bm25IDF(totalDocs, df)
		ideal += idf * maxWeight

		var termScore float64
		for i, field := range fieldsData {
			tf := field.freq[term]
			if tf == 0 {
				continue
			}
			present[term] = true
			termScore += field.weight * idf * bm25TF(tf, len(field.tokens), avgLens[i])
		}
		raw += termScore
		contributions[term] = termScore
	}

	fit := 0.0
	if ideal > 0 {
		fit = 100 * raw / ideal
	}
	if fit < 0 {
		fit = 0
	}
	if fit > 100 {
		fit = 100
	}

	matched := make([]string, 0, len(present))
	missing := make([]string, 0, len(queryTerms))
	matchedContribution := make(map[string]float64, len(present))
	for _, term := range queryTerms {
		display := displayTerm(term, originals)
		if present[term] {
			matched = append(matched, display)
			matchedContribution[display] = contributions[term]
		} else {
			missing = append(missing, display)
		}
	}
	sort.SliceStable(matched, func(i, j int) bool {
		ti, tj := matchedContribution[matched[i]], matchedContribution[matched[j]]
		if ti != tj {
			return ti > tj
		}
		return matched[i] < matched[j]
	})
	sort.Strings(missing)

	score := int(math.Round(fit))
	return FitResult{
		Score:            score,
		RawBM25:          raw,
		Calibrated:       false,
		MatchedTerms:     matched,
		MissingTerms:     missing,
		Reason:           buildReason(score, raw, matched, missing, len(queryTerms)),
		Weights:          s.weights,
		MechanismVersion: MechanismVersion,
		ResourceVersion:  s.analyzer.Version(),
		IndexVersion:     search.IndexVersion,
	}
}

func displayTerm(stem string, originals map[string]string) string {
	if original := originals[stem]; original != "" {
		return original
	}
	return stem
}
