package discovery

import (
	"sort"
	"strings"
	"unicode"
)

// Stopwords in Portuguese and English + common resume template filler words.
var stopwords = map[string]bool{
	// English
	"a": true, "about": true, "above": true, "after": true, "again": true, "against": true,
	"all": true, "am": true, "an": true, "and": true, "any": true, "are": true, "as": true,
	"at": true, "be": true, "because": true, "been": true, "before": true, "being": true,
	"below": true, "between": true, "both": true, "but": true, "by": true, "can": true,
	"could": true, "did": true, "do": true, "does": true, "doing": true, "down": true,
	"during": true, "each": true, "few": true, "for": true, "from": true, "further": true,
	"had": true, "has": true, "have": true, "having": true, "he": true, "her": true,
	"here": true, "hers": true, "herself": true, "him": true, "himself": true, "his": true,
	"how": true, "i": true, "if": true, "in": true, "into": true, "is": true, "it": true,
	"its": true, "itself": true, "just": true, "me": true, "more": true, "most": true,
	"my": true, "myself": true, "no": true, "nor": true, "not": true, "now": true, "of": true,
	"off": true, "on": true, "once": true, "only": true, "or": true, "other": true, "our": true,
	"ours": true, "ourselves": true, "out": true, "over": true, "own": true, "same": true,
	"she": true, "should": true, "so": true, "some": true, "such": true, "than": true,
	"that": true, "the": true, "their": true, "theirs": true, "them": true, "themselves": true,
	"then": true, "there": true, "these": true, "they": true, "this": true, "those": true,
	"through": true, "to": true, "too": true, "under": true, "until": true, "up": true,
	"very": true, "was": true, "we": true, "were": true, "what": true, "when": true,
	"where": true, "which": true, "while": true, "who": true, "whom": true, "why": true,
	"with": true, "would": true, "you": true, "your": true, "yours": true, "yourself": true,

	// Portuguese
	"de": true, "o": true, "que": true, "e": true, "da": true, "em": true,
	"um": true, "para": true, "com": true, "não": true, "uma": true, "os": true,
	"se": true, "na": true, "por": true, "mais": true, "dos": true, "como": true,
	"mas": true, "foi": true, "ao": true, "ele": true, "das": true, "tem": true, "à": true,
	"seu": true, "sua": true, "ou": true, "ser": true, "quando": true, "muito": true, "nos": true,
	"já": true, "eu": true, "também": true, "só": true, "pelo": true, "pela": true, "até": true,
	"isso": true, "ela": true, "entre": true, "era": true, "depois": true, "sem": true,
	"mesmo": true, "aos": true, "ter": true, "seus": true, "quem": true, "nas": true,
	"esse": true, "eles": true, "estão": true, "você": true, "tinha": true, "foram": true,
	"essa": true, "num": true, "nem": true, "suas": true, "meu": true, "às": true, "minha": true,
	"têm": true, "numa": true, "pelos": true, "elas": true, "havia": true, "seja": true, "qual": true,
	"será": true, "nós": true, "tenho": true, "lhe": true, "deles": true, "essas": true, "esses": true,
	"pelas": true, "este": true, "fosse": true, "dele": true, "tu": true, "te": true, "vocês": true,
	"sobre": true, "ano": true, "anos": true, "mes": true, "meses": true, "dia": true, "dias": true,

	// Resume boilerplate noise
	"curriculum": true, "resume": true, "vitae": true, "experiência": true, "experience": true,
	"trabalho": true, "work": true, "responsabilidades": true, "responsibilities": true,
	"atuação": true, "atividades": true, "projetos": true, "projects": true,
	"skills": true, "habilidades": true, "competências": true, "competencias": true,
	"formação": true, "education": true, "graduação": true, "contato": true, "contact": true,
	"email": true, "phone": true, "telefone": true, "endereço": true, "resumo": true, "summary": true,
	"perfil": true, "profile": true, "profissional": true, "professional": true,
}

// ExtractResumeKeywords extracts the top N most frequent technical/meaningful tokens
// from resume markdown text using local heuristic frequency analysis (zero LLM cost).
func ExtractResumeKeywords(resumeMD string, topN int) []string {
	if topN <= 0 {
		topN = 20
	}

	// Split by non-alphanumeric except technical chars like #, +, ., -
	tokens := tokenize(resumeMD)
	freq := make(map[string]int)

	for _, token := range tokens {
		t := strings.ToLower(token)
		// Strip trailing/leading punctuation
		t = strings.Trim(t, ".-_/,()")
		if len(t) < 2 {
			continue
		}
		if stopwords[t] {
			continue
		}
		freq[t]++
	}

	type kv struct {
		Key   string
		Count int
	}
	var pairs []kv
	for k, v := range freq {
		pairs = append(pairs, kv{Key: k, Count: v})
	}

	sort.Slice(pairs, func(i, j int) bool {
		if pairs[i].Count == pairs[j].Count {
			return pairs[i].Key < pairs[j].Key
		}
		return pairs[i].Count > pairs[j].Count
	})

	limit := topN
	if len(pairs) < limit {
		limit = len(pairs)
	}

	result := make([]string, limit)
	for i := 0; i < limit; i++ {
		result[i] = pairs[i].Key
	}
	return result
}

func tokenize(text string) []string {
	var tokens []string
	var current strings.Builder

	for _, r := range text {
		// Alphanumeric + technical markers (C#, C++, .NET)
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '#' || r == '+' || r == '.' || r == '-' {
			current.WriteRune(r)
		} else {
			if current.Len() > 0 {
				tokens = append(tokens, current.String())
				current.Reset()
			}
		}
	}
	if current.Len() > 0 {
		tokens = append(tokens, current.String())
	}
	return tokens
}

// FilterResume evaluates whether a job passes the resume keyword adherence threshold.
// Returns (passes, matchedTerms).
// If resumeKeywords is empty, returns (true, nil) (no-op).
func FilterResume(title, description string, resumeKeywords []string, minMatches int, matchMode string) (bool, []string) {
	if len(resumeKeywords) == 0 {
		return true, nil
	}
	if minMatches <= 0 {
		minMatches = 1
	}

	text := title + "\n" + description
	var matched []string

	for _, kw := range resumeKeywords {
		kw = strings.TrimSpace(kw)
		if kw == "" {
			continue
		}
		if MatchKeyword(text, kw) {
			matched = append(matched, kw)
		}
	}

	if strings.EqualFold(matchMode, "all") {
		return len(matched) == len(resumeKeywords), matched
	}

	// Default: "any" with minMatches
	return len(matched) >= minMatches, matched
}
