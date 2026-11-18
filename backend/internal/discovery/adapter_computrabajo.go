package discovery

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// ComputrabajoAdapter scrapes server-rendered listing cards (es-419, 19 LATAM
// countries). robots.txt disallows the filter parameters by=, pubdate= and
// sal=; the adapter never sends them — keyword filtering happens after fetch.
type ComputrabajoAdapter struct {
	BaseURL    string
	SiteURL    string
	HTTPClient *http.Client
}

func NewComputrabajoAdapter() *ComputrabajoAdapter {
	return &ComputrabajoAdapter{
		BaseURL:    "https://www.computrabajo.com.br",
		SiteURL:    "https://www.computrabajo.com.br",
		HTTPClient: &http.Client{Timeout: 25 * time.Second},
	}
}

func (a *ComputrabajoAdapter) Slug() string { return "computrabajo" }
func (a *ComputrabajoAdapter) Name() string { return "Computrabajo" }

func (a *ComputrabajoAdapter) DefaultCapabilities() Capabilities {
	return Capabilities{
		AccessType:            AccessScraping,
		SupportsKeywordSearch: true,
		SupportsPagination:    true,
		MaxResultsPerRun:      500,
		RateLimit:             RateLimitConfig{Requests: 1, Period: "second", Strategy: "token_bucket"},
		RobotsDisallowedPaths: []string{"by=", "pubdate=", "sal="},
	}
}

func (a *ComputrabajoAdapter) Fetch(ctx context.Context, cfg SourceConfig, keywords []string) ([]DiscoveredJob, error) {
	limit := effectiveLimit(cfg.Capabilities.MaxResultsPerRun, 500)
	query := slugifyKeyword(strings.Join(keywords, " "))
	path := "/ofertas-de-trabajo"
	if query != "" {
		path = "/trabajo-de-" + query
	}
	spec := cardSpec{
		CardClasses:     []string{"box_offer", "offer", "ij-OfferCard", "b_BoxCard"},
		TitleClasses:    []string{"js-o-link", "offer-title", "title", "fwB"},
		CompanyClasses:  []string{"company", "fs16", "offer-company", "fc_base"},
		LocationClasses: []string{"location", "fs13", "offer-location", "fc_base"},
		SalaryClasses:   []string{"salary", "salario"},
		LinkClasses:     []string{"js-o-link", "offer-title", "title"},
	}

	var jobs []DiscoveredJob
	seen := map[string]bool{}
	for page := 1; len(jobs) < limit; page++ {
		// Only the pagination parameter p= is allowed by robots.txt.
		rawURL := buildQuery(strings.TrimRight(a.BaseURL, "/")+path, map[string]string{
			"p": fmt.Sprint(page),
		})
		doc, finalURL, err := fetchPage(ctx, a.HTTPClient, rawURL, cfg.RateLimiter)
		if err != nil {
			return jobs, fmt.Errorf("computrabajo: %w", err)
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
