package filtering

import (
	"strings"
	"testing"

	"github.com/maicondmenezes/openjob/internal/ranking"
)

func newFilter() *Service {
	return NewService(ranking.NewService(nil, nil))
}

func TestFilterWithKeywordsMinMatches(t *testing.T) {
	svc := newFilter()
	keywords := []string{"go", "docker", "kubernetes", "postgresql", "kafka"}
	title := "Go Software Engineer"
	desc := "Looking for an engineer with Go and Docker experience to build distributed systems with Kafka."

	got := svc.FilterWithKeywords(1, keywords, "", title, desc, Policy{MinMatches: 3, MatchMode: "any"})
	if !got.Accepted || got.MatchCount != 3 {
		t.Fatalf("min=3: %+v", got)
	}
	got = svc.FilterWithKeywords(1, keywords, "", title, desc, Policy{MinMatches: 4, MatchMode: "any"})
	if got.Accepted || got.MatchCount != 3 {
		t.Fatalf("min=4 should reject: %+v", got)
	}
}

func TestFilterMatchModeAll(t *testing.T) {
	svc := newFilter()
	title := "Backend"
	desc := "Go and Docker"
	if got := svc.FilterWithKeywords(1, []string{"go", "docker"}, "", title, desc, Policy{MatchMode: "all"}); !got.Accepted {
		t.Fatalf("all should pass: %+v", got)
	}
	if got := svc.FilterWithKeywords(1, []string{"go", "python"}, "", title, desc, Policy{MatchMode: "all"}); got.Accepted {
		t.Fatalf("all should fail: %+v", got)
	}
}

func TestFilterMinScore(t *testing.T) {
	svc := newFilter()
	title := "Backend Go"
	desc := "Requisitos\n- Go\n- PostgreSQL\n- Docker"
	resume := "Experiência com Go, PostgreSQL e Docker."
	fit := svc.Ranking().ScoreText(resume, title, desc)
	high := float64(100)
	low := float64(0)

	if got := svc.FilterText(resume, title, desc, Policy{MinScore: &low}); !got.Accepted {
		t.Fatalf("low threshold should accept: %+v", got)
	}
	if got := svc.FilterText(resume, title, desc, Policy{MinScore: &high}); got.Accepted {
		t.Fatalf("high threshold should reject: %+v (score %d)", got, fit.Score)
	}
}

func TestFilterReasonsAreExplicit(t *testing.T) {
	svc := newFilter()
	got := svc.FilterWithKeywords(1, []string{"go", "rust"}, "", "Backend", "Go developer", Policy{MinMatches: 2})
	if got.Accepted {
		t.Fatal("expected rejection")
	}
	if !strings.Contains(strings.ToLower(got.Reason), "rejeitada") {
		t.Errorf("reason = %q", got.Reason)
	}
	if got.MissingTerms == nil {
		t.Error("missing terms must be reported")
	}
}

func TestFilterIsDeterministic(t *testing.T) {
	svc := newFilter()
	keywords := []string{"go", "docker", "kubernetes", "postgresql"}
	first := svc.FilterWithKeywords(7, keywords, "Go Docker", "Backend Go", "Requisitos\n- Go\n- Docker\n- Kubernetes", Policy{MinMatches: 2})
	for i := 0; i < 50; i++ {
		got := svc.FilterWithKeywords(7, keywords, "Go Docker", "Backend Go", "Requisitos\n- Go\n- Docker\n- Kubernetes", Policy{MinMatches: 2})
		if got.Accepted != first.Accepted || got.MatchCount != first.MatchCount ||
			strings.Join(got.MatchedTerms, ",") != strings.Join(first.MatchedTerms, ",") ||
			got.Reason != first.Reason {
			t.Fatalf("non-deterministic: %+v vs %+v", got, first)
		}
	}
}

func TestPolicyNormalize(t *testing.T) {
	p := Policy{MinMatches: -2}.Normalize()
	if p.MinMatches != 0 || p.MatchMode != "any" {
		t.Fatalf("normalize = %+v", p)
	}
}
