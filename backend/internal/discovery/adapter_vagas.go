package discovery

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// VagasAdapter scrapes Vagas.com.br listing cards (pt-BR, Brazil).
//
// The listing already carries title/company, so details are an optional,
// hard-capped resource: ~20 detail pages per IP/day before a ~24h ban. The
// adapter spends at most capabilities.max_details_per_run and never exceeds
// cfg.DailyRemaining (persisted in source_daily_usage). It never touches the
// robots-disallowed /vagas/pesquisas path.
type VagasAdapter struct {
	BaseURL    string
	SiteURL    string
	HTTPClient *http.Client
}

func NewVagasAdapter() *VagasAdapter {
	return &VagasAdapter{
		BaseURL:    "https://www.vagas.com.br",
		SiteURL:    "https://www.vagas.com.br",
		HTTPClient: &http.Client{Timeout: 25 * time.Second},
	}
}

func (a *VagasAdapter) Slug() string { return "vagas" }
func (a *VagasAdapter) Name() string { return "Vagas.com.br" }

func (a *VagasAdapter) DefaultCapabilities() Capabilities {
	return Capabilities{
		AccessType:            AccessScraping,
		SupportsKeywordSearch: true,
		SupportsPagination:    true,
		MaxResultsPerRun:      200,
		RateLimit:             RateLimitConfig{Requests: 1, Period: "2s", Strategy: "token_bucket"},
		RequiresDailyQuota:    true,
		DailyQuota:            20,
		MaxDetailsPerRun:      2,
		RobotsDisallowedPaths: []string{"/vagas/pesquisas"},
	}
}

func (a *VagasAdapter) Fetch(ctx context.Context, cfg SourceConfig, keywords []string) ([]DiscoveredJob, error) {
	limit := effectiveLimit(cfg.Capabilities.MaxResultsPerRun, 200)
	query := slugifyKeyword(strings.Join(keywords, " "))
	path := "/vagas"
	if query != "" {
		path = "/vagas-de-" + query
	}
	spec := cardSpec{
		CardClasses:     []string{"vaga", "vaga-card", "job", "card-vaga"},
		TitleClasses:    []string{"link-detalhes-vaga", "vaga-title", "title", "cargo"},
		CompanyClasses:  []string{"emprVaga", "company", "empresa"},
		LocationClasses: []string{"local", "location", "vaga-local"},
		LinkClasses:     []string{"link-detalhes-vaga", "vaga-title", "title"},
	}

	var jobs []DiscoveredJob
	seen := map[string]bool{}
	for page := 1; len(jobs) < limit; page++ {
		rawURL := buildQuery(strings.TrimRight(a.BaseURL, "/")+path, map[string]string{
			"pagina": fmt.Sprint(page),
		})
		doc, finalURL, err := fetchPage(ctx, a.HTTPClient, rawURL, cfg.RateLimiter)
		if err != nil {
			return jobs, fmt.Errorf("vagas: %w", err)
		}
		if cfg.Usage != nil {
			cfg.Usage.Requests++
		}
		pageJobs := parseListingCards(doc, finalURL, spec)
		added := 0
		for _, job := range pageJobs {
			if seen[job.URL] {
				continue
			}
			seen[job.URL] = true
			job.Source = a.Slug()
			jobs = append(jobs, job)
			added++
			if len(jobs) >= limit {
				break
			}
		}
		if added == 0 || len(pageJobs) == 0 {
			break
		}
	}

	a.enrichDetails(ctx, cfg, jobs)
	return jobs, nil
}

// enrichDetails spends the tiny daily detail budget only after the listing was
// collected. Errors (including 429/403) stop further details without losing
// the listing; a hard block is surfaced by the next listing run/backoff.
func (a *VagasAdapter) enrichDetails(ctx context.Context, cfg SourceConfig, jobs []DiscoveredJob) {
	budget := cfg.Capabilities.MaxDetailsPerRun
	if budget > cfg.DailyRemaining {
		budget = cfg.DailyRemaining
	}
	if budget <= 0 || cfg.Usage == nil {
		return
	}
	for i := 0; i < len(jobs) && budget > 0; i++ {
		if isRobotsDisallowed(jobs[i].URL, cfg.Capabilities.RobotsDisallowedPaths) {
			continue
		}
		desc, err := a.fetchDetail(ctx, cfg, jobs[i].URL)
		if err != nil {
			return
		}
		cfg.Usage.Requests++
		cfg.Usage.DetailRequests++
		budget--
		if desc != "" {
			jobs[i].Description = appendMetaDescription(jobs[i].Description, desc)
		}
	}
}

func (a *VagasAdapter) fetchDetail(ctx context.Context, cfg SourceConfig, jobURL string) (string, error) {
	doc, _, err := fetchPage(ctx, a.HTTPClient, jobURL, cfg.RateLimiter)
	if err != nil {
		return "", err
	}
	for _, class := range []string{"descricao", "vaga-descricao", "job-description", "description"} {
		if n := firstByClass(doc, class); n != nil {
			if text := strings.TrimSpace(textNode(n)); text != "" {
				return text, nil
			}
		}
	}
	if n := firstByAttr(doc, "id", "descricao"); n != nil {
		return textNode(n), nil
	}
	return "", nil
}

// isRobotsDisallowed blocks a URL that matches any declared path fragment.
func isRobotsDisallowed(rawURL string, disallowed []string) bool {
	for _, path := range disallowed {
		if path != "" && strings.Contains(rawURL, path) {
			return true
		}
	}
	return false
}
