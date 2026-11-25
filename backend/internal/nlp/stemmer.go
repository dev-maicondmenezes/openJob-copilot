package nlp

import (
	"strings"
	"sync"

	"github.com/freeeve/go-stemmers"
)

var (
	stemmerOnce sync.Once
	stemmerByLang map[Language]*stemmers.Stemmer
)

func loadStemmers() {
	stemmerByLang = map[Language]*stemmers.Stemmer{
		LangPortuguese: stemmers.New(stemmers.Portuguese),
		LangEnglish:    stemmers.New(stemmers.English),
		LangSpanish:    stemmers.New(stemmers.Spanish),
	}
}

// Stem returns the Snowball stem of a single normalized word. Non-letter
// tokens (technical identifiers such as c++, c# or .net) are returned as-is.
func Stem(lang Language, word string) string {
	if word == "" || !isLettersOnly(word) {
		return word
	}
	if len([]rune(word)) < 3 {
		return word
	}
	stemmerOnce.Do(loadStemmers)
	s, ok := stemmerByLang[lang]
	if !ok {
		s = stemmerByLang[LangPortuguese]
	}
	return s.Stem(word)
}

func isLettersOnly(s string) bool {
	for _, r := range s {
		if !isLetter(r) {
			return false
		}
	}
	return true
}

func isLetter(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') ||
		(r >= 'á' && r <= 'ú') || (r >= 'Á' && r <= 'Ú') ||
		r == 'ñ' || r == 'Ñ' || r == 'ü' || r == 'Ü' || r == 'ç' || r == 'Ç'
}

func isDigit(r rune) bool { return r >= '0' && r <= '9' }

func isSpace(r rune) bool {
	return r == ' ' || r == '\t' || r == '\n' || r == '\r' || r == '\f' || r == '\v'
}

func trimLower(s string) string { return strings.TrimSpace(strings.ToLower(s)) }
