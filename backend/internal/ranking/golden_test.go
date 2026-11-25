package ranking

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

type goldenFit struct {
	Score        int      `json:"score"`
	RawBM25      float64  `json:"raw_bm25"`
	MatchedTerms []string `json:"matched_terms"`
	MissingTerms []string `json:"missing_terms"`
}

// TestGoldenScoring locks the deterministic fit across professional areas and
// match strengths. Update with UPDATE_GOLDEN=1 only after an intentional
// algorithm change.
func TestGoldenScoring(t *testing.T) {
	svc := NewService(nil, nil)
	cases := []string{
		"exact_match", "strong_match", "partial_match", "weak_match",
		"exact_match_en", "strong_match_en", "partial_match_es", "weak_match_es",
	}
	for _, name := range cases {
		name := name
		t.Run(name, func(t *testing.T) {
			dir := filepath.Join("testdata", "scoring", name)
			resume := readFixture(t, dir, "resume.md")
			title := readFixture(t, dir, "title.txt")
			job := readFixture(t, dir, "job.md")

			fit := svc.ScoreText(resume, title, job)
			got := goldenFit{
				Score:        fit.Score,
				RawBM25:      fit.RawBM25,
				MatchedTerms: fit.MatchedTerms,
				MissingTerms: fit.MissingTerms,
			}
			if got.MatchedTerms == nil {
				got.MatchedTerms = []string{}
			}
			if got.MissingTerms == nil {
				got.MissingTerms = []string{}
			}

			path := filepath.Join(dir, "expected.json")
			if os.Getenv("UPDATE_GOLDEN") == "1" {
				writeGolden(t, path, got)
				return
			}
			var want goldenFit
			readGolden(t, path, &want)
			if got.Score != want.Score {
				t.Errorf("score = %d, want %d", got.Score, want.Score)
			}
			if math.Abs(got.RawBM25-want.RawBM25) > 1e-9 {
				t.Errorf("raw_bm25 = %v, want %v", got.RawBM25, want.RawBM25)
			}
			if !reflect.DeepEqual(got.MatchedTerms, want.MatchedTerms) {
				t.Errorf("matched = %v, want %v", got.MatchedTerms, want.MatchedTerms)
			}
			if !reflect.DeepEqual(got.MissingTerms, want.MissingTerms) {
				t.Errorf("missing = %v, want %v", got.MissingTerms, want.MissingTerms)
			}
		})
	}
}

func readFixture(t *testing.T, dir, name string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(dir, name))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func readGolden(t *testing.T, path string, v any) {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(b, v); err != nil {
		t.Fatal(err)
	}
}

func writeGolden(t *testing.T, path string, v any) {
	t.Helper()
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, append(b, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestScoringOrdering(t *testing.T) {
	svc := NewService(nil, nil)
	score := func(name string) int {
		dir := filepath.Join("testdata", "scoring", name)
		return svc.ScoreText(
			readFixture(t, dir, "resume.md"),
			readFixture(t, dir, "title.txt"),
			readFixture(t, dir, "job.md"),
		).Score
	}
	exact, strong := score("exact_match"), score("strong_match")
	partial, weak := score("partial_match"), score("weak_match")
	if !(exact > strong && strong > partial && partial > weak) {
		t.Fatalf("expected exact > strong > partial > weak, got %d > %d > %d > %d", exact, strong, partial, weak)
	}
}
