package email

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"unicode"

	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"

	"github.com/maicondmenezes/openjob/internal/llm"
	"github.com/maicondmenezes/openjob/internal/store"
)

// ATSAllowlist contains known ATS relay domains.
var ATSAllowlist = map[string]bool{
	"greenhouse.io":         true,
	"us.greenhouse-mail.io": true,
	"gh-mail.io":            true,
	"lever.co":              true,
	"hire.lever.co":         true,
	"myworkday.com":         true,
	"myworkdayjobs.com":     true,
	"ashbyhq.com":           true,
	"smartrecruiters.com":   true,
	"icims.com":             true,
	"workable.com":          true,
	"jobvite.com":           true,
	"taleo.net":             true,
	"bamboohr.com":          true,
	"rippling.com":          true,
	"recruitee.com":         true,
	"gupy.io":               true,
	"kenoby.com":            true,
}

var publicWebmail = map[string]bool{
	"gmail.com":      true,
	"googlemail.com": true,
	"yahoo.com":      true,
	"hotmail.com":    true,
	"outlook.com":    true,
	"live.com":       true,
	"icloud.com":     true,
	"proton.me":      true,
	"protonmail.com": true,
}

// MatchCandidate represents an open application available for matching.
type MatchCandidate struct {
	Application store.Application
	Job         store.Job
}

// MatchResult is the outcome of matching an email against open candidacies.
type MatchResult struct {
	ApplicationID *int64
	Confidence    float64
	Reason        string
}

// Matcher runs the deterministic cascade (ADR 0032).
type Matcher struct {
	repo      store.Repository
	llmClient llm.Client
	llmCfg    *llm.Config
}

// NewMatcher creates a cascade matcher.
func NewMatcher(repo store.Repository, llmClient llm.Client, llmCfg *llm.Config) *Matcher {
	return &Matcher{
		repo:      repo,
		llmClient: llmClient,
		llmCfg:    llmCfg,
	}
}

// Match executes the cascade against the open candidates.
func (m *Matcher) Match(ctx context.Context, msg RawMessage, candidates []MatchCandidate) MatchResult {
	if len(candidates) == 0 {
		return MatchResult{Confidence: 0.0, Reason: "no_candidates"}
	}

	// 1. Thread continuity
	if msg.ThreadID != "" && m.repo != nil {
		if app, err := m.repo.FindApplicationByThreadID(ctx, msg.ThreadID); err == nil && app != nil {
			id := app.ID
			return MatchResult{
				ApplicationID: &id,
				Confidence:    1.0,
				Reason:        "thread_continuity",
			}
		}
	}

	senderDomain := extractDomain(msg.FromAddress)
	isATS := isATSDomain(senderDomain)

	normSubject := normalizeText(msg.Subject)
	normBody := normalizeText(msg.BodyText)
	normFrom := normalizeText(msg.FromName)

	// 2. Company name match in subject or body
	type scoredCandidate struct {
		cand       MatchCandidate
		confidence float64
		reason     string
	}

	var matched []scoredCandidate

	for _, c := range candidates {
		compNorm := normalizeCompanyName(c.Job.Company)
		if len(compNorm) < 2 {
			continue
		}

		inSubject := containsWord(normSubject, compNorm)
		inFrom := containsWord(normFrom, compNorm)
		inBody := containsWord(normBody, compNorm)

		if inSubject || inFrom {
			conf := 0.90
			if isATS {
				conf = 0.95 // ATS relay + company name match
			}
			matched = append(matched, scoredCandidate{
				cand:       c,
				confidence: conf,
				reason:     "company_in_subject_or_from",
			})
		} else if inBody {
			conf := 0.85
			if isATS {
				conf = 0.90
			}
			matched = append(matched, scoredCandidate{
				cand:       c,
				confidence: conf,
				reason:     "company_in_body",
			})
		}
	}

	if len(matched) == 1 {
		id := matched[0].cand.Application.ID
		return MatchResult{
			ApplicationID: &id,
			Confidence:    matched[0].confidence,
			Reason:        matched[0].reason,
		}
	} else if len(matched) > 1 {
		// Ambiguous multiple matches: pick highest or mark as suggested
		best := matched[0]
		for _, m := range matched[1:] {
			if m.confidence > best.confidence {
				best = m
			}
		}
		id := best.cand.Application.ID
		return MatchResult{
			ApplicationID: &id,
			Confidence:    0.70, // lowered due to ambiguity
			Reason:        "ambiguous_company_match",
		}
	}

	// 3. Corporate domain match (only if not an ATS and not webmail)
	if !isATS && !publicWebmail[senderDomain] && senderDomain != "" {
		for _, c := range candidates {
			jobDomain := extractDomainFromURL(c.Job.URL)
			if jobDomain != "" && (jobDomain == senderDomain || strings.HasSuffix(senderDomain, "."+jobDomain) || strings.HasSuffix(jobDomain, "."+senderDomain)) {
				id := c.Application.ID
				return MatchResult{
					ApplicationID: &id,
					Confidence:    0.85,
					Reason:        "corporate_domain_match",
				}
			}
		}
	}

	// 4. ATS domain without company name match -> candidate suggestion
	if isATS && len(candidates) > 0 {
		// Check if job title matches
		for _, c := range candidates {
			titleNorm := normalizeText(c.Job.Title)
			if len(titleNorm) >= 4 && (containsWord(normSubject, titleNorm) || containsWord(normBody, titleNorm)) {
				id := c.Application.ID
				return MatchResult{
					ApplicationID: &id,
					Confidence:    0.75, // suggested
					Reason:        "ats_job_title_match",
				}
			}
		}

		// Fallback: if only one open application exists, suggest it with 0.50
		if len(candidates) == 1 {
			id := candidates[0].Application.ID
			return MatchResult{
				ApplicationID: &id,
				Confidence:    0.50,
				Reason:        "ats_single_candidate_suggestion",
			}
		}
	}

	// 5. LLM fallback (cauda)
	if m.llmClient != nil && m.llmCfg != nil && m.llmCfg.APIKey != "" {
		if llmRes, err := m.matchWithLLM(ctx, msg, candidates); err == nil && llmRes.ApplicationID != nil {
			return llmRes
		}
	}

	return MatchResult{
		ApplicationID: nil,
		Confidence:    0.0,
		Reason:        "no_match",
	}
}

