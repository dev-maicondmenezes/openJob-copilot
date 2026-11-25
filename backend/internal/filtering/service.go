package filtering

import (
	"fmt"
	"sort"
	"strings"

	"github.com/maicondmenezes/openjob/internal/nlp"
	"github.com/maicondmenezes/openjob/internal/ranking"
)

// Service consumes the ranking engine and applies a Policy.
type Service struct {
	ranking *ranking.Service
}

// NewService wires the filter to the shared ranking engine.
func NewService(r *ranking.Service) *Service {
	if r == nil {
		r = ranking.NewService(nil, nil)
	}
	return &Service{ranking: r}
}

// Ranking exposes the composed ranking engine.
func (s *Service) Ranking() *ranking.Service { return s.ranking }

// FilterText scores and filters one vacancy using the resume terms as the
// match vocabulary.
func (s *Service) FilterText(resume, title, description string, policy Policy) Result {
	fit := s.ranking.ScoreText(resume, title, description)
	return s.Evaluate(0, fit, fit.MatchedTerms, fit.MissingTerms, policy)
}

// FilterWithKeywords filters using an explicit resume keyword list, which
// preserves the legacy min_matches semantics (distinct keywords found in the
// vacancy). The score still comes from the ranking engine.
func (s *Service) FilterWithKeywords(jobID int64, keywords []string, resume, title, description string, policy Policy) Result {
	fit := s.ranking.ScoreText(resume, title, description)
	if len(keywords) == 0 {
		return s.Evaluate(jobID, fit, nil, nil, policy)
	}

	lang := s.ranking.Analyzer().DetectLanguage(title + "\n" + description)
	res := s.ranking.Analyzer().Resources()
	jobTerms := make(map[string]struct{})
	for _, term := range nlp.Terms(nlp.TokenizeContent(title+"\n"+description, lang, res)) {
		jobTerms[term] = struct{}{}
	}

	matched := make([]string, 0, len(keywords))
	missing := make([]string, 0, len(keywords))
	for _, keyword := range keywords {
		keyword = strings.TrimSpace(keyword)
		if keyword == "" {
			continue
		}
		terms := nlp.Terms(nlp.TokenizeContent(keyword, lang, res))
		if len(terms) == 0 {
			continue
		}
		all := true
		for _, term := range terms {
			if _, ok := jobTerms[term]; !ok {
				all = false
				break
			}
		}
		if all {
			matched = append(matched, keyword)
		} else {
			missing = append(missing, keyword)
		}
	}
	sort.Strings(matched)
	sort.Strings(missing)
	return s.Evaluate(jobID, fit, matched, missing, policy)
}

// Evaluate applies the policy to a ranking result and explains the decision.
func (s *Service) Evaluate(jobID int64, fit ranking.FitResult, matched, missing []string, policy Policy) Result {
	policy = policy.Normalize()
	result := Result{
		JobID:        jobID,
		MatchedTerms: append([]string(nil), matched...),
		MissingTerms: append([]string(nil), missing...),
		MatchCount:   len(matched),
		RawBM25:      fit.RawBM25,
		Score:        fit.Score,
	}

	if policy.MinScore != nil && float64(fit.Score) < *policy.MinScore {
		result.Reason = fmt.Sprintf("Rejeitada: score %d abaixo do mínimo %.0f.", fit.Score, *policy.MinScore)
		return result
	}
	if strings.EqualFold(policy.MatchMode, "all") {
		if len(missing) > 0 {
			result.Reason = fmt.Sprintf("Rejeitada: modo 'all' exige todos os termos (%d/%d correspondidos).",
				len(matched), len(matched)+len(missing))
			return result
		}
		result.Accepted = true
		result.Reason = fmt.Sprintf("Aceita: todos os %d termos correspondidos (score %d).", len(matched), fit.Score)
		return result
	}
	if policy.MinMatches > 0 && len(matched) < policy.MinMatches {
		result.Reason = fmt.Sprintf("Rejeitada: %d/%d correspondências (mínimo %d).",
			len(matched), len(matched)+len(missing), policy.MinMatches)
		return result
	}
	result.Accepted = true
	if policy.MinMatches > 0 {
		result.Reason = fmt.Sprintf("Aceita: %d correspondências (mínimo %d, score %d).",
			len(matched), policy.MinMatches, fit.Score)
	} else {
		result.Reason = fmt.Sprintf("Aceita: score %d.", fit.Score)
	}
	return result
}
