package score

import (
	"fmt"
	"math"
	"regexp"
	"sort"
	"strings"
	"unicode"

	"github.com/freeeve/go-stemmers"
	"golang.org/x/text/unicode/norm"
)

const (
	bm25K1 = 1.2
	bm25B  = 0.75
)

type language int

const (
	langPortuguese language = iota
	langEnglish
	langSpanish
)

var (
	tokenPattern     = regexp.MustCompile("(?i)\\.net|asp\\.net|node\\.js|react\\.js|vue\\.js|next\\.js|nuxt\\.js|c\\+\\+|c#|f#|ci/cd|[\\p{L}\\p{N}]+(?:[._+#-][\\p{L}\\p{N}]+)*")
	paragraphPattern = regexp.MustCompile("\\n[ \\t]*\\n+")

	portugueseStemmer = stemmers.New(stemmers.Portuguese)
	englishStemmer    = stemmers.New(stemmers.English)
	spanishStemmer    = stemmers.New(stemmers.Spanish)

	stopwords = map[language]map[string]struct{}{
		langPortuguese: makeWordSet("a à ao aos aquela aquelas aquele aqueles aquilo as até com como contra da das de dela delas dele deles depois desde do dos e é ela elas ele eles em enquanto entre era essa essas esse esses esta estas este estes eu foi foram há isso isto já lhe lhes lo mas me mesmo meu meus minha minhas muito na nas não nem no nos nós nossa nossas nosso nossos num numa o os ou para pela pelas pelo pelos por qual quando que quem se sem ser será seu seus sua suas são só também te tem tendo ter teu teus ti todo todos tu tua tuas tudo um uma umas uns você vocês vos"),
		langEnglish:    makeWordSet("a about above after again against all am an and any are as at be because been before being below between both but by can did do does doing down during each few for from further had has have having he her here hers herself him himself his how i if in into is it its itself just me more most my myself no nor not of off on once only or other our ours ourselves out over own same she should so some such than that the their theirs them themselves then there these they this those through to too under until up very was we were what when where which while who whom why will with you your yours yourself yourselves"),
		langSpanish:    makeWordSet("a al algo algunas algunos ante antes como con contra cual cuando de del desde donde durante e el ella ellas ellos en entre era es esa esas ese eso esos esta estas este esto estos fue ha hasta hay la las le les lo los más me mi mis mucha muchas mucho muchos muy nada ni no nos nosotros nuestra nuestras nuestro nuestros o otra otras otro otros para pero poca poco por porque que quien quienes se sea según ser si sin sobre son su sus también te tener tiene todo todos tu tus un una unas uno unos y ya yo"),
	}
	technicalTerms = makeWordSet("go c r c++ c# f# .net asp.net node.js react.js vue.js next.js nuxt.js ci/cd aws azure gcp sql mysql postgresql postgres mongodb redis kafka grpc rest graphql http https html css xml json yaml toml api apis sdk ide git github gitlab bitbucket docker kubernetes k8s terraform ansible linux unix windows ios android lambda jwt oauth ml ai nlp llm rag javascript typescript python")
	// Negation changes the meaning of a requirement, so these words must not
	// be discarded as ordinary stopwords (though this scorer does not perform
	// full grammatical negation-scope analysis).
	negationTerms = makeWordSet("não nao sem nunca jamais no nor not without never sin")
)

func makeWordSet(words string) map[string]struct{} {
	set := make(map[string]struct{})
	for _, word := range strings.Fields(words) {
		set[word] = struct{}{}
	}
	return set
}

func stemmerFor(lang language) *stemmers.Stemmer {
	switch lang {
	case langEnglish:
		return englishStemmer
	case langSpanish:
		return spanishStemmer
	default:
		return portugueseStemmer
	}
}

// detectLanguage chooses a Snowball stemmer using language-specific stopwords.
// It is deliberately lightweight and defaults to Portuguese, the product's
// primary language; it is not intended as a general-purpose language detector.
func detectLanguage(text string) language {
	tokens := tokenPattern.FindAllString(norm.NFKC.String(strings.ToLower(text)), -1)
	scores := [3]int{}
	for _, token := range tokens {
		for lang := language(0); lang <= langSpanish; lang++ {
			if _, ok := stopwords[lang][token]; ok {
				scores[lang]++
			}
		}
	}
	best := langPortuguese
	for lang := langEnglish; lang <= langSpanish; lang++ {
		if scores[lang] > scores[best] {
			best = lang
		}
	}
	return best
}

