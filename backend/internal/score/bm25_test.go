package score

import (
	"strings"
	"testing"
)

func TestScoreBM25RejectsEmptyInputs(t *testing.T) {
	t.Parallel()
	if _, err := ScoreBM25("", "Go developer"); err != ErrEmptyCurriculum {
		t.Fatalf("empty curriculum: got %v, want %v", err, ErrEmptyCurriculum)
	}
	if _, err := ScoreBM25("Experiência com Go", " "); err != ErrEmptyJobDescription {
		t.Fatalf("empty JD: got %v, want %v", err, ErrEmptyJobDescription)
	}
	if _, err := ScoreBM25("Experiência com Go", "<script>alert(1)</script>"); err != ErrEmptyJobDescription {
		t.Fatalf("HTML-only JD: got %v, want %v", err, ErrEmptyJobDescription)
	}
}

func TestScoreBM25ReturnsZeroWhenThereAreNoTokens(t *testing.T) {
	t.Parallel()
	got, err := ScoreBM25("!!! --- ...", "### ***")
	if err != nil {
		t.Fatal(err)
	}
	if got.Score != 0 || got.Reason == "" {
		t.Fatalf("unexpected result: %+v", got)
	}
}

func TestScoreBM25RanksRelevantResumeAboveUnrelatedResume(t *testing.T) {
	t.Parallel()
	jd := "Desenvolvedor backend Go com experiência em PostgreSQL, Docker, Kubernetes e APIs REST."
	relevant := "Experiência profissional\n\nDesenvolvedor backend Go. Construção de APIs REST e microsserviços.\n\nTecnologias\n\nPostgreSQL, Docker, Kubernetes, Go."
	unrelated := "Experiência profissional\n\nDesigner gráfico especializado em identidade visual, fotografia e edição de vídeo.\n\nTecnologias\n\nPhotoshop, Illustrator, Figma."
	relevantResult, err := ScoreBM25(relevant, jd)
	if err != nil {
		t.Fatal(err)
	}
	unrelatedResult, err := ScoreBM25(unrelated, jd)
	if err != nil {
		t.Fatal(err)
	}
	if relevantResult.Score <= unrelatedResult.Score {
		t.Fatalf("relevant score %d should exceed unrelated score %d", relevantResult.Score, unrelatedResult.Score)
	}
	if relevantResult.Score < 60 {
		t.Errorf("relevant score = %d, want >= 60", relevantResult.Score)
	}
	if unrelatedResult.Score > 20 {
		t.Errorf("unrelated score = %d, want <= 20", unrelatedResult.Score)
	}
}

func TestScoreBM25IsDeterministic(t *testing.T) {
	t.Parallel()
	cv := "Resumo\n\nEngenheiro de software com Go, PostgreSQL e Docker.\n\nProjetos\n\nAPIs REST com Kubernetes."
	jd := "Pessoa desenvolvedora Go com PostgreSQL, Docker e Kubernetes."
	first, err := ScoreBM25(cv, jd)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 100; i++ {
		got, err := ScoreBM25(cv, jd)
		if err != nil {
			t.Fatal(err)
		}
		if got != first {
			t.Fatalf("iteration %d: got %+v, want %+v", i, got, first)
		}
	}
}

func TestScoreBM25ScoreIsBounded(t *testing.T) {
	t.Parallel()
	cv := strings.Repeat("Go PostgreSQL Docker Kubernetes ", 100)
	jd := strings.Repeat("Go PostgreSQL Docker Kubernetes ", 30)
	got, err := ScoreBM25(cv, jd)
	if err != nil {
		t.Fatal(err)
	}
	if got.Score < 0 || got.Score > 100 {
		t.Fatalf("score = %d, outside [0,100]", got.Score)
	}
}

func TestScoreBM25ReportsMatchedTermsAndGaps(t *testing.T) {
	t.Parallel()
	cv := "Desenvolvimento backend com Go e PostgreSQL."
	jd := "Desenvolvimento backend com Go, PostgreSQL, Kubernetes e Terraform."
	got, err := ScoreBM25(cv, jd)
	if err != nil {
		t.Fatal(err)
	}
	for _, term := range []string{"go", "postgresql", "kubernetes", "terraform"} {
		if !strings.Contains(got.Reason, term) {
			t.Errorf("reason %q missing %q", got.Reason, term)
		}
	}
	if !strings.Contains(got.Reason, "Lacunas:") {
		t.Errorf("reason %q should identify gaps", got.Reason)
	}
}

func TestScoreBM25HandlesHTMLAndEntities(t *testing.T) {
	t.Parallel()
	cv := "Desenvolvedor Go com experiência em APIs REST."
	jd := "<p>Desenvolvedor <strong>Go</strong></p><ul><li>APIs REST</li><li>PostgreSQL &amp; Docker</li></ul>"
	got, err := ScoreBM25(cv, jd)
	if err != nil {
		t.Fatal(err)
	}
	if got.Score <= 0 || got.Score >= 100 {
		t.Fatalf("expected partial positive match, got %d", got.Score)
	}
}

