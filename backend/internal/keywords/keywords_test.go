package keywords

import (
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/maicondmenezes/openjob/internal/nlp"
)

var update = flag.Bool("update", false, "update golden files")

func analyzer(t *testing.T) *nlp.Analyzer {
	t.Helper()
	a, err := nlp.NewAnalyzer()
	if err != nil {
		t.Fatal(err)
	}
	return a
}

func TestExtractYAKEIsDeterministicAndRanked(t *testing.T) {
	a := analyzer(t)
	doc := a.AnalyzeText("cv", "## Experiência\n\nDesenvolvedor Go com PostgreSQL, Docker e Kubernetes. Go, Go e mais Go.", "curriculum")
	var first Result
	for i := 0; i < 20; i++ {
		got := NewExtractor(a, 5).Extract(doc)
		if i == 0 {
			first = got
			if len(got.Keywords) == 0 {
				t.Fatal("no keywords")
			}
			if got.AlgorithmVersion != AlgorithmVersion {
				t.Errorf("algorithm version = %q", got.AlgorithmVersion)
			}
			if got.Keywords[0].Rank != 1 {
				t.Errorf("first rank = %d, want 1", got.Keywords[0].Rank)
			}
		} else if !equalResults(first, got) {
			t.Fatalf("non-deterministic result: %+v vs %+v", first, got)
		}
	}
}

func TestKeywordsKeepSectionProvenance(t *testing.T) {
	a := analyzer(t)
	text := "## Experiência\n\nDesenvolvedor Go com Docker.\n\n## Habilidades\n\nPostgreSQL e Kubernetes."
	result := NewExtractor(a, 20).ExtractText(text, "curriculum")
	sections := map[string]bool{}
	for _, kw := range result.Keywords {
		if kw.Section == "" {
			t.Errorf("keyword without section: %+v", kw)
		}
		sections[kw.Section] = true
	}
	if !sections["experiência"] || !sections["habilidades"] {
		t.Errorf("missing section provenance: %+v", result.Keywords)
	}
}

func TestExtractRespectsTopN(t *testing.T) {
	a := analyzer(t)
	text := "Go Docker Kubernetes PostgreSQL Redis Kafka Terraform Ansible Linux AWS Azure GCP"
	result := NewExtractor(a, 4).ExtractText(text, "curriculum")
	if len(result.Keywords) != 4 {
		t.Fatalf("keywords = %d, want 4: %+v", len(result.Keywords), result.Keywords)
	}
	for i, kw := range result.Keywords {
		if kw.Rank != i+1 {
			t.Errorf("rank[%d] = %d", i, kw.Rank)
		}
	}
}

func TestExtractDeduplicatesExpressions(t *testing.T) {
	a := analyzer(t)
	text := "## A\n\nDocker Kubernetes Docker Kubernetes\n\n## B\n\nDocker Kubernetes"
	result := NewExtractor(a, 20).ExtractText(text, "curriculum")
	seen := map[string]bool{}
	for _, kw := range result.Keywords {
		if seen[kw.Normalized] {
			t.Errorf("duplicate keyword %q", kw.Normalized)
		}
		seen[kw.Normalized] = true
	}
}

func TestRAKEFallbackProducesRankedKeywords(t *testing.T) {
	a := analyzer(t)
	doc := a.AnalyzeText("cv", "Gestão de projetos, liderança de equipes, análise de dados e comunicação.", "curriculum")
	keywords := ExtractRAKE(doc.Sections[0], a, nlp.LangPortuguese)
	if len(keywords) == 0 {
		t.Fatal("RAKE produced no keywords")
	}
	for i, kw := range keywords {
		if kw.Source != SourceRAKE {
			t.Errorf("source = %q", kw.Source)
		}
		if kw.Rank != i+1 {
			t.Errorf("rank[%d] = %d", i, kw.Rank)
		}
		if kw.Normalized == "" || kw.Text == "" {
			t.Errorf("incomplete keyword: %+v", kw)
		}
	}
}

func TestExtractorFallsBackToRAKEWhenYAKEIsEmpty(t *testing.T) {
	a := analyzer(t)
	text := "## Experiência\n\nGestão de projetos e liderança de equipes"
	e := NewExtractor(a, 10)
	e.yake = func(nlp.TextSection) []Keyword { return nil }
	result := e.ExtractText(text, "curriculum")
	if len(result.Keywords) == 0 {
		t.Fatal("expected RAKE fallback keywords")
	}
	found := false
	for _, kw := range result.Keywords {
		if kw.Source == SourceRAKE {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected a RAKE keyword, got %+v", result.Keywords)
	}
}

func TestStringsPreserveLegacyContract(t *testing.T) {
	a := analyzer(t)
	got := NewExtractor(a, 10).Strings("Go Docker Kubernetes PostgreSQL", 3)
	if len(got) == 0 || len(got) > 3 {
		t.Fatalf("strings = %v", got)
	}
	for _, kw := range got {
		if kw != strings.ToLower(kw) {
			t.Errorf("keyword %q is not lowercase", kw)
		}
	}
}

func TestGoldenKeywords(t *testing.T) {
	a := analyzer(t)
	cases := []string{"portuguese", "english", "spanish"}
	for _, name := range cases {
		t.Run(name, func(t *testing.T) {
			dir := filepath.Join("testdata", "keywords", name)
			input, err := os.ReadFile(filepath.Join(dir, "input.md"))
			if err != nil {
				t.Fatal(err)
			}
			result := NewExtractor(a, 10).ExtractText(string(input), "curriculum")
			got, err := json.MarshalIndent(result, "", "  ")
			if err != nil {
				t.Fatal(err)
			}
			golden := filepath.Join(dir, "expected.json")
			if *update {
				if err := os.WriteFile(golden, append(got, '\n'), 0o644); err != nil {
					t.Fatal(err)
				}
				return
			}
			want, err := os.ReadFile(golden)
			if err != nil {
				t.Fatal(err)
			}
			if strings.TrimSpace(string(want)) != strings.TrimSpace(string(got)) {
				t.Errorf("golden mismatch for %s\n--- got ---\n%s\n--- want ---\n%s", name, got, want)
			}
		})
	}
}

func BenchmarkExtract(b *testing.B) {
	a := nlp.MustAnalyzer()
	text := strings.Repeat("Experiência com Go, PostgreSQL, Docker, Kubernetes e APIs REST.\n\n", 30)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = NewExtractor(a, 20).ExtractText(text, "curriculum")
	}
}

func equalResults(a, b Result) bool {
	left, _ := json.Marshal(a)
	right, _ := json.Marshal(b)
	return string(left) == string(right)
}


