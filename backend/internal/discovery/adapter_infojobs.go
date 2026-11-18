package discovery

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// InfoJobsAdapter scrapes public InfoJobs listing pages (pt-BR/BR, es/ES).
// Pace is 1 req/2s and the daily request quota (75) is enforced before every
// page; 429/403 become BlockedError so the orchestrator schedules a backoff.
type InfoJobsAdapter struct {
	BaseURL    string
	SiteURL    string
	HTTPClient *http.Client
}

func NewInfoJobsAdapter() *InfoJobsAdapter {
	return &InfoJobsAdapter{
		BaseURL:    "https://www.infojobs.com.br",
		SiteURL:    "https://www.infojobs.com.br",
		HTTPClient: &http.Client{Timeout: 25 * time.Second},
	}
}

func (a *InfoJobsAdapter) Slug() string { return "infojobs" }
func (a *InfoJobsAdapter) Name() string { return "InfoJobs" }

func (a *InfoJobsAdapter) DefaultCapabilities() Capabilities {
	return Capabilities{
		AccessType:            AccessScraping,
		SupportsKeywordSearch: true,
		SupportsPagination:    true,
		MaxResultsPerRun:      200,
		RateLimit:             RateLimitConfig{Requests: 1, Period: "2s", Strategy: "token_bucket"},
		RequiresDailyQuota:    true,
		DailyQuota:            75,
		QuotaUnit:             "requests",
	}
}

func (a *InfoJobsAdapter) Fetch(ctx context.Context, cfg SourceConfig, keywords []string) ([]DiscoveredJob, error) {
	limit := effectiveLimit(cfg.Capabilities.MaxResultsPerRun, 200)
	query := strings.Join(keywords, " ")
	spec := cardSpec{
		CardClasses:     []string{"js_card", "ij-OfferCard", "offer-card", "vaga"},
		TitleClasses:    []string{"js_vacancyTitle", "title", "offer-title"},
		CompanyClasses:  []string{"js_vacancyCompany", "company", "offer-company"},
		LocationClasses: []string{"js_vacancyLocation", "location", "offer-location"},
		SalaryClasses:   []string{"js_vacancySalary", "salary"},
		LinkClasses:     []string{"js_vacancyTitle", "title", "offer-title"},
	}

	var jobs []DiscoveredJob
	seen := map[string]bool{}
	remaining := cfg.DailyRemaining
	if !cfg.Capabilities.RequiresDailyQuota {
		remaining = -1 // no quota configured: unlimited
	}
	for page := 1; len(jobs) < limit; page++ {
		if remaining == 0 {
			break // quota exhausted today: return what was already collected
		}
		rawURL := buildQuery(strings.TrimRight(a.BaseURL, "/")+"/empregos.aspx", map[string]string{
			"palabra": query,
			"pagina":  fmt.Sprint(page),
		})
		doc, finalURL, err := fetchPage(ctx, a.HTTPClient, rawURL, cfg.RateLimiter)
		if err != nil {
			return jobs, fmt.Errorf("infojobs: %w", err)
		}
		if cfg.Usage != nil {
			cfg.Usage.Requests++
		}
		if remaining > 0 {
			remaining--
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
	return jobs, nil
}
