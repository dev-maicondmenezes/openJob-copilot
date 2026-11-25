package nlp

import (
	"strings"
	"testing"
)

func TestEmbeddedResourcesLoad(t *testing.T) {
	res, err := EmbeddedResources()
	if err != nil {
		t.Fatal(err)
	}
	if res.Version() == "" {
		t.Fatal("missing resource version")
	}
	for _, lang := range SupportedLanguages {
		if len(res.Stopwords(lang)) == 0 {
			t.Errorf("no stopwords for %s", lang)
		}
	}
	if len(res.Checksums()) != 3 {
		t.Errorf("checksums = %v, want 3 entries", res.Checksums())
	}
}

func TestDetectLanguage(t *testing.T) {
	cases := []struct {
		text string
		want Language
	}{
		{"Desenvolvedor com experiência em sistemas e aplicações distribuídas", LangPortuguese},
		{"Senior developer with experience in scalable systems and applications", LangEnglish},
		{"Desarrollador con experiencia en sistemas y aplicaciones empresariales", LangSpanish},
		{"Go PostgreSQL Docker", LangPortuguese},
	}
	for _, tc := range cases {
		if got := DetectLanguage(tc.text, nil); got != tc.want {
			t.Errorf("DetectLanguage(%q) = %s, want %s", tc.text, got, tc.want)
		}
	}
}

func TestNormalizeUnicode(t *testing.T) {
	// Full-width and combining forms normalize to the same canonical text.
	if got := Normalize("ＡＰＩ"); got != "api" {
		t.Errorf("Normalize = %q, want api", got)
	}
	if got := Normalize("Cafe\u0301"); got != "café" {
		t.Errorf("Normalize = %q, want café", got)
	}
}

func TestTokenizePreservesOriginalAndOffsets(t *testing.T) {
	text := "Go e PostgreSQL, Docker!"
	tokens := TokenizeAll(text, LangPortuguese)
	if len(tokens) != 4 {
		t.Fatalf("tokens = %v, want 4", tokens)
	}
	for _, token := range tokens {
		if text[token.StartOffset:token.EndOffset] != token.Original {
			t.Errorf("offsets %d:%d = %q, want %q", token.StartOffset, token.EndOffset, text[token.StartOffset:token.EndOffset], token.Original)
		}
	}
	if tokens[0].Original != "Go" || tokens[0].Normalized != "go" {
		t.Errorf("unexpected first token: %+v", tokens[0])
	}
	if tokens[1].Original != "e" {
		t.Errorf("second token = %+v, want stopword 'e' present in TokenizeAll", tokens[1])
	}
}

func TestTokenizeContentDropsStopwordsAndStems(t *testing.T) {
	tokens := TokenizeContent("Desenvolvimento de aplicações e sistemas distribuídos", LangPortuguese, MustResources(t))
	terms := Terms(tokens)
	joined := strings.Join(terms, " ")
	if strings.Contains(joined, " de ") || strings.Contains(joined, " e ") {
		t.Errorf("stopwords not removed: %v", terms)
	}
	for _, term := range terms {
		if term != "desenvolv" && term != "aplic" && term != "sistem" && term != "distribuíd" {
			t.Errorf("unexpected stem %q in %v", term, terms)
		}
	}
}

func TestTokenizeContentPreservesNegations(t *testing.T) {
	for _, tc := range []struct {
		text string
		lang Language
		want string
	}{
		{"não tenho experiência", LangPortuguese, "não"},
		{"sem experiência", LangPortuguese, "sem"},
		{"not required", LangEnglish, "not"},
		{"without experience", LangEnglish, "without"},
		{"sin experiencia", LangSpanish, "sin"},
	} {
		terms := Terms(TokenizeContent(tc.text, tc.lang, MustResources(t)))
		found := false
		for _, term := range terms {
			if term == tc.want {
				found = true
			}
		}
		if !found {
			t.Errorf("negation %q removed from %q: %v", tc.want, tc.text, terms)
		}
	}
}

func TestTechnicalTokensPreserved(t *testing.T) {
	terms := Terms(TokenizeContent("Go C++ C# .NET Node.js Kubernetes", LangEnglish, MustResources(t)))
	joined := strings.Join(terms, " ")
	for _, want := range []string{"go", "c++", "c#", ".net", "node.js", "kubernetes"} {
		if !strings.Contains(joined, want) {
			t.Errorf("technical token %q missing from %v", want, terms)
		}
	}
}

func TestSegmentMarkdownSections(t *testing.T) {
	text := "# Fulano\n\n## Experiência\nGo e Docker\n\n## Habilidades\nPostgreSQL\n"
	sections := Segment(text)
	if len(sections) != 3 {
		t.Fatalf("sections = %d, want 3: %+v", len(sections), sections)
	}
	if sections[1].Title != "Experiência" || !strings.Contains(sections[1].Text, "Go e Docker") {
		t.Errorf("unexpected section: %+v", sections[1])
	}
	for _, section := range sections {
		if text[section.StartOffset:section.EndOffset] != section.Text {
			t.Errorf("section offsets do not match text: %+v", section)
		}
	}
}

func TestAnalyzeDocumentIsDeterministic(t *testing.T) {
	a := MustAnalyzer()
	doc := a.AnalyzeText("cv-1", "# Resumo\n\nEngenheiro Go com Docker.\n\n## Habilidades\nPostgreSQL e Kubernetes.", "curriculum")
	if doc.Language != string(LangPortuguese) {
		t.Errorf("language = %s", doc.Language)
	}
	if doc.ResourceVersion == "" {
		t.Error("missing resource version")
	}
	if len(doc.Sections) != 2 {
		t.Fatalf("sections = %d, want 2", len(doc.Sections))
	}
	first := TermsOf(doc.Sections[0].Tokens)
	for i := 0; i < 50; i++ {
		again := a.AnalyzeText("cv-1", doc.OriginalText, "curriculum")
		second := TermsOf(again.Sections[0].Tokens)
		if strings.Join(first, ",") != strings.Join(second, ",") {
			t.Fatalf("non-deterministic analysis: %v vs %v", first, second)
		}
	}
}

func TestAnalyzeHonorsProvidedSections(t *testing.T) {
	a := MustAnalyzer()
	text := "Alpha\nBravo Charlie"
	doc := a.Analyze(TextDocument{
		ID:           "d",
		OriginalText: text,
		Sections: []TextSection{
			{ID: "one", Title: "One", StartOffset: 0, EndOffset: 6},
			{ID: "two", Title: "Two", StartOffset: 6, EndOffset: len(text)},
		},
	})
	if len(doc.Sections) != 2 {
		t.Fatalf("sections = %d", len(doc.Sections))
	}
	if len(doc.Sections[0].Tokens) != 1 || len(doc.Sections[1].Tokens) != 2 {
		t.Errorf("unexpected tokens: %+v", doc.Sections)
	}
	if doc.Sections[1].Tokens[0].StartOffset != 0 {
		t.Errorf("token offsets must be section-relative: %+v", doc.Sections[1].Tokens[0])
	}
}

func MustResources(t *testing.T) *Resources {
	t.Helper()
	res, err := EmbeddedResources()
	if err != nil {
		t.Fatal(err)
	}
	return res
}