func tokenize(text string) []string {
	return tokenizeWithLanguage(text, detectLanguage(text))
}

// tokenizeWithLanguage performs compatibility normalization, stopword
// removal and Snowball stemming while preserving technical identifiers.
func tokenizeWithLanguage(text string, lang language) []string {
	text = norm.NFKC.String(strings.ToLower(text))
	parts := tokenPattern.FindAllString(text, -1)
	tokens := make([]string, 0, len(parts))
	stemmer := stemmerFor(lang)

	for _, part := range parts {
		token := strings.TrimSpace(part)
		if token == "" {
			continue
		}
		if _, stop := stopwords[lang][token]; stop {
			if _, negation := negationTerms[token]; !negation {
				continue
			}
		}
		token = canonicalTechnicalToken(token)
		if len([]rune(token)) == 1 {
			if _, ok := technicalTerms[token]; !ok {
				continue
			}
		}
		if _, technical := technicalTerms[token]; !technical && isLettersOnly(token) && len([]rune(token)) >= 4 {
			token = stemmer.Stem(token)
		}
		tokens = append(tokens, token)
	}
	return tokens
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

func isLettersOnly(s string) bool {
	for _, r := range s {
		if !unicode.IsLetter(r) {
			return false
		}
	}
	return true
}

func termFreq(tokens []string) map[string]int {
	freq := make(map[string]int, len(tokens))
	for _, token := range tokens {
		freq[token]++
	}
	return freq
}

func uniqueTerms(tokens []string) []string {
	seen := make(map[string]struct{}, len(tokens))
	unique := make([]string, 0, len(tokens))
	for _, token := range tokens {
		if _, ok := seen[token]; ok {
			continue
		}
		seen[token] = struct{}{}
		unique = append(unique, token)
	}
	return unique
}

type termHit struct {
	term  string
	score float64
}

// resumeSegments treats paragraphs as local documents for estimating term
// specificity within this single curriculum. No global corpus is required.
func resumeSegments(curriculum string, lang language) [][]string {
	normalized := strings.ReplaceAll(curriculum, "\r\n", "\n")
	paragraphs := paragraphPattern.Split(normalized, -1)
	segments := make([][]string, 0, len(paragraphs))
	for _, paragraph := range paragraphs {
		if tokens := tokenizeWithLanguage(paragraph, lang); len(tokens) > 0 {
			segments = append(segments, tokens)
		}
	}
	if len(segments) == 0 {
		if tokens := tokenizeWithLanguage(curriculum, lang); len(tokens) > 0 {
			segments = append(segments, tokens)
		}
	}
	return segments
}

// bm25IDF uses positive smoothed IDF over the sections of this curriculum.
// With a one-section CV all present terms have equal IDF; this limitation is
// inherent to pairwise scoring without an external document corpus.
func bm25IDF(totalSections, matchingSections int) float64 {
	if totalSections <= 0 || matchingSections < 0 {
		return 0
	}
	if matchingSections > totalSections {
		matchingSections = totalSections
	}
	return math.Log(1 + (float64(totalSections-matchingSections)+0.5)/(float64(matchingSections)+0.5))
}

func bm25TermFrequency(tf, docLen int, avgDocLen float64) float64 {
	if tf <= 0 || docLen <= 0 || avgDocLen <= 0 {
		return 0
	}
	lengthNorm := 1 - bm25B + bm25B*float64(docLen)/avgDocLen
	frequency := float64(tf)
	return frequency * (bm25K1 + 1) / (frequency + bm25K1*lengthNorm)
}

// ScoreBM25 computes a deterministic lexical fit score (0–100) for exactly
// one curriculum and one job description. Snowball stemming improves
// morphological matching in Portuguese, English and Spanish. It does not
// infer semantic equivalence or estimate hiring probability.
func ScoreBM25(curriculumMD, jdText string) (Result, error) {
	if strings.TrimSpace(curriculumMD) == "" {
		return Result{}, ErrEmptyCurriculum
	}
	if strings.TrimSpace(jdText) == "" {
		return Result{}, ErrEmptyJobDescription
	}

	cleanedJD := CleanJD(jdText)
	if strings.TrimSpace(cleanedJD) == "" {
		return Result{}, ErrEmptyJobDescription
	}

	// Use one language decision for both texts so their tokens always share
	// the same stemming rules, even when one input is too short to classify.
	pairLanguage := detectLanguage(curriculumMD + "\\n" + cleanedJD)
	jdTokens := tokenizeWithLanguage(cleanedJD, pairLanguage)
	segments := resumeSegments(curriculumMD, pairLanguage)
	if len(segments) == 0 || len(jdTokens) == 0 {
		return Result{Score: 0, Reason: "Aderência de 0/100 — não foi possível identificar termos relevantes no currículo ou na vaga."}, nil
	}

	// Reuse section tokens as the complete CV token stream. This avoids
	// stemming and allocating the same curriculum tokens a second time.
	cvTokens := make([]string, 0)
	for _, segment := range segments {
		cvTokens = append(cvTokens, segment...)
	}

	cvFreq := termFreq(cvTokens)
	jdFreq := termFreq(jdTokens)
	queryTerms := uniqueTerms(jdTokens)

	docFreq := make(map[string]int, len(cvFreq))
	totalSegmentLength := 0
	for _, segment := range segments {
		totalSegmentLength += len(segment)
		for _, term := range uniqueTerms(segment) {
			docFreq[term]++
		}
	}

	docLen := len(cvTokens)
	avgDocLen := float64(totalSegmentLength) / float64(len(segments))
	if avgDocLen <= 0 {
		avgDocLen = float64(docLen)
	}

	hits := make([]termHit, 0, len(queryTerms))
	gaps := make([]string, 0, len(queryTerms))
	var weightedScore, idealScore float64

	for _, term := range queryTerms {
		idf := bm25IDF(len(segments), docFreq[term])
		queryWeight := 1 + math.Log1p(float64(jdFreq[term]-1))
		idealScore += idf * queryWeight * bm25TermFrequency(1, docLen, avgDocLen)

		tf := cvFreq[term]
		if tf == 0 {
			gaps = append(gaps, term)
			continue
		}
		contribution := idf * queryWeight * bm25TermFrequency(tf, docLen, avgDocLen)
		weightedScore += contribution
		hits = append(hits, termHit{term: term, score: contribution})
	}

	fit := 0.0
	if idealScore > 0 {
		fit = 100 * weightedScore / idealScore
	}
	fit = math.Max(0, math.Min(100, fit))
	normalized := int(math.Round(fit))

	sort.Slice(hits, func(i, j int) bool {
		if hits[i].score != hits[j].score {
			return hits[i].score > hits[j].score
		}
		return hits[i].term < hits[j].term
	})
	sort.Strings(gaps)

	return Result{Score: normalized, Reason: buildReason(normalized, hits, gaps, len(queryTerms))}, nil
}

func buildReason(score int, hits []termHit, gaps []string, totalReqTerms int) string {
	if len(hits) == 0 {
		reason := fmt.Sprintf("Aderência de %d/100 — nenhum termo da vaga foi encontrado no currículo.", score)
		if len(gaps) > 0 {
			gapN := min(5, len(gaps))
			reason += " Lacunas: " + strings.Join(gaps[:gapN], ", ") + "."
		}
		return reason
	}
	topN := min(5, len(hits))
	matches := make([]string, topN)
	for i := 0; i < topN; i++ {
		matches[i] = hits[i].term
	}
	reason := fmt.Sprintf("Aderência de %d/100 — termos encontrados: %s (%d/%d termos-chave).",
		score, strings.Join(matches, ", "), len(hits), totalReqTerms)
	if len(gaps) > 0 {
		gapN := min(5, len(gaps))
		reason += " Lacunas: " + strings.Join(gaps[:gapN], ", ") + "."
	}
	return reason
}
