package discovery

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// CathoAdapter scrapes public Catho search pages (pt-BR, Brazil). Login-only
// pages are never accessed; the pace is a conservative 1 req/s with a pause
// between pages (the token bucket enforces it).
type CathoAdapter struct {
	BaseURL    string
	SiteURL    string
	HTTPClient *http.Client
}

func NewCathoAdapter() *CathoAdapter {
	return &CathoAdapter{
		BaseURL:    "https://www.catho.com.br",
		SiteURL:    "https://www.catho.com.br",
		HTTPClient: &http.Client{Timeout: 25 * time.Second},
	}
}

func (a *CathoAdapter) Slug() string { return "catho" }
func (a *CathoAdapter) Name() string { return "Catho" }

func (a *CathoAdapter) DefaultCapabilities() Capabilities {
	return Capabilities{
		AccessType:            AccessScraping,
		SupportsKeywordSearch: true,
		SupportsPagination:    true,
		MaxResultsPerRun:      300,
		RateLimit:             RateLimitConfig{Requests: 1, Period: "second", Strategy: "token_bucket"},
	}
}

func (a *CathoAdapter) Fetch(ctx context.Context, cfg SourceConfig, keywords []string) ([]DiscoveredJob, error) {
	limit := effectiveLimit(cfg.Capabilities.MaxResultsPerRun, 300)
	query := slugifyKeyword(strings.Join(keywords, " "))
	path := "/vagas/"
	if query != "" {
		path = "/vagas/" + query + "/"
	}
	spec := cardSpec{
		CardClasses:     []string{"job-card", "card-vaga", "vaga", "job"},
		TitleClasses:    []string{"job-title", "title", "cargo"},
		CompanyClasses:  []string{"company", "empresa", "job-company"},
		LocationClasses: []string{"location", "local", "job-location"},
		SalaryClasses:   []string{"salary", "salario"},
		LinkClasses:     []string{"job-title", "title", "cargo"},
	}

	var jobs []DiscoveredJob
	seen := map[string]bool{}
	for page := 1; len(jobs) < limit; page++ {
		rawURL := buildQuery(strings.TrimRight(a.BaseURL, "/")+path, map[string]string{
			"page": fmt.Sprint(page),
		})
		doc, finalURL, err := fetchPage(ctx, a.HTTPClient, rawURL, cfg.RateLimiter)
		if err != nil {
			return jobs, fmt.Errorf("catho: %w", err)
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
	return jobs, nil
}
