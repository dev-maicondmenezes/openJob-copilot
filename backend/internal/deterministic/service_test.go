package deterministic

import (
	"context"
	"strings"
	"testing"

	"github.com/maicondmenezes/openjob/internal/filtering"
)

func newService(t *testing.T) *Service {
	t.Helper()
	svc, err := New(DefaultOptions(nil))
	if err != nil {
		t.Fatal(err)
	}
	return svc
}

func TestExtractKeywordsUsesCurriculumSections(t *testing.T) {
	svc := newService(t)
	curriculum := `## Experiência

Desenvolvedor Go com Docker e Kubernetes.

## Habilidades

PostgreSQL e Redis.`
	result, err := svc.ExtractKeywords(context.Background(), curriculum, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Keywords) == 0 {
		t.Fatal("no keywords")
	}
	seen := map[string]bool{}
	for _, kw := range result.Keywords {
		seen[kw.Section] = true
	}
	if !seen["experience"] || !seen["skills"] {
		t.Errorf("sections = %v", seen)
	}
}

func TestExtractKeywordStrings(t *testing.T) {
	svc := newService(t)
	got, err := svc.ExtractKeywordStrings(context.Background(), "Go Docker Kubernetes PostgreSQL", 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) == 0 || len(got) > 3 {
		t.Fatalf("keywords = %v", got)
	}
	for _, kw := range got {
		if kw != strings.ToLower(kw) {
			t.Errorf("keyword %q not lowercase", kw)
		}
	}
}

func TestScoreAndFilter(t *testing.T) {
	svc := newService(t)
	resume := "Experiência com Go, PostgreSQL e Docker."
	fit, err := svc.ScoreJob(context.Background(), resume, "Backend Go", "Requisitos\n- Go\n- PostgreSQL\n- Docker")
	if err != nil {
		t.Fatal(err)
	}
	if fit.Score <= 0 {
		t.Fatalf("fit = %+v", fit)
	}
	result, err := svc.FilterJob(context.Background(), 1, []string{"go", "postgresql"}, resume, "Backend Go", "Requisitos\n- Go\n- PostgreSQL", filtering.Policy{MinMatches: 2})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Accepted || result.MatchCount != 2 {
		t.Fatalf("filter = %+v", result)
	}
}

func TestContextCancellation(t *testing.T) {
	svc := newService(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := svc.ExtractKeywords(ctx, "Go", 5); err == nil {
		t.Fatal("expected context error")
	}
}

func BenchmarkExtractKeywords(b *testing.B) {
	svc, err := New(DefaultOptions(nil))
	if err != nil {
		b.Fatal(err)
	}
	curriculum := `## Experiência

Desenvolvedor Go com Docker, Kubernetes e PostgreSQL.

## Habilidades

Redis, mensageria e APIs REST.`
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := svc.ExtractKeywordStrings(context.Background(), curriculum, 20); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkScoreJob(b *testing.B) {
	svc, err := New(DefaultOptions(nil))
	if err != nil {
		b.Fatal(err)
	}
	resume := "Go, PostgreSQL, Docker, Kubernetes, APIs REST, serviços distribuídos."
	title := "Backend Engineer (Go)"
	description := "Requisitos\n- Go\n- PostgreSQL\n- Docker\n- Kubernetes\n\nResponsabilidades\n- Construir APIs REST"
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := svc.ScoreJob(context.Background(), resume, title, description); err != nil {
			b.Fatal(err)
		}
	}
}
