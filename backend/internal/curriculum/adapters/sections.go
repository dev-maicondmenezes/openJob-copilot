// Package adapters maps the existing curriculum representation onto the
// generic NLP text contracts (Fase 5). It performs structural section
// identification only; it never interprets evidence, rewrites experiences or
// performs tailoring.
package adapters

import (
	"strings"

	"github.com/maicondmenezes/openjob/internal/nlp"
)

// sectionIDs maps known curriculum headings (in the three supported
// languages) to stable section identifiers. The identifiers are what the
// keyword engine reports as provenance.
var sectionIDs = []struct {
	terms []string
	id    string
}{
	{[]string{"resumo", "summary", "profile", "perfil", "sobre", "about", "objetivo"}, "summary"},
	{[]string{"experiência", "experiencia", "experience", "histórico", "historico", "trajetória"}, "experience"},
	{[]string{"habilidades", "skills", "competências", "competencias", "tecnologias", "conhecimentos"}, "skills"},
	{[]string{"formação", "formacao", "education", "educação", "academic"}, "education"},
	{[]string{"projetos", "projects", "portfolio", "portfólio"}, "projects"},
	{[]string{"certificações", "certificacoes", "certifications", "cursos", "licenças", "licencas"}, "certifications"},
	{[]string{"idiomas", "languages"}, "languages"},
	{[]string{"contato", "contact", "links"}, "contact"},
}

// SectionID maps a curriculum heading to a stable identifier.
func SectionID(title string) string {
	normalized := strings.ToLower(strings.TrimSpace(title))
	for _, mapping := range sectionIDs {
		for _, term := range mapping.terms {
			if normalized == term || strings.HasPrefix(normalized, term+" ") || strings.Contains(normalized, term) {
				return mapping.id
			}
		}
	}
	if slug := nlp.Slug(title); slug != "" {
		return slug
	}
	return "other"
}

// Adapt converts a curriculum markdown into a generic NLP document, mapping
// each structural section to its stable identifier.
func Adapt(contentMD, sourceID string) nlp.TextDocument {
	segments := nlp.Segment(contentMD)
	sections := make([]nlp.TextSection, 0, len(segments))
	for i, segment := range segments {
		id := SectionID(segment.Title)
		if id == "other" && segment.Title == "" {
			id = "summary"
			if i > 0 {
				id = "section-" + itoa(i)
			}
		}
		sections = append(sections, nlp.TextSection{
			ID:          id,
			Title:       segment.Title,
			Text:        segment.Text,
			StartOffset: segment.StartOffset,
			EndOffset:   segment.EndOffset,
		})
	}
	return nlp.TextDocument{
		ID:           sourceID,
		OriginalText: contentMD,
		Sections:     sections,
		Metadata:     nlp.DocumentMetadata{SourceType: "curriculum", SourceID: sourceID},
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}
