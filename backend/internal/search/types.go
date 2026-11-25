// Package search defines the storage-agnostic contracts of the global job
// search index (Fase 5 / B11.3). SQL and FTS5 live in the store package; the
// ranking engine only depends on this interface.
package search

import "context"

// IndexVersion identifies the indexing configuration (tokenizer + weights
// contract). Bump it whenever the indexed representation changes so a rebuild
// can be forced and cached results invalidated.
const IndexVersion = "fts5-weighted-1.0"

// Document is one indexed job. Text fields are already tokenized/normalized
// by the NLP core so the index and the query share the same terms.
type Document struct {
	JobID            int64
	Title            string
	Requirements     string
	Responsibilities string
	Description      string
}

// Weights are the per-field BM25 weights. The initial values follow the
// release plan (title 4, requirements 3, responsibilities 2, description 1).
type Weights struct {
	Title            float64
	Requirements     float64
	Responsibilities float64
	Description      float64
}

// DefaultWeights returns the experimental field weights of the MVP.
func DefaultWeights() Weights {
	return Weights{Title: 4.0, Requirements: 3.0, Responsibilities: 2.0, Description: 1.0}
}

// Hit is one ranked document. RawBM25 is positive (higher is better).
type Hit struct {
	JobID   int64
	RawBM25 float64
}

// Stats describes the indexed corpus.
type Stats struct {
	Documents              int
	AvgTitleLen            float64
	AvgRequirementsLen     float64
	AvgResponsibilitiesLen float64
	AvgDescriptionLen      float64
}

// Index is the search-index boundary consumed by the ranking engine.
type Index interface {
	Upsert(ctx context.Context, doc Document) error
	Delete(ctx context.Context, jobID int64) error
	Rebuild(ctx context.Context, docs []Document) error
	Search(ctx context.Context, match string, weights Weights, limit int) ([]Hit, error)
	TermDocFreq(ctx context.Context, term string) (int, error)
	Stats(ctx context.Context) (Stats, error)
	Count(ctx context.Context) (int, error)
}
