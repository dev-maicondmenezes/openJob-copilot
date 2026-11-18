package discovery

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// BumeranAdapter uses the internal searchV2 endpoint (POST) that powers the
// public Bumeran listing. robots.txt only disallows the recientes= and
// relevantes= query modes; the JSON search body below does not use them.
type BumeranAdapter struct {
	BaseURL    string // site base, e.g. https://www.bumeran.com.ar
	APIURL     string // internal search endpoint
	SiteURL    string
	HTTPClient *http.Client
}

func NewBumeranAdapter() *BumeranAdapter {
	return &BumeranAdapter{
		BaseURL:    "https://www.bumeran.com.ar",
		APIURL:     "https://www.bumeran.com.ar/api/avisos/searchV2",
		SiteURL:    "https://www.bumeran.com.ar",
		HTTPClient: &http.Client{Timeout: 25 * time.Second},
	}
}

func (a *BumeranAdapter) Slug() string { return "bumeran" }
func (a *BumeranAdapter) Name() string { return "Bumeran" }

func (a *BumeranAdapter) DefaultCapabilities() Capabilities {
	return Capabilities{
		AccessType:            AccessScraping,
		SupportsKeywordSearch: true,
		SupportsPagination:    true,
		MaxResultsPerRun:      1200,
		RateLimit:             RateLimitConfig{Requests: 1, Period: "second", Strategy: "token_bucket"},
		RobotsDisallowedPaths: []string{"recientes=", "relevantes="},
	}
}

func (a *BumeranAdapter) Fetch(ctx context.Context, cfg SourceConfig, keywords []string) ([]DiscoveredJob, error) {
	limit := effectiveLimit(cfg.Capabilities.MaxResultsPerRun, 1200)
	query := strings.TrimSpace(strings.Join(keywords, " "))
	pageSize := 50

	var jobs []DiscoveredJob
	seen := map[string]bool{}
	for page := 1; len(jobs) < limit; page++ {
		payload := map[string]any{
			"filtro": map[string]any{
				"keyword": query,
			},
			"pagina":        page,
			"tamanioPagina": pageSize,
		}
		var response any
		if err := apiPostJSON(ctx, a.HTTPClient, a.APIURL, cfg.RateLimiter, payload, &response); err != nil {
			return jobs, fmt.Errorf("bumeran: %w", err)
		}
		if cfg.Usage != nil {
			cfg.Usage.Requests++
		}
		items := jsonItems(response)
		if len(items) == 0 {
			break
		}
		added := 0
		for _, item := range items {
			job := apiJobFromMap(item, a.SiteURL)
			if job.URL == "" {
				if slug := jStr(item, "slug", "id", "avisoId"); slug != "" {
					job.URL = strings.TrimRight(a.SiteURL, "/") + "/empleos/" + slug
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
