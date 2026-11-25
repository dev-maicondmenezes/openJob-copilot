package nlp

// Analyzer is the shared entry point of the NLP core. It is safe for
// concurrent use: resources are read-only after loading.
type Analyzer struct {
	resources *Resources
	language  Language
}

// Option configures an Analyzer.
type Option func(*Analyzer)

// WithLanguage forces a language instead of detecting it per document.
func WithLanguage(lang Language) Option {
	return func(a *Analyzer) { a.language = lang }
}

// WithResources injects a resource set (tests, custom snapshots).
func WithResources(r *Resources) Option {
	return func(a *Analyzer) { a.resources = r }
}

// NewAnalyzer loads the embedded resources unless provided.
func NewAnalyzer(opts ...Option) (*Analyzer, error) {
	a := &Analyzer{}
	for _, opt := range opts {
		opt(a)
	}
	if a.resources == nil {
		res, err := EmbeddedResources()
		if err != nil {
			return nil, err
		}
		a.resources = res
	}
	return a, nil
}

// MustAnalyzer is a convenience for tests and wiring where embedded resources
// are known to be present.
func MustAnalyzer(opts ...Option) *Analyzer {
	a, err := NewAnalyzer(opts...)
	if err != nil {
		panic(err)
	}
	return a
}

// Resources exposes the loaded resources (read-only).
func (a *Analyzer) Resources() *Resources { return a.resources }

// Version returns the resource version stamped on every analysis.
func (a *Analyzer) Version() string { return a.resources.Version() }

// Language returns the forced language or Portuguese when detection is used.
func (a *Analyzer) Language() Language { return a.language }

// DetectLanguage classifies text, honoring the forced language.
func (a *Analyzer) DetectLanguage(text string) Language {
	if a.language != "" {
		return a.language
	}
	return DetectLanguage(text, a.resources)
}

// Normalize applies the shared normalization rules.
func (a *Analyzer) Normalize(text string) string { return Normalize(text) }

// Stem returns the Snowball stem for a language.
func (a *Analyzer) Stem(lang Language, word string) string { return Stem(lang, word) }

// TokenizeAll returns all tokens (stopwords included) of text.
func (a *Analyzer) TokenizeAll(text string, lang Language) []Token {
	return TokenizeAll(text, lang)
}

// Terms returns the matching stems of text, dropping stopwords.
func (a *Analyzer) Terms(text string, lang Language) []string {
	return Terms(TokenizeContent(text, lang, a.resources))
}

// TermsOf returns the matching stems of an already tokenized section.
func TermsOf(tokens []Token) []string { return Terms(tokens) }

// AnalyzeText builds and analyzes a generic document from raw text.
func (a *Analyzer) AnalyzeText(id, text, sourceType string) TextDocument {
	return a.Analyze(TextDocument{
		ID:           id,
		OriginalText: text,
		Metadata:     DocumentMetadata{SourceType: sourceType, SourceID: id},
	})
}

// Analyze fills the document language, sections and tokens while preserving
// the original text and byte offsets. Sections provided by the caller are
// respected; otherwise generic Markdown segmentation is used.
func (a *Analyzer) Analyze(doc TextDocument) TextDocument {
	lang := a.language
	if doc.Language != "" {
		lang = ParseLanguage(doc.Language)
	} else if lang == "" {
		lang = DetectLanguage(doc.OriginalText, a.resources)
	}
	doc.Language = string(lang)
	doc.ResourceVersion = a.resources.Version()

	raw := doc.Sections
	if len(raw) == 0 {
		segments := Segment(doc.OriginalText)
		raw = make([]TextSection, 0, len(segments))
		for _, seg := range segments {
			raw = append(raw, TextSection{
				ID:          Slug(seg.Title),
				Title:       seg.Title,
				Text:        seg.Text,
				StartOffset: seg.StartOffset,
				EndOffset:   seg.EndOffset,
			})
		}
	}

	sections := make([]TextSection, 0, len(raw))
	for i, section := range raw {
		text := section.Text
		if text == "" && section.StartOffset >= 0 && section.EndOffset <= len(doc.OriginalText) && section.EndOffset > section.StartOffset {
			text = doc.OriginalText[section.StartOffset:section.EndOffset]
		}
		tokens := TokenizeContent(text, lang, a.resources)
		if section.ID == "" {
			section.ID = Slug(section.Title)
		}
		if section.ID == "" {
			section.ID = "section-" + itoa(i)
		}
		section.Text = text
		section.Tokens = tokens
		sections = append(sections, section)
	}
	doc.Sections = sections
	return doc
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
