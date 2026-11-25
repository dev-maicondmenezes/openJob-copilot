package ranking

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/maicondmenezes/openjob/internal/nlp"
	"github.com/maicondmenezes/openjob/internal/store"
)

func testAnalyzer(t *testing.T) *nlp.Analyzer {
	t.Helper()
	a, err := nlp.NewAnalyzer()
	if err != nil {
		t.Fatal(err)
	}
	return a
}

func TestSplitJobFields(t *testing.T) {
	fields := SplitJobFields("Backend Go", `Sobre a vaga
Trabalhamos com sistemas distribuídos.

Responsabilidades
- Construir APIs
- Manter serviços

Requisitos
- Go
- PostgreSQL`)
	if fields.Title != "Backend Go" {
		t.Errorf("title = %q", fields.Title)
	}
	if !strings.Contains(fields.Responsibilities, "Construir APIs") {
		t.Errorf("responsibilities = %q", fields.Responsibilities)
	}
	if !strings.Contains(fields.Requirements, "PostgreSQL") {
		t.Errorf("requirements = %q", fields.Requirements)
	}
	if fields.Description == "" {
		t.Error("description must keep the full text")
	}
}

func TestBuildQuery(t *testing.T) {
	a := testAnalyzer(t)
	query, terms := BuildQuery(a, "Go PostgreSQL Docker Kubernetes", 3)
	if query == "" || len(terms) != 3 {
		t.Fatalf("query=%q terms=%v", query, terms)
	}
	if !strings.Contains(query, ` OR `) {
		t.Errorf("query should OR the terms: %q", query)
	}
	for _, term := range terms {
		if strings.ContainsAny(term, `"`) {
			t.Errorf("term not sanitized: %q", term)
		}
	}
}

func TestScoreTextRanksRelevantAboveUnrelated(t *testing.T) {
	a := testAnalyzer(t)
	svc := NewService(a, nil)
	title := "Desenvolvedor Backend Go"
	jd := `Requisitos
- Go
- PostgreSQL
- Docker
- Kubernetes
- APIs REST
Responsabilidades
- Construir microsserviços`
	relevant := "Experiência com Go, PostgreSQL, Docker, Kubernetes e APIs REST."
	unrelated := "Designer gráfico especializado em identidade visual, fotografia e edição de vídeo."

	rel := svc.ScoreText(relevant, title, jd)
	unrel := svc.ScoreText(unrelated, title, jd)
	if rel.Score <= unrel.Score {
		t.Fatalf("relevant %d should exceed unrelated %d (%+v / %+v)", rel.Score, unrel.Score, rel, unrel)
	}
	if rel.Score < 50 {
		t.Errorf("relevant score = %d, want >= 50", rel.Score)
	}
	if unrel.Score > 25 {
		t.Errorf("unrelated score = %d, want <= 25", unrel.Score)
	}
	if rel.RawBM25 <= 0 {
		t.Errorf("raw BM25 = %f, want positive", rel.RawBM25)
	}
	if rel.Calibrated {
		t.Error("MVP normalization must not claim validated calibration")
	}
	if rel.MechanismVersion != MechanismVersion {
		t.Errorf("mechanism version = %q", rel.MechanismVersion)
	}
}

func TestScoreTextIsDeterministic(t *testing.T) {
	a := testAnalyzer(t)
	svc := NewService(a, nil)
	cv := "Resumo\n\nEngenheiro de software com Go, PostgreSQL e Docker.\n\nProjetos\n\nAPIs REST com Kubernetes."
	jd := "Requisitos\n- Go\n- PostgreSQL\n- Docker\n- Kubernetes"
	first := svc.ScoreText(cv, "Backend", jd)
	for i := 0; i < 100; i++ {
		got := svc.ScoreText(cv, "Backend", jd)
		if got.Score != first.Score || got.RawBM25 != first.RawBM25 ||
			strings.Join(got.MatchedTerms, ",") != strings.Join(first.MatchedTerms, ",") ||
			got.Reason != first.Reason {
			t.Fatalf("non-deterministic: %+v vs %+v", got, first)
		}
	}
}

