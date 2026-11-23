package classify

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/maicondmenezes/openjob/internal/llm"
)

// Closed vocabulary for email signals (ADR 0033 / spec CP17).
const (
	SignalAck         = "ack"
	SignalScreening   = "screening"
	SignalInterview   = "interview"
	SignalAssessment  = "assessment"
	SignalOffer       = "offer"
	SignalRejection   = "rejection"
	SignalInfoRequest = "info_request"
	SignalOther       = "other"
)

var ValidSignals = map[string]bool{
	SignalAck:         true,
	SignalScreening:   true,
	SignalInterview:   true,
	SignalAssessment:  true,
	SignalOffer:       true,
	SignalRejection:   true,
	SignalInfoRequest: true,
	SignalOther:       true,
}

// Classifier classifies email messages into controlled signals.
type Classifier struct {
	llmClient llm.Client
	llmCfg    *llm.Config
}

// NewClassifier creates a classifier, optionally with LLM fallback.
func NewClassifier(llmClient llm.Client, cfg *llm.Config) *Classifier {
	return &Classifier{
		llmClient: llmClient,
		llmCfg:    cfg,
	}
}

// Result contains the classified signal and confidence.
type Result struct {
	Signal     string
	Confidence float64
	Source     string // "rule" | "llm" | "fallback"
}

// Multilingual regex patterns (PT-BR, EN-US, ES-419).
var (
	rejectionPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)\b(unfortunately|we regret|not moving forward|decided to pursue other|pursuing other candidates|not selected for this role)\b`),
		regexp.MustCompile(`(?i)\b(infelizmente|lamentamos|n[aã]o daremos continuidade|n[aã]o seguir[aá] no processo|n[aã]o foi selecionad[oa]|outro candidato)\b`),
		regexp.MustCompile(`(?i)\b(lamentamos|no continuaremos|no seleccionado|hemos decidido no seguir|otras candidaturas)\b`),
		regexp.MustCompile(`(?i)\b(wishing you the best in your job search|desejamos sucesso em sua jornada|deseamos mucho [eé]xito)\b`),
	}

	offerPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)\b(offer letter|job offer|pleased to offer you|formal offer|extending an offer)\b`),
		regexp.MustCompile(`(?i)\b(oferta de trabalho|proposta de emprego|carta oferta|proposta formal|formalizar proposta|temos uma proposta)\b`),
		regexp.MustCompile(`(?i)\b(oferta de empleo|carta de oferta|propuesta de trabajo|propuesta formal)\b`),
	}

	interviewPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)\b(interview invitation|schedule an interview|invitation to interview|technical interview|interview scheduled|congratulations.*invited to interview)\b`),
		regexp.MustCompile(`(?i)\b(convite para entrevista|entrevista t[eé]cnica|agendar entrevista|conversar sobre a vaga|bater um papo|agendamento de entrevista)\b`),
		regexp.MustCompile(`(?i)\b(invitaci[oó]n a entrevista|agendar entrevista|entrevista t[eé]cnica|entrevista de trabajo)\b`),
		regexp.MustCompile(`(?i)\b(calendly\.com|meet\.google\.com|zoom\.us/j/|teams\.microsoft\.com/l/meetup)\b`),
	}

	assessmentPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)\b(coding challenge|technical assessment|online assessment|take-home assignment|hackerrank|codility|codesignal)\b`),
		regexp.MustCompile(`(?i)\b(desafio t[eé]cnico|teste t[eé]cnico|teste pr[aá]tico|etapa de teste|prova t[eé]cnica)\b`),
		regexp.MustCompile(`(?i)\b(prueba t[eé]cnica|desaf[ií]o t[eé]cnico|evaluaci[oó]n t[eé]cnica)\b`),
	}

	screeningPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)\b(initial screening|recruiter screen|introductory chat|screening call)\b`),
		regexp.MustCompile(`(?i)\b(triagem|bate-papo inicial|primeiro contato|alinhamento inicial)\b`),
		regexp.MustCompile(`(?i)\b(cribado inicial|charla inicial|primer contacto)\b`),
	}

	ackPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)\b(application received|thank you for applying|received your application|application confirmed|thanks for applying)\b`),
		regexp.MustCompile(`(?i)\b(recebemos sua candidatura|candidatura confirmada|obrigado por se candidatar|recebemos seu curr[ií]culo|inscri[cç][aã]o confirmada)\b`),
		regexp.MustCompile(`(?i)\b(hemos recibido tu postulaci[oó]n|candidatura recibida|gracias por postularte|postulaci[oó]n confirmada)\b`),
	}

	infoRequestPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)\b(additional information|please fill out|more details needed|action required.*application)\b`),
		regexp.MustCompile(`(?i)\b(informa[cç][oõ]es adicionais|mais detalhes|preencher formul[aá]rio|a[cç][aã]o necess[aá]ria)\b`),
		regexp.MustCompile(`(?i)\b(informaci[oó]n adicional|completar formulario|acci[oó]n requerida)\b`),
	}
)

// Classify determines the email signal and confidence score.
func (c *Classifier) Classify(ctx context.Context, subject, body string) Result {
	combined := subject + "\n" + body

	// Rejection has highest priority to avoid false positive advances
	for _, p := range rejectionPatterns {
		if p.MatchString(combined) {
			return Result{Signal: SignalRejection, Confidence: 0.95, Source: "rule"}
		}
	}

	// Formal offer
	for _, p := range offerPatterns {
		if p.MatchString(combined) {
			return Result{Signal: SignalOffer, Confidence: 0.95, Source: "rule"}
		}
	}

	// Interview
	for _, p := range interviewPatterns {
		if p.MatchString(combined) {
			return Result{Signal: SignalInterview, Confidence: 0.90, Source: "rule"}
		}
	}

	// Assessment / Challenge
	for _, p := range assessmentPatterns {
		if p.MatchString(combined) {
			return Result{Signal: SignalAssessment, Confidence: 0.90, Source: "rule"}
		}
	}

	// Initial screening
	for _, p := range screeningPatterns {
		if p.MatchString(combined) {
			return Result{Signal: SignalScreening, Confidence: 0.85, Source: "rule"}
		}
	}

	// Acknowledgement / received
	for _, p := range ackPatterns {
		if p.MatchString(combined) {
			return Result{Signal: SignalAck, Confidence: 0.90, Source: "rule"}
		}
	}

	// Info request
	for _, p := range infoRequestPatterns {
		if p.MatchString(combined) {
			return Result{Signal: SignalInfoRequest, Confidence: 0.85, Source: "rule"}
		}
	}

	// LLM tail if available
	if c.llmClient != nil && c.llmCfg != nil && c.llmCfg.APIKey != "" {
		if llmRes, err := c.classifyWithLLM(ctx, subject, body); err == nil && llmRes.Signal != SignalOther {
			return llmRes
		}
	}

	return Result{Signal: SignalOther, Confidence: 0.30, Source: "fallback"}
}

func (c *Classifier) classifyWithLLM(ctx context.Context, subject, body string) (Result, error) {
	truncBody := body
	if len(truncBody) > 1000 {
		truncBody = truncBody[:1000]
	}

	prompt := fmt.Sprintf(`You are an email classifier for job application tracking.
