package api

import (
	"testing"

	"github.com/maicondmenezes/openjob/internal/store"
)

func TestJobTextsDeduplicatesCorpus(t *testing.T) {
	jobs := []store.Job{
		{ID: 1, URL: "https://example.com/a", Title: "A", Company: "Acme", Description: "Go"},
		{ID: 2, URL: " https://EXAMPLE.com/a ", Title: "A duplicate", Company: "Acme", Description: "Go"},
		{ID: 3, Title: "Same", Company: "Acme", Description: "Docker"},
		{ID: 4, Title: " same ", Company: "acme", Description: "docker"},
		{ID: 5, Title: "Other", Company: "Acme", Description: "Kubernetes"},
	}
	texts := jobTexts(jobs)
	if len(texts) != 3 {
		t.Fatalf("texts = %d, want 3: %+v", len(texts), texts)
	}
	ids := make(map[int64]bool, len(texts))
	for _, text := range texts {
		ids[text.JobID] = true
	}
	for _, id := range []int64{1, 3, 5} {
		if !ids[id] {
			t.Errorf("job %d missing from deduplicated index", id)
		}
	}
}