func TestScoreTextReportsMatchedAndMissingTerms(t *testing.T) {
	a := testAnalyzer(t)
	svc := NewService(a, nil)
	cv := "Desenvolvimento backend com Go, PostgreSQL, Kubernetes e Terraform."
	jd := "Requisitos\n- Go\n- PostgreSQL"
	got := svc.ScoreText(cv, "Backend", jd)
	joined := strings.Join(got.MatchedTerms, " ") + " " + strings.Join(got.MissingTerms, " ")
	for _, term := range []string{"Go", "PostgreSQL", "Kubernetes", "Terraform"} {
		if !strings.Contains(joined, term) {
			t.Errorf("terms missing %q: matched=%v missing=%v", term, got.MatchedTerms, got.MissingTerms)
		}
	}
	if !strings.Contains(got.Reason, "Lacunas:") {
		t.Errorf("reason should list gaps: %q", got.Reason)
	}
}

func TestScoreTextEmptyInputs(t *testing.T) {
	svc := NewService(testAnalyzer(t), nil)
	got := svc.ScoreText("", "Title", "Requisitos\n- Go")
	if got.Score != 0 || got.Reason == "" {
		t.Fatalf("unexpected empty result: %+v", got)
	}
}

func TestIndexSearchAndRebuild(t *testing.T) {
	a := testAnalyzer(t)
	repo, err := store.Open(filepath.Join(t.TempDir(), "index.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer repo.Close()
	svc := NewService(a, repo)
	ctx := context.Background()

	jobs := []JobText{
		{JobID: 1, Title: "Backend Go", Description: "Requisitos\n- Go\n- PostgreSQL\n- Docker"},
		{JobID: 2, Title: "Designer Gráfico", Description: "Requisitos\n- Figma\n- Photoshop"},
		{JobID: 3, Title: "Engenheiro de Dados", Description: "Requisitos\n- Python\n- SQL\n- Go"},
	}
	for _, job := range jobs {
		if err := svc.IndexJob(ctx, job.JobID, job.Title, job.Description); err != nil {
			t.Fatal(err)
		}
	}
	if n, err := svc.IndexCount(ctx); err != nil || n != 3 {
		t.Fatalf("index count = %d, %v", n, err)
	}
	stats, err := svc.IndexStats(ctx)
	if err != nil || stats.Documents != 3 {
		t.Fatalf("stats = %+v, %v", stats, err)
	}

	hits, err := svc.Search(ctx, "Go PostgreSQL Docker Kubernetes", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) == 0 || hits[0].JobID != 1 {
		t.Fatalf("hits = %+v, want job 1 first", hits)
	}
	if hits[0].RawBM25 <= 0 {
		t.Errorf("raw BM25 = %f, want positive", hits[0].RawBM25)
	}

	// Rebuild replaces the index atomically.
	if err := svc.RebuildIndex(ctx, []JobText{{JobID: 9, Title: "Only", Description: "Go"}}); err != nil {
		t.Fatal(err)
	}
	if n, _ := svc.IndexCount(ctx); n != 1 {
		t.Fatalf("after rebuild count = %d", n)
	}
	if err := svc.DeleteJob(ctx, 9); err != nil {
		t.Fatal(err)
	}
	if n, _ := svc.IndexCount(ctx); n != 0 {
		t.Fatalf("after delete count = %d", n)
	}
}

func TestScoreTextUsesGlobalCorpusStats(t *testing.T) {
	a := testAnalyzer(t)
	repo, err := store.Open(filepath.Join(t.TempDir(), "index.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer repo.Close()
	svc := NewService(a, repo)
	ctx := context.Background()
	for i := 1; i <= 20; i++ {
		title := "Vaga comum"
		desc := "Requisitos\n- comunicação\n- equipe"
		if i == 1 {
			title = "Backend Go"
			desc = "Requisitos\n- Go\n- PostgreSQL"
		}
		if err := svc.IndexJob(ctx, int64(i), title, desc); err != nil {
			t.Fatal(err)
		}
	}
	got := svc.ScoreText("Go PostgreSQL", "Backend Go", "Requisitos\n- Go\n- PostgreSQL")
	if got.RawBM25 <= 0 || got.Score <= 0 {
		t.Fatalf("expected positive score with global stats: %+v", got)
	}
	if got.ResourceVersion == "" {
		t.Error("missing resource version")
	}
}

func BenchmarkScoreText(b *testing.B) {
	a := nlp.MustAnalyzer()
	svc := NewService(a, nil)
	cv := strings.Repeat("Experiência com Go, PostgreSQL, Docker, Kubernetes e APIs REST.\n\n", 20)
	jd := "Requisitos\n- Go\n- PostgreSQL\n- Docker\n- Kubernetes\n- APIs REST"
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = svc.ScoreText(cv, "Backend Go", jd)
	}
}
