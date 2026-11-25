package nlp

import "strings"

// Language is a BCP-47-ish identifier restricted to the languages supported
// by the MVP. The zero value is Portuguese, the product's primary language.
type Language string

const (
	LangPortuguese Language = "pt-BR"
	LangEnglish    Language = "en-US"
	LangSpanish    Language = "es-419"
)

// SupportedLanguages lists the languages with bundled resources.
var SupportedLanguages = []Language{LangPortuguese, LangEnglish, LangSpanish}

// ParseLanguage maps an external code (pt, pt-BR, en, es, es-419...) to a
// supported Language, falling back to Portuguese.
func ParseLanguage(code string) Language {
	code = strings.ToLower(strings.TrimSpace(code))
	switch {
	case strings.HasPrefix(code, "en"):
		return LangEnglish
	case strings.HasPrefix(code, "es"):
		return LangSpanish
	case strings.HasPrefix(code, "pt"):
		return LangPortuguese
	default:
		return LangPortuguese
	}
}

// StemmerCode returns the short Snowball code for a language.
func (l Language) StemmerCode() string {
	switch l {
	case LangEnglish:
		return "en"
	case LangSpanish:
		return "es"
	default:
		return "pt"
	}
}
