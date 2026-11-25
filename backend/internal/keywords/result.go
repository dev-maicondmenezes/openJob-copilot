// Package keywords implements deterministic keyword extraction over generic
// NLP documents (Fase 5 / B11.2). YAKE is the primary algorithm; RAKE is a
// fallback when YAKE produces no usable candidates for a section. Raw scores
// from the two algorithms are never compared: only their per-section rank is
// used during consolidation.
package keywords

// AlgorithmVersion identifies the extraction pipeline and its scoring rules.
const AlgorithmVersion = "yake-rake-1.0"

// Sources of a keyword.
const (
	SourceYAKE = "yake"
	SourceRAKE = "rake"
)

// Keyword is one extracted expression with provenance.
type Keyword struct {
	Text       string  `json:"text"`
	Normalized string  `json:"normalized"`
	Section    string  `json:"section"`
	Source     string  `json:"source"`
	Rank       int     `json:"rank"`
	Score      float64 `json:"score"`
}

// Result is the output of one extraction run.
type Result struct {
	Keywords         []Keyword `json:"keywords"`
	AlgorithmVersion string    `json:"algorithm_version"`
	ResourceVersion  string    `json:"resource_version,omitempty"`
}
