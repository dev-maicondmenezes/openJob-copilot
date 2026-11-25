package deterministic

import (
	"context"

	"github.com/maicondmenezes/openjob/internal/curriculum/adapters"
	"github.com/maicondmenezes/openjob/internal/filtering"
	"github.com/maicondmenezes/openjob/internal/keywords"
	"github.com/maicondmenezes/openjob/internal/nlp"
	"github.com/maicondmenezes/openjob/internal/ranking"
	"github.com/maicondmenezes/openjob/internal/search"
)

// Service is the shared facade consumed by the API, workers and discovery.
type Service struct {
	Analyzer *nlp.Analyzer
	Keywords *keywords.Extractor
	Ranking  *ranking.Service
	Filter   *filtering.Service

	weights search.Weights
}

// New builds the facade with the embedded NLP resources and the optional
// global index.
func New(opts Options) (*Service, error) {
	analyzer, err := nlp.NewAnalyzer()
	if err != nil {
		return nil, err
	}
	weights := opts.Weights
	if weights == (search.Weights{}) {
		weights = search.DefaultWeights()
	}
	rankingService := ranking.NewService(analyzer, opts.Index).WithWeights(weights)
	return &Service{
		Analyzer: analyzer,
		Keywords: keywords.NewExtractor(analyzer, opts.TopN),
		Ranking:  rankingService,
		Filter:   filtering.NewService(rankingService),
		weights:  weights,
	}, nil
}

// Weights returns the active BM25 weights.
func (s *Service) Weights() search.Weights { return s.weights }

// ExtractKeywords extracts deterministic keywords from a curriculum, using the
// curriculum section adapter to preserve section provenance.
func (s *Service) ExtractKeywords(ctx context.Context, curriculumMD string, topN int) (keywords.Result, error) {
	if err := ctx.Err(); err != nil {
		return keywords.Result{}, err
	}
	doc := adapters.Adapt(curriculumMD, "curriculum")
	return s.Keywords.ExtractWithTopN(doc, topN), nil
}

// ExtractKeywordStrings preserves the legacy []string resume_keywords
// contract while using the deterministic engine.
func (s *Service) ExtractKeywordStrings(ctx context.Context, curriculumMD string, topN int) ([]string, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return s.Keywords.Strings(curriculumMD, topN), nil
}

// ScoreJob computes the deterministic, explainable fit of a curriculum
// against a vacancy.
func (s *Service) ScoreJob(ctx context.Context, curriculumMD, title, description string) (ranking.FitResult, error) {
	if err := ctx.Err(); err != nil {
		return ranking.FitResult{}, err
	}
	return s.Ranking.ScoreText(curriculumMD, title, description), nil
}

// FilterJob scores a vacancy and applies the acceptance policy. When
// resumeKeywords is provided, min_matches preserves the legacy semantics.
func (s *Service) FilterJob(ctx context.Context, jobID int64, resumeKeywords []string, curriculumMD, title, description string, policy filtering.Policy) (filtering.Result, error) {
	if err := ctx.Err(); err != nil {
		return filtering.Result{}, err
	}
	if len(resumeKeywords) > 0 {
		return s.Filter.FilterWithKeywords(jobID, resumeKeywords, curriculumMD, title, description, policy), nil
	}
	return s.Filter.FilterText(curriculumMD, title, description, policy), nil
}

// IndexJob stores one vacancy in the global search index.
func (s *Service) IndexJob(ctx context.Context, jobID int64, title, description string) error {
	return s.Ranking.IndexJob(ctx, jobID, title, description)
}

// RebuildIndex replaces the global index with the supplied vacancies.
func (s *Service) RebuildIndex(ctx context.Context, jobs []ranking.JobText) error {
	return s.Ranking.RebuildIndex(ctx, jobs)
}

// Search returns vacancies ranked by weighted BM25 for the curriculum.
func (s *Service) Search(ctx context.Context, curriculumMD string, limit int) ([]search.Hit, error) {
	return s.Ranking.Search(ctx, curriculumMD, limit)
}

// IndexCount returns the number of indexed vacancies.
func (s *Service) IndexCount(ctx context.Context) (int, error) {
	return s.Ranking.IndexCount(ctx)
}
