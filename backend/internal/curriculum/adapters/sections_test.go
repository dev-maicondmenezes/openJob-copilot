package adapters

import "testing"

func TestSectionID(t *testing.T) {
	cases := map[string]string{
		"Resumo":        "summary",
		"Experiência":   "experience",
		"Habilidades":   "skills",
		"Formação":      "education",
		"Projetos":      "projects",
		"Certificações": "certifications",
		"Idiomas":       "languages",
		"Contato":       "contact",
		"Summary":       "summary",
		"Experience":    "experience",
		"Education":     "education",
		"Otro":          "otro",
	}
	for title, want := range cases {
		if got := SectionID(title); got != want {
			t.Errorf("SectionID(%q) = %q, want %q", title, got, want)
		}
	}
}

func TestAdaptPreservesOffsetsAndSections(t *testing.T) {
	text := "# Fulano\n\n## Experiência\n\nGo e Docker\n\n## Habilidades\n\nPostgreSQL"
	doc := Adapt(text, "cv-1")
	if doc.Metadata.SourceType != "curriculum" || doc.Metadata.SourceID != "cv-1" {
		t.Errorf("metadata = %+v", doc.Metadata)
	}
	if len(doc.Sections) < 2 {
		t.Fatalf("sections = %d", len(doc.Sections))
	}
	var ids []string
	for _, section := range doc.Sections {
		ids = append(ids, section.ID)
		if text[section.StartOffset:section.EndOffset] != section.Text {
			t.Errorf("offset mismatch for %q", section.ID)
		}
	}
	found := map[string]bool{}
	for _, id := range ids {
		found[id] = true
	}
	if !found["experience"] || !found["skills"] {
		t.Errorf("missing structural ids: %v", ids)
	}
}
