// Package ranking implements deterministic, explainable relevance between a
// curriculum and the global corpus of vacancies (Fase 5 / B11.3).
//
// The engine uses weighted BM25 (k1=1.2, b=0.75) and keeps the raw ranking
// score separate from the interface fit score. The latter is a provisional,
// monotonic normalization of the raw score against an ideal document; it is
// not a probability and must not be presented as a validated percentage.
package ranking

import "github.com/maicondmenezes/openjob/internal/search"

// MechanismVersion identifies the scoring configuration.
const MechanismVersion = "weighted-bm25-1.0"

// FitResult is the explainable relevance of one vacancy.
type FitResult struct {
	// Score is the provisional 0–100 interface value derived from RawBM25.
	Score int `json:"score"`
	// RawBM25 is the raw weighted BM25 score (higher is better).
	RawBM25 float64 `json:"raw_bm25"`
	// Calibrated reports whether a validated calibration was applied. The MVP
	// uses a monotonic normalization, so this stays false.
	Calibrated   bool     `json:"calibrated"`
	MatchedTerms []string `json:"matched_terms"`
	MissingTerms []string `json:"missing_terms"`
	Reason       string   `json:"reason"`
	// Weights and versions make the result reproducible.
	Weights          search.Weights `json:"-"`
	MechanismVersion string         `json:"mechanism_version"`
	ResourceVersion  string         `json:"resource_version"`
	IndexVersion     string         `json:"index_version"`
}

// Explanation carries the provenance used to justify a match.
type Explanation struct {
	JobID        int64    `json:"job_id"`
	RawBM25      float64  `json:"raw_bm25"`
	MatchedTerms []string `json:"matched_terms"`
	MissingTerms []string `json:"missing_terms"`
}
