package filtering

// Result explains an acceptance/rejection decision.
type Result struct {
	JobID        int64    `json:"job_id"`
	Accepted     bool     `json:"accepted"`
	MatchedTerms []string `json:"matched_terms"`
	MissingTerms []string `json:"missing_terms"`
	MatchCount   int      `json:"match_count"`
	RawBM25      float64  `json:"raw_bm25"`
	Score        int      `json:"score"`
	Reason       string   `json:"reason"`
}
