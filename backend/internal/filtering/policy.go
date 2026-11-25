// Package filtering applies acceptance policies on top of ranking results
// (Fase 5 / B11.4). It does not implement an independent similarity metric:
// ranking scores the vacancy, filtering only decides whether it is accepted.
package filtering

// Policy is the initial acceptance policy.
//
// MinMatches counts distinct resume keywords found in the vacancy, preserving
// the legacy min_matches semantics. MinScore is an optional threshold over the
// ranking fit score. MatchMode "all" requires every resume keyword to match;
// the default ("any") only requires MinMatches.
type Policy struct {
	MinMatches int
	MinScore   *float64
	MatchMode  string
}

// DefaultPolicy mirrors the discovery defaults (3 matches, "any").
func DefaultPolicy() Policy {
	return Policy{MinMatches: 3, MatchMode: "any"}
}

// Normalize clamps invalid values to safe defaults.
func (p Policy) Normalize() Policy {
	if p.MinMatches < 0 {
		p.MinMatches = 0
	}
	if p.MatchMode == "" {
		p.MatchMode = "any"
	}
	return p
}
