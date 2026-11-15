package discovery

import (
	"testing"
)

func TestExtractResumeKeywords(t *testing.T) {
	resume := `# João da Silva - Desenvolvedor Backend Senior
Experiência sólida com Go, Docker, Kubernetes, PostgreSQL e microsserviços.
Desenvolvimento de APIs RESTful e gRPC de alta performance em Go.
Atuação com Go, Kafka, RabbitMQ, Docker e arquitetura limpa em Go.
`
	keywords := ExtractResumeKeywords(resume, 5)
	if len(keywords) == 0 {
		t.Fatal("expected extracted keywords, got empty slice")
	}

	// "go" was mentioned 4 times, should be top keyword
	if keywords[0] != "go" {
		t.Errorf("top keyword = %q, want 'go'", keywords[0])
	}

	// Verify stopwords like "com", "de", "e" are not present
	for _, kw := range keywords {
		if stopwords[kw] {
			t.Errorf("stopword %q found in extracted keywords", kw)
		}
	}
}

func TestFilterResume(t *testing.T) {
	resumeKeywords := []string{"go", "docker", "kubernetes", "postgresql", "kafka"}

	title := "Go Software Engineer"
	desc := "Looking for an engineer with Go and Docker experience to build distributed systems with Kafka."

	// 3 matches ("go", "docker", "kafka"): minMatches=3 should pass
	ok, matches := FilterResume(title, desc, resumeKeywords, 3, "any")
	if !ok || len(matches) != 3 {
		t.Errorf("minMatches=3: got ok=%v, matches=%v", ok, matches)
	}

	// minMatches=4 should fail (only 3 matches)
	ok, matches = FilterResume(title, desc, resumeKeywords, 4, "any")
	if ok || len(matches) != 3 {
		t.Errorf("minMatches=4: got ok=%v, matches=%v", ok, matches)
	}

	// empty resume keywords is no-op (passes all)
	ok, _ = FilterResume(title, desc, nil, 3, "any")
	if !ok {
		t.Errorf("empty resume keywords should pass as no-op")
	}
}
