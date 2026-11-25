package keywords

import (
	"strings"

	"github.com/maicondmenezes/openjob/internal/nlp"
)

// DefaultTopN is the default number of keywords returned.
const DefaultTopN = 20

// Extractor composes the NLP core and the YAKE/RAKE engines.
type Extractor struct {
	analyzer *nlp.Analyzer
	topN     int
	yake     func(nlp.TextSection) []Keyword
	rake     func(nlp.TextSection, *nlp.Analyzer, nlp.Language) []Keyword
}

// NewExtractor builds an extractor. A nil analyzer uses the embedded NLP
// resources. topN <= 0 falls back to DefaultTopN.
func NewExtractor(analyzer *nlp.Analyzer, topN int) *Extractor {
	if analyzer == nil {
		analyzer = nlp.MustAnalyzer()
	}
	if topN <= 0 {
		topN = DefaultTopN
	}
	return &Extractor{
		analyzer: analyzer,
		topN:     topN,
		yake:     ExtractYAKE,
		rake:     ExtractRAKE,
	}
}

// Analyzer exposes the shared analyzer.
func (e *Extractor) Analyzer() *nlp.Analyzer { return e.analyzer }

// Extract runs the engine over an already-built document.
func (e *Extractor) Extract(doc nlp.TextDocument) Result {
	return e.ExtractWithTopN(doc, e.topN)
}

// ExtractWithTopN runs the engine with an explicit top-N.
func (e *Extractor) ExtractWithTopN(doc nlp.TextDocument, topN int) Result {
	analyzed := e.analyzer.Analyze(doc)
	lang := nlp.ParseLanguage(analyzed.Language)

	groups := make([][]Keyword, 0, len(analyzed.Sections))
	for _, section := range analyzed.Sections {
		if len(section.Tokens) == 0 {
			continue
		}
		extracted := e.yake(section)
		if len(extracted) == 0 {
			extracted = e.rake(section, e.analyzer, lang)
		}
		if len(extracted) > 0 {
			groups = append(groups, extracted)
		}
	}

	return Result{
		Keywords:         consolidate(groups, topN),
		AlgorithmVersion: AlgorithmVersion,
		ResourceVersion:  analyzed.ResourceVersion,
	}
}

// ExtractText analyzes generic text and extracts keywords.
func (e *Extractor) ExtractText(text, sourceType string) Result {
	return e.Extract(e.analyzer.AnalyzeText("document", text, sourceType))
}

// Strings returns the extracted expressions as plain lowercase strings,
// preserving the legacy resume_keywords contract ([]string).
func (e *Extractor) Strings(text string, topN int) []string {
	result := e.ExtractText(text, "curriculum")
	if topN > 0 && len(result.Keywords) > topN {
		result.Keywords = result.Keywords[:topN]
	}
	out := make([]string, 0, len(result.Keywords))
	seen := make(map[string]struct{}, len(result.Keywords))
	for _, kw := range result.Keywords {
		text := strings.ToLower(strings.TrimSpace(kw.Text))
		if text == "" {
			continue
		}
		if _, ok := seen[text]; ok {
			continue
		}
		seen[text] = struct{}{}
		out = append(out, text)
	}
	return out
}