func (m *Matcher) matchWithLLM(ctx context.Context, msg RawMessage, candidates []MatchCandidate) (MatchResult, error) {
	type CandInfo struct {
		ID      int64  `json:"id"`
		Company string `json:"company"`
		Title   string `json:"title"`
	}
	var candList []CandInfo
	for _, c := range candidates {
		candList = append(candList, CandInfo{
			ID:      c.Application.ID,
			Company: c.Job.Company,
			Title:   c.Job.Title,
		})
	}
	candsJSON, _ := json.Marshal(candList)

	truncBody := msg.BodyText
	if len(truncBody) > 800 {
		truncBody = truncBody[:800]
	}

	prompt := fmt.Sprintf(`Given this email from a job recruiter/ATS and a list of active job applications, determine if the email belongs to any of the applications.

Sender: %s (%s)
Subject: %s
Body: %s

Active applications:
%s

If you are confident it matches one of the applications, output its application ID and your confidence (0.0 to 1.0). If it does not clearly match any, output null for application_id.
Respond ONLY with JSON: {"application_id": <id or null>, "confidence": <float>, "reason": "<short explanation>"}`,
		msg.FromAddress, msg.FromName, msg.Subject, truncBody, string(candsJSON))

	req := llm.CompletionRequest{
		Messages: []llm.Message{
			{Role: "system", Content: "You are an assistant matching emails to job applications."},
			{Role: "user", Content: prompt},
		},
		Temperature: 0.0,
		MaxTokens:   60,
	}

	resp, err := m.llmClient.Complete(ctx, *m.llmCfg, req)
	if err != nil {
		return MatchResult{}, err
	}

	var parsed struct {
		ApplicationID *int64  `json:"application_id"`
		Confidence    float64 `json:"confidence"`
		Reason        string  `json:"reason"`
	}

	start := strings.Index(resp, "{")
	end := strings.LastIndex(resp, "}")
	if start >= 0 && end > start {
		if err := json.Unmarshal([]byte(resp[start:end+1]), &parsed); err == nil && parsed.ApplicationID != nil {
			for _, c := range candidates {
				if c.Application.ID == *parsed.ApplicationID {
					conf := parsed.Confidence
					if conf > 0.85 {
						conf = 0.85 // cap LLM confidence
					}
					return MatchResult{
						ApplicationID: parsed.ApplicationID,
						Confidence:    conf,
						Reason:        "llm_tail:" + parsed.Reason,
					}, nil
				}
			}
		}
	}

	return MatchResult{Confidence: 0.0, Reason: "llm_no_match"}, nil
}

func extractDomain(emailAddr string) string {
	parts := strings.Split(emailAddr, "@")
	if len(parts) == 2 {
		return strings.ToLower(strings.TrimSpace(parts[1]))
	}
	return ""
}

func extractDomainFromURL(rawURL string) string {
	if rawURL == "" {
		return ""
	}
	if !strings.HasPrefix(rawURL, "http://") && !strings.HasPrefix(rawURL, "https://") {
		rawURL = "https://" + rawURL
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	host := strings.ToLower(u.Hostname())
	return strings.TrimPrefix(host, "www.")
}

func isATSDomain(domain string) bool {
	if ATSAllowlist[domain] {
		return true
	}
	for ats := range ATSAllowlist {
		if strings.HasSuffix(domain, "."+ats) {
			return true
		}
	}
	return false
}

var legalSuffixRegex = regexp.MustCompile(`(?i)\b(ltda|s/?a|inc|llc|corp|corporation|gmbh|me|epp|tecnologia|services|servicos|brasil|solutions)\b`)

func normalizeCompanyName(name string) string {
	n := normalizeText(name)
	n = legalSuffixRegex.ReplaceAllString(n, " ")
	return strings.TrimSpace(strings.Join(strings.Fields(n), " "))
}

func normalizeText(s string) string {
	t := transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
	res, _, _ := transform.String(t, s)
	res = strings.ToLower(res)
	clean := strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return r
		}
		return ' '
	}, res)
	return strings.TrimSpace(strings.Join(strings.Fields(clean), " "))
}

func containsWord(haystack, needle string) bool {
	if needle == "" || haystack == "" {
		return false
	}
	pattern := `(?i)\b` + regexp.QuoteMeta(needle) + `\b`
	if re, err := regexp.Compile(pattern); err == nil {
		return re.MatchString(haystack)
	}
	return strings.Contains(haystack, needle)
}
