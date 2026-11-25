package ranking

import (
	"context"
	"strings"

	"github.com/maicondmenezes/openjob/internal/nlp"
	"github.com/maicondmenezes/openjob/internal/search"
)

// Service composes the NLP core, the weighted BM25 scorer and (optionally) the
// global job index. It is the only ranking entry point used by consumers.
type Service struct {
	analyzer *nlp.Analyzer
	index    search.Index
	weights  search.Weights
}

// NewService builds a ranking service. index may be nil for pairwise scoring.
func NewService(analyzer *nlp.Analyzer, index search.Index) *Service {
	if analyzer == nil {
		analyzer = nlp.MustAnalyzer()
	}
	return &Service{
		analyzer: analyzer,
		index:    index,
		weights:  search.DefaultWeights(),
	}
}

// WithWeights overrides the field weights (e.g. after calibration).
func (s *Service) WithWeights(weights search.Weights) *Service {
	s.weights = weights
	return s
}

// Analyzer exposes the shared NLP core.
func (s *Service) Analyzer() *nlp.Analyzer { return s.analyzer }

// Weights returns the active field weights.
func (s *Service) Weights() search.Weights { return s.weights }

// ScoreText computes the deterministic, explainable fit of a curriculum
// against one vacancy.
func (s *Service) ScoreText(resume, title, description string) FitResult {
	return s.scoreText(resume, SplitJobFields(title, description))
}

// Search runs a weighted BM25 query over the global index.
func (s *Service) Search(ctx context.Context, resume string, limit int) ([]search.Hit, error) {
	if s.index == nil {
		return nil, nil
	}
	query, _ := BuildQuery(s.analyzer, resume, DefaultMaxQueryTerms)
	if query == "" {
		return nil, nil
	}
	return s.index.Search(ctx, query, s.weights, limit)
}

// Explain computes matched/missing terms for a job without changing the score.
func (s *Service) Explain(resume, title, description string) Explanation {
	result := s.ScoreText(resume, title, description)
	return Explanation{
		RawBM25:      result.RawBM25,
		MatchedTerms: result.MatchedTerms,
		MissingTerms: result.MissingTerms,
	}
}

// IndexJob tokenizes and stores one vacancy in the global index.
func (s *Service) IndexJob(ctx context.Context, jobID int64, title, description string) error {
	if s.index == nil {
		return nil
	}
	return s.index.Upsert(ctx, s.document(jobID, SplitJobFields(title, description)))
}

// DeleteJob removes a vacancy from the index.
func (s *Service) DeleteJob(ctx context.Context, jobID int64) error {
	if s.index == nil {
		return nil
	}
	return s.index.Delete(ctx, jobID)
}

// JobText is the minimal input needed to (re)build the index.
type JobText struct {
	JobID       int64
	Title       string
	Description string
}

// RebuildIndex replaces the whole index with the supplied vacancies.
func (s *Service) RebuildIndex(ctx context.Context, jobs []JobText) error {
	if s.index == nil {
		return nil
	}
	docs := make([]search.Document, 0, len(jobs))
	for _, job := range jobs {
		docs = append(docs, s.document(job.JobID, SplitJobFields(job.Title, job.Description)))
	}
	return s.index.Rebuild(ctx, docs)
}

// IndexStats returns corpus statistics (documents and average field lengths).
func (s *Service) IndexStats(ctx context.Context) (search.Stats, error) {
	if s.index == nil {
		return search.Stats{}, nil
	}
	return s.index.Stats(ctx)
}

// IndexCount returns how many vacancies are indexed.
func (s *Service) IndexCount(ctx context.Context) (int, error) {
	if s.index == nil {
		return 0, nil
	}
	return s.index.Count(ctx)
}

func (s *Service) document(jobID int64, fields JobFields) search.Document {
	lang := s.analyzer.DetectLanguage(fields.Title + "\n" + fields.Description)
	res := s.analyzer.Resources()
	return search.Document{
		JobID:            jobID,
		Title:            termsString(fields.Title, lang, res),
		Requirements:     termsString(fields.Requirements, lang, res),
		Responsibilities: termsString(fields.Responsibilities, lang, res),
		Description:      termsString(fields.Description, lang, res),
	}
}

func termsString(text string, lang nlp.Language, res *nlp.Resources) string {
	if strings.TrimSpace(text) == "" {
		return ""
	}
	return strings.Join(nlp.Terms(nlp.TokenizeContent(text, lang, res)), " ")
}