func TestSnowballPortugueseMatchesInflections(t *testing.T) {
	t.Parallel()
	cv := tokenizeWithLanguage("Desenvolvimento de aplicações e sistemas distribuídos", langPortuguese)
	jd := tokenizeWithLanguage("Desenvolver aplicação e sistema distribuído", langPortuguese)
	cvSet := makeWordSet(strings.Join(cv, " "))
	for _, term := range jd {
		if _, ok := cvSet[term]; !ok {
			t.Errorf("expected normalized term %q in CV tokens %v; JD tokens %v", term, cv, jd)
		}
	}
}

func TestSnowballEnglishMatchesInflections(t *testing.T) {
	t.Parallel()
	cv := tokenizeWithLanguage("Building scalable applications and managing distributed systems", langEnglish)
	jd := tokenizeWithLanguage("Build scalable application and manage distributed system", langEnglish)
	cvSet := makeWordSet(strings.Join(cv, " "))
	for _, term := range jd {
		if _, ok := cvSet[term]; !ok {
			t.Errorf("expected normalized term %q in CV tokens %v; JD tokens %v", term, cv, jd)
		}
	}
}

func TestSnowballSpanishMatchesInflections(t *testing.T) {
	t.Parallel()
	cv := tokenizeWithLanguage("Desarrollando aplicaciones empresariales y gestionando sistemas", langSpanish)
	jd := tokenizeWithLanguage("Desarrollar aplicación empresarial y gestionar sistema", langSpanish)
	cvSet := makeWordSet(strings.Join(cv, " "))
	for _, term := range jd {
		if _, ok := cvSet[term]; !ok {
			t.Errorf("expected normalized term %q in CV tokens %v; JD tokens %v", term, cv, jd)
		}
	}
}

func TestTechnicalTermsArePreserved(t *testing.T) {
	t.Parallel()
	got := tokenizeWithLanguage("Go C R C++ C# .NET Node.js React.js Kubernetes", langEnglish)
	joined := strings.Join(got, " ")
	for _, term := range []string{"go", "c", "r", "c++", "c#", ".net", "node.js", "react.js", "kubernetes"} {
		if !strings.Contains(joined, term) {
			t.Errorf("technical token %q missing from %q", term, joined)
		}
	}
}

func TestTechnicalAliasesAreCanonicalized(t *testing.T) {
	t.Parallel()
	got := tokenizeWithLanguage("Golang dotnet NodeJS ReactJS K8s postgres JS TS", langEnglish)
	joined := strings.Join(got, " ")
	for _, term := range []string{"go", ".net", "node.js", "react.js", "kubernetes", "postgresql", "javascript", "typescript"} {
		if !strings.Contains(joined, term) {
			t.Errorf("canonical token %q missing from %q", term, joined)
		}
	}
}

func TestNegationsAreNotRemovedAsStopwords(t *testing.T) {
	t.Parallel()
	cases := []struct {
		text string
		lang language
		want string
	}{
		{"não tenho experiência", langPortuguese, "não"},
		{"sem experiência", langPortuguese, "sem"},
		{"not required", langEnglish, "not"},
		{"without experience", langEnglish, "without"},
		{"sin experiencia", langSpanish, "sin"},
	}
	for _, tc := range cases {
		tokens := tokenizeWithLanguage(tc.text, tc.lang)
		found := false
		for _, token := range tokens {
			if token == tc.want {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("negation %q was removed from %q: %v", tc.want, tc.text, tokens)
		}
	}
}

func TestDetectLanguage(t *testing.T) {
	t.Parallel()
	cases := []struct {
		text string
		want language
	}{
		{"Desenvolvedor com experiência em sistemas e aplicações", langPortuguese},
		{"Senior developer with experience in scalable systems", langEnglish},
		{"Desarrollador con experiencia en sistemas y aplicaciones", langSpanish},
	}
	for _, tc := range cases {
		if got := detectLanguage(tc.text); got != tc.want {
			t.Errorf("detectLanguage(%q) = %v, want %v", tc.text, got, tc.want)
		}
	}
}

func TestBM25IDFAndTermFrequency(t *testing.T) {
	t.Parallel()
	common, rare := bm25IDF(4, 4), bm25IDF(4, 1)
	if common <= 0 || rare <= common {
		t.Fatalf("unexpected IDFs: common=%f rare=%f", common, rare)
	}
	once, twice, many := bm25TermFrequency(1, 100, 100), bm25TermFrequency(2, 100, 100), bm25TermFrequency(100, 100, 100)
	if !(once < twice && twice < many && many < bm25K1+1) {
		t.Fatalf("unexpected TF saturation: %f %f %f", once, twice, many)
	}
}

func BenchmarkScoreBM25(b *testing.B) {
	cv := strings.Repeat("Experiência com Go, PostgreSQL, Docker, Kubernetes e APIs REST.\n\n", 20)
	jd := strings.Repeat("Desenvolvedor backend Go com PostgreSQL, Docker, Kubernetes e APIs REST. ", 5)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := ScoreBM25(cv, jd); err != nil {
			b.Fatal(err)
		}
	}
}
