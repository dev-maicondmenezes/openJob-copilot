package filtering

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

type goldenFilter struct {
	Accepted     bool     `json:"accepted"`
	MatchCount   int      `json:"match_count"`
	MatchedTerms []string `json:"matched_terms"`
	MissingTerms []string `json:"missing_terms"`
	Score        int      `json:"score"`
}

// TestGoldenFiltering locks acceptance decisions across professional areas and
// relevance levels. Update with UPDATE_GOLDEN=1 only after an intentional
// policy or ranking change.
func TestGoldenFiltering(t *testing.T) {
	svc := NewService(nil)
	cases := []string{"relevant", "partially_relevant", "unrelated", "relevant_en", "unrelated_es"}
	for _, name := range cases {
		name := name
		t.Run(name, func(t *testing.T) {
			dir := filepath.Join("testdata", name)
			resume := readFile(t, dir, "resume.md")
			title := readFile(t, dir, "title.txt")
			job := readFile(t, dir, "job.md")
			var keywords []string
			readJSON(t, filepath.Join(dir, "keywords.json"), &keywords)

			result := svc.FilterWithKeywords(1, keywords, resume, title, job, Policy{MinMatches: 2})
			got := goldenFilter{
				Accepted:     result.Accepted,
				MatchCount:   result.MatchCount,
				MatchedTerms: result.MatchedTerms,
				MissingTerms: result.MissingTerms,
				Score:        result.Score,
			}
			if got.MatchedTerms == nil {
				got.MatchedTerms = []string{}
			}
			if got.MissingTerms == nil {
				got.MissingTerms = []string{}
			}

			path := filepath.Join(dir, "expected.json")
			if os.Getenv("UPDATE_GOLDEN") == "1" {
				writeJSON(t, path, got)
				return
			}
			var want goldenFilter
			readJSON(t, path, &want)
			if !reflect.DeepEqual(got, want) {
				t.Errorf("result = %+v, want %+v", got, want)
			}
		})
	}
}

func readFile(t *testing.T, dir, name string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(dir, name))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func readJSON(t *testing.T, path string, v any) {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(b, v); err != nil {
		t.Fatal(err)
	}
}

func writeJSON(t *testing.T, path string, v any) {
	t.Helper()
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, append(b, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestFilteringOrdering(t *testing.T) {
	svc := NewService(nil)
	filter := func(name string) Result {
		dir := filepath.Join("testdata", name)
		var keywords []string
		readJSON(t, filepath.Join(dir, "keywords.json"), &keywords)
		return svc.FilterWithKeywords(1, keywords,
			readFile(t, dir, "resume.md"),
			readFile(t, dir, "title.txt"),
			readFile(t, dir, "job.md"),
			Policy{MinMatches: 2},
		)
	}
	relevant, partial, unrelated := filter("relevant"), filter("partially_relevant"), filter("unrelated")
	if !relevant.Accepted || !partial.Accepted || unrelated.Accepted {
		t.Fatalf("acceptance = %v/%v/%v, want true/true/false", relevant.Accepted, partial.Accepted, unrelated.Accepted)
	}
	if !(relevant.MatchCount > partial.MatchCount && partial.MatchCount > unrelated.MatchCount) {
		t.Fatalf("expected relevant > partial > unrelated matches, got %d/%d/%d", relevant.MatchCount, partial.MatchCount, unrelated.MatchCount)
	}
}

func BenchmarkFilterWithKeywords(b *testing.B) {
	svc := NewService(nil)
	dir := filepath.Join("testdata", "relevant")
	resume := readFileB(b, dir, "resume.md")
	title := readFileB(b, dir, "title.txt")
	job := readFileB(b, dir, "job.md")
	keywords := []string{"cuidado ao paciente", "triagem", "prontuário eletrônico", "bacharelado em enfermagem"}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		svc.FilterWithKeywords(1, keywords, resume, title, job, Policy{MinMatches: 2})
	}
}

func readFileB(b *testing.B, dir, name string) string {
	b.Helper()
	raw, err := os.ReadFile(filepath.Join(dir, name))
	if err != nil {
		b.Fatal(err)
	}
	return string(raw)
}
