package discovery

import (
	"testing"

	"github.com/maicondmenezes/openjob/internal/filtering"
)

// TestFilterEquivalenceWithLegacy locks the compatibility contract of
// min_matches: for exact-token keywords, the deterministic filter must accept
// and reject exactly like the legacy FilterResume.
func TestFilterEquivalenceWithLegacy(t *testing.T) {
	det := filtering.NewService(nil)
	cases := []struct {
		name       string
		title      string
		job        string
		keywords   []string
		minMatches int
		matchMode  string
	}{
		{
			name:       "all matched",
			title:      "Backend Go",
			job:        "Go, Docker e PostgreSQL",
			keywords:   []string{"go", "docker", "postgresql"},
			minMatches: 3,
			matchMode:  "any",
		},
		{
			name:       "below minimum",
			title:      "Backend Go",
			job:        "Go e Docker",
			keywords:   []string{"go", "docker", "postgresql"},
			minMatches: 3,
			matchMode:  "any",
		},
		{
			name:       "any with one match",
			title:      "Backend Go",
			job:        "Go e Docker",
			keywords:   []string{"go", "rust", "elixir"},
			minMatches: 1,
			matchMode:  "any",
		},
		{
			name:       "all mode complete",
			title:      "Backend Go",
			job:        "Go e Docker",
			keywords:   []string{"go", "docker"},
			minMatches: 1,
			matchMode:  "all",
		},
		{
			name:       "all mode missing one",
			title:      "Backend Go",
			job:        "Go e Docker",
			keywords:   []string{"go", "docker", "postgresql"},
			minMatches: 1,
			matchMode:  "all",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			legacyAccepted, legacyMatched := FilterResume(tc.title, tc.job, tc.keywords, tc.minMatches, tc.matchMode)
			result := det.FilterWithKeywords(0, tc.keywords, "curriculum", tc.title, tc.job, filtering.Policy{
				MinMatches: tc.minMatches,
				MatchMode:  tc.matchMode,
			})
			if result.Accepted != legacyAccepted {
				t.Errorf("accepted = %v, legacy = %v", result.Accepted, legacyAccepted)
			}
			if result.MatchCount != len(legacyMatched) {
				t.Errorf("match_count = %d, legacy = %d", result.MatchCount, len(legacyMatched))
			}
		})
	}
}

// TestFilterInflectionImprovement documents an intentional difference: the
// deterministic engine matches inflected forms via stemming, while the legacy
// substring matcher does not.
func TestFilterInflectionImprovement(t *testing.T) {
	legacyAccepted, _ := FilterResume("", "Desenvolvedores backend", []string{"desenvolvedor"}, 1, "any")
	if legacyAccepted {
		t.Fatal("legacy matcher unexpectedly matched the inflected form")
	}

	result := filtering.NewService(nil).FilterWithKeywords(
		0,
		[]string{"desenvolvedor"},
		"Desenvolvedor backend",
		"",
		"Desenvolvedores backend",
		filtering.Policy{MinMatches: 1},
	)
	if !result.Accepted {
		t.Fatalf("deterministic filter should match inflections: %+v", result)
	}
}
