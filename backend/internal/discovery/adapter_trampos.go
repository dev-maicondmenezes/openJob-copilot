package discovery

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

// TramposAdapter consumes the public Trampos REST API (pt-BR, Brazil).
type TramposAdapter struct {
	BaseURL    string
	SiteURL    string
	HTTPClient *http.Client
}

func NewTramposAdapter() *TramposAdapter {
	return &TramposAdapter{
		BaseURL:    "https://trampos.co/api/v1/opportunities",
		SiteURL:    "https://trampos.co",
		HTTPClient: &http.Client{Timeout: 20 * time.Second},
	}
}

func (a *TramposAdapter) Slug() string { return "trampos" }
func (a *TramposAdapter) Name() string { return "Trampos" }

func (a *TramposAdapter) DefaultCapabilities() Capabilities {
	return Capabilities{
		AccessType:            AccessAPI,
		SupportsKeywordSearch: false,
		SupportsPagination:    true,
		MaxResultsPerRun:      100,
		RateLimit:             RateLimitConfig{Requests: 1, Period: "second", Strategy: "token_bucket"},
	}
}

func (a *TramposAdapter) Fetch(ctx context.Context, cfg SourceConfig, _ []string) ([]DiscoveredJob, error) {
	limit := effectiveLimit(cfg.Capabilities.MaxResultsPerRun, 100)
	var jobs []DiscoveredJob
	seen := map[string]bool{}

	for page := 1; len(jobs) < limit; page++ {
		// Trampos has no reliable keyword parameter (CP14: textual search is
		// too literal); keywords are filtered client-side by the orchestrator.
		rawURL := buildQuery(a.BaseURL, map[string]string{
			"page": fmt.Sprint(page),
		})
		var payload any
		if err := apiGetJSON(ctx, a.HTTPClient, rawURL, cfg.RateLimiter, &payload); err != nil {
			return jobs, fmt.Errorf("trampos: %w", err)
		}
		items := jsonItems(payload)
		if len(items) == 0 {
			break
		}
		added := 0
		for _, item := range items {
			job := apiJobFromMap(item, a.SiteURL)
			if job.Title == "" || job.URL == "" || seen[job.URL] {
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
		if added == 0 {
			break
		}
	}
	return jobs, nil
}
