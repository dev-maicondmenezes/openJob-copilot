// Package deterministic is the composition facade of the deterministic
// engines (Fase 5 / B11). Handlers and workers depend on this facade instead
// of knowing YAKE, RAKE, Snowball or FTS5.
package deterministic

import (
	"github.com/maicondmenezes/openjob/internal/search"
)

// Options configures the facade.
type Options struct {
	// Index is the optional global job index (SQLite FTS5 in production).
	Index search.Index
	// TopN is the default keyword count (0 = keywords.DefaultTopN).
	TopN int
	// Weights overrides the BM25 field weights when non-zero.
	Weights search.Weights
}

// DefaultOptions returns the MVP defaults.
func DefaultOptions(index search.Index) Options {
	return Options{Index: index, TopN: 0, Weights: search.DefaultWeights()}
}
