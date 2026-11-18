package discovery

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// ManfredAdapter consumes the public getManfred API (es, Spain + LATAM). The
// feed is board-wide in a single call and rejects requests without `lang`, so
// there is no pagination; keyword/region filtering happens client-side in the
// orchestrator, as with every other source.
type ManfredAdapter struct {
	BaseURL    string
	SiteURL    string
	HTTPClient *http.Client
}

func NewManfredAdapter() *ManfredAdapter {
	return &ManfredAdapter{
		BaseURL:    "https://www.getmanfred.com/api/v2/public/offers",
		SiteURL:    "https://www.getmanfred.com",
		HTTPClient: &http.Client{Timeout: 20 * time.Second},
	}
}

func (a *ManfredAdapter) Slug() string { return "getmanfred" }
func (a *ManfredAdapter) Name() string { return "getManfred" }

func (a *ManfredAdapter) DefaultCapabilities() Capabilities {
	return Capabilities{
		AccessType:            AccessAPI,
		SupportsKeywordSearch: false,
		SupportsPagination:    false,
		MaxResultsPerRun:      100,
		RateLimit:             RateLimitConfig{Requests: 1, Period: "second", Strategy: "token_bucket"},
	}
}

func (a *ManfredAdapter) Fetch(ctx context.Context, cfg SourceConfig, _ []string) ([]DiscoveredJob, error) {
	limit := effectiveLimit(cfg.Capabilities.MaxResultsPerRun, 100)
	rawURL := buildQuery(a.BaseURL, map[string]string{"lang": "es"})
	var payload any
	if err := apiGetJSON(ctx, a.HTTPClient, rawURL, cfg.RateLimiter, &payload); err != nil {
		return nil, fmt.Errorf("getmanfred: %w", err)
	}
	items := jsonItems(payload)
	jobs := make([]DiscoveredJob, 0, len(items))
	seen := map[string]bool{}
	for _, item := range items {
		job := apiJobFromMap(item, a.SiteURL)
		if job.URL == "" {
			if slug := jStr(item, "slug", "id"); slug != "" {
				job.URL = strings.TrimRight(a.SiteURL, "/") + "/offers/" + slug
			}
		}
		if job.Title == "" || job.URL == "" || seen[job.URL] {
			continue
		}
		seen[job.URL] = true
		job.Source = a.Slug()
		jobs = append(jobs, job)
		if len(jobs) >= limit {
			break
		}
	}
	return jobs, nil
}
