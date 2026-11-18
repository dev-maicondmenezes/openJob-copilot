package discovery

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// GetOnBoardAdapter consumes the public (zero-auth) Get on Board API.
// LATAM tech board with es/pt/en locales.
type GetOnBoardAdapter struct {
	BaseURL    string
	SiteURL    string
	HTTPClient *http.Client
}

func NewGetOnBoardAdapter() *GetOnBoardAdapter {
	return &GetOnBoardAdapter{
		BaseURL:    "https://www.getonbrd.com/api/v0",
		SiteURL:    "https://www.getonbrd.com",
		HTTPClient: &http.Client{Timeout: 20 * time.Second},
	}
}

func (a *GetOnBoardAdapter) Slug() string { return "getonboard" }
func (a *GetOnBoardAdapter) Name() string { return "Get on Board" }

func (a *GetOnBoardAdapter) DefaultCapabilities() Capabilities {
	return Capabilities{
		AccessType:            AccessAPI,
		SupportsKeywordSearch: true,
		SupportsPagination:    true,
		MaxResultsPerRun:      100,
		RateLimit:             RateLimitConfig{Requests: 1, Period: "second", Strategy: "token_bucket"},
	}
}

func (a *GetOnBoardAdapter) Fetch(ctx context.Context, cfg SourceConfig, keywords []string) ([]DiscoveredJob, error) {
	limit := effectiveLimit(cfg.Capabilities.MaxResultsPerRun, 100)
	query := strings.TrimSpace(strings.Join(keywords, " "))
	perPage := 50
	if limit < perPage {
		perPage = limit
	}

	var jobs []DiscoveredJob
	seen := map[string]bool{}
	for page := 1; len(jobs) < limit; page++ {
		rawURL := buildQuery(strings.TrimRight(a.BaseURL, "/")+"/search/jobs", map[string]string{
			"query":    query,
			"per_page": fmt.Sprint(perPage),
			"page":     fmt.Sprint(page),
		})
		if query == "" {
			rawURL = buildQuery(strings.TrimRight(a.BaseURL, "/")+"/jobs", map[string]string{
				"per_page": fmt.Sprint(perPage),
				"page":     fmt.Sprint(page),
			})
		}
		var payload any
		if err := apiGetJSON(ctx, a.HTTPClient, rawURL, cfg.RateLimiter, &payload); err != nil {
			return jobs, fmt.Errorf("getonboard: %w", err)
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
		if added == 0 || len(items) < perPage {
			break
		}
	}
	return jobs, nil
}

func effectiveLimit(configured, fallback int) int {
	if configured <= 0 {
		configured = fallback
	}
	if configured > fallback {
		configured = fallback
	}
	return configured
}
