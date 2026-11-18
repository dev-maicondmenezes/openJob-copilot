package discovery

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// GupyAdapter consumes the public Gupy job portal API (pt-BR, Brazil).
type GupyAdapter struct {
	BaseURL    string
	SiteURL    string
	HTTPClient *http.Client
}

func NewGupyAdapter() *GupyAdapter {
	return &GupyAdapter{
		BaseURL:    "https://portal.api.gupy.io/api/v1/jobs",
		SiteURL:    "https://portal.gupy.io",
		HTTPClient: &http.Client{Timeout: 20 * time.Second},
	}
}

func (a *GupyAdapter) Slug() string { return "gupy" }
func (a *GupyAdapter) Name() string { return "Gupy" }

func (a *GupyAdapter) DefaultCapabilities() Capabilities {
	return Capabilities{
		AccessType:            AccessAPI,
		SupportsKeywordSearch: true,
		SupportsPagination:    true,
		MaxResultsPerRun:      100,
		RateLimit:             RateLimitConfig{Requests: 1, Period: "second", Strategy: "token_bucket"},
	}
}

func (a *GupyAdapter) Fetch(ctx context.Context, cfg SourceConfig, keywords []string) ([]DiscoveredJob, error) {
	limit := effectiveLimit(cfg.Capabilities.MaxResultsPerRun, 100)
	query := strings.TrimSpace(strings.Join(keywords, " "))
	pageSize := 50
	if limit < pageSize {
		pageSize = limit
	}

	var jobs []DiscoveredJob
	seen := map[string]bool{}
	for offset := 0; len(jobs) < limit; offset += pageSize {
		rawURL := buildQuery(a.BaseURL, map[string]string{
			"jobName": query,
			"limit":   fmt.Sprint(pageSize),
			"offset":  fmt.Sprint(offset),
		})
		var payload any
		if err := apiGetJSON(ctx, a.HTTPClient, rawURL, cfg.RateLimiter, &payload); err != nil {
			return jobs, fmt.Errorf("gupy: %w", err)
		}
		items := jsonItems(payload)
		if len(items) == 0 {
			break
		}
		added := 0
		for _, item := range items {
			job := apiJobFromMap(item, a.SiteURL)
			if job.URL == "" {
				if id := jStr(item, "id"); id != "" {
					job.URL = strings.TrimRight(a.SiteURL, "/") + "/job/" + id
				}
			}
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
		if added == 0 || len(items) < pageSize {
			break
		}
	}
	return jobs, nil
}