Classify the following email into EXACTLY ONE of these categories:
- ack (application received, confirmation of application)
- screening (initial recruiter phone screen or intro call)
- interview (technical or hiring manager interview invitation or scheduling)
- assessment (coding challenge, test, take-home project)
- offer (formal job offer letter)
- rejection (rejection, not selected, passing on candidate)
- info_request (request for more documents or details)
- other (unrelated, newsletter, spam, or cannot determine)

Subject: %s
Body snippet: %s

Respond ONLY with a JSON object: {"signal": "<category>", "confidence": <0.0-1.0>}`, subject, truncBody)

	req := llm.CompletionRequest{
		Messages: []llm.Message{
			{Role: "system", Content: "You classify job application emails into a strict vocabulary."},
			{Role: "user", Content: prompt},
		},
		Temperature: 0.0,
		MaxTokens:   50,
	}

	resp, err := c.llmClient.Complete(ctx, *c.llmCfg, req)
	if err != nil {
		return Result{}, err
	}

	var parsed struct {
		Signal     string  `json:"signal"`
		Confidence float64 `json:"confidence"`
	}
	// Extract JSON
	start := strings.Index(resp, "{")
	end := strings.LastIndex(resp, "}")
	if start >= 0 && end > start {
		if err := json.Unmarshal([]byte(resp[start:end+1]), &parsed); err == nil {
			signal := strings.ToLower(strings.TrimSpace(parsed.Signal))
			if ValidSignals[signal] {
				conf := parsed.Confidence
				if conf <= 0 {
					conf = 0.70
				}
				if conf > 0.95 {
					conf = 0.95
				}
				return Result{
					Signal:     signal,
					Confidence: conf,
					Source:     "llm",
				}, nil
			}
		}
	}

	return Result{Signal: SignalOther, Confidence: 0.30, Source: "llm"}, nil
}
