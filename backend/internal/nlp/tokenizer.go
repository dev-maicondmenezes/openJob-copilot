package nlp

import (
	"regexp"
	"strings"
)

// tokenPattern matches Unicode words while keeping technical identifiers
// such as c++, c#, .net, node.js and ci/cd as a single token.
var tokenPattern = regexp.MustCompile("(?i)\\.net|asp\\.net|node\\.js|react\\.js|vue\\.js|next\\.js|nuxt\\.js|c\\+\\+|c#|f#|ci/cd|[\\p{L}\\p{N}]+(?:[._+#-][\\p{L}\\p{N}]+)*")

var technicalTerms = wordSet("go c r c++ c# f# .net asp.net node.js react.js vue.js next.js nuxt.js ci/cd aws azure gcp sql mysql postgresql postgres mongodb redis kafka grpc rest graphql http https html css xml json yaml toml api apis sdk ide git github gitlab bitbucket docker kubernetes k8s terraform ansible linux unix windows ios android lambda jwt oauth ml ai nlp llm rag javascript typescript python")

func wordSet(words string) map[string]struct{} {
	set := make(map[string]struct{})
	for _, word := range strings.Fields(words) {
		set[word] = struct{}{}
	}
	return set
}

// TokenizeAll returns every token of text, including stopwords, preserving
// original forms and byte offsets.
func TokenizeAll(text string, lang Language) []Token {
	indices := tokenPattern.FindAllStringIndex(text, -1)
	tokens := make([]Token, 0, len(indices))
	for _, idx := range indices {
		original := text[idx[0]:idx[1]]
		normalized := canonicalTechnicalToken(NormalizeWord(original))
		if normalized == "" {
			continue
		}
		tokens = append(tokens, Token{
			Original:    original,
			Normalized:  normalized,
			Stem:        stemToken(lang, normalized),
			StartOffset: idx[0],
			EndOffset:   idx[1],
			Position:    len(tokens),
		})
	}
	return tokens
}

// TokenizeContent tokenizes text and drops stopwords, keeping negations and
// technical identifiers. This is the token stream used by the engines.
func TokenizeContent(text string, lang Language, res *Resources) []Token {
	all := TokenizeAll(text, lang)
	tokens := make([]Token, 0, len(all))
	for _, token := range all {
		if res != nil && res.IsStopword(lang, token.Normalized) {
			continue
		}
		if len([]rune(token.Normalized)) < 2 {
			if _, ok := technicalTerms[token.Normalized]; !ok {
				continue
			}
		}
		token.Position = len(tokens)
		tokens = append(tokens, token)
	}
	return tokens
}

func stemToken(lang Language, normalized string) string {
	if _, technical := technicalTerms[normalized]; technical {
		return normalized
	}
	if _, negation := negationTerms[normalized]; negation {
		return normalized
	}
	return Stem(lang, normalized)
}

func canonicalTechnicalToken(token string) string {
	switch token {
	case "golang":
		return "go"
	case "dotnet":
		return ".net"
	case "nodejs":
		return "node.js"
	case "reactjs":
		return "react.js"
	case "vuejs":
		return "vue.js"
	case "nextjs":
		return "next.js"
	case "nuxtjs":
		return "nuxt.js"
	case "k8s":
		return "kubernetes"
	case "postgres":
		return "postgresql"
	case "js":
		return "javascript"
	case "ts":
		return "typescript"
	case "py":
		return "python"
	default:
		return token
	}
}

// DetectLanguage classifies text using the relative frequency of stopwords
// from each bundled language. It is intentionally lightweight and defaults to
// Portuguese; it is not a general-purpose language detector.
func DetectLanguage(text string, res *Resources) Language {
	if res == nil {
		if r, err := EmbeddedResources(); err == nil {
			res = r
		}
	}
	normalized := Normalize(text)
	words := tokenPattern.FindAllString(normalized, -1)
	scores := make(map[Language]int, len(SupportedLanguages))
	for _, word := range words {
		word = canonicalTechnicalToken(word)
		for _, lang := range SupportedLanguages {
			if _, ok := res.Stopwords(lang)[word]; ok {
				scores[lang]++
			}
		}
	}
	best := LangPortuguese
	for _, lang := range SupportedLanguages {
		if scores[lang] > scores[best] {
			best = lang
		}
	}
	return best
}
