package discovery

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/html"
)

// programathorCacheTTL is the CP14 listing cache window: repeated discovery
// runs within six hours reuse the parsed listing instead of hitting the site
// again, which keeps the effective request rate well below the 1 req/2s floor.
const programathorCacheTTL = 6 * time.Hour

type programathorCacheEntry struct {
	doc      *html.Node
	finalURL string
	expires  time.Time
}

// ProgramathorAdapter scrapes the public Programathor listing (pt-BR, Brazil).
// Conservative pace: 1 request every 2s, plus a 6h listing cache.
type ProgramathorAdapter struct {
	BaseURL    string
	SiteURL    string
	HTTPClient *http.Client

	mu    sync.Mutex
	cache map[string]programathorCacheEntry
}

func NewProgramathorAdapter() *ProgramathorAdapter {
	return &ProgramathorAdapter{
		BaseURL:    "https://programathor.com.br",
		SiteURL:    "https://programathor.com.br",
		HTTPClient: &http.Client{Timeout: 25 * time.Second},
		cache:      map[string]programathorCacheEntry{},
	}
}

func (a *ProgramathorAdapter) Slug() string { return "programathor" }
func (a *ProgramathorAdapter) Name() string { return "Programathor" }

func (a *ProgramathorAdapter) DefaultCapabilities() Capabilities {
	return Capabilities{
		AccessType:            AccessScraping,
		SupportsKeywordSearch: true,
		SupportsPagination:    true,
		MaxResultsPerRun:      200,
		RateLimit:             RateLimitConfig{Requests: 1, Period: "2s", Strategy: "token_bucket"},
	}
}

// cachedPage returns a cached listing page or fetches and stores it. Cache hits
// skip both the network and the rate limiter, since no request is made; the
// bool reports whether the page came from the cache.
func (a *ProgramathorAdapter) cachedPage(ctx context.Context, rawURL string, wait RequestWaiter) (*html.Node, string, bool, error) {
	now := time.Now()
	a.mu.Lock()
	if entry, ok := a.cache[rawURL]; ok && now.Before(entry.expires) {
		a.mu.Unlock()
		return entry.doc, entry.finalURL, true, nil
	}
	a.mu.Unlock()

	doc, finalURL, err := fetchPage(ctx, a.HTTPClient, rawURL, wait)
	if err != nil {
		return nil, finalURL, false, err
	}

	a.mu.Lock()
	if a.cache == nil {
		a.cache = map[string]programathorCacheEntry{}
	}
	a.cache[rawURL] = programathorCacheEntry{doc: doc, finalURL: finalURL, expires: now.Add(programathorCacheTTL)}
	a.mu.Unlock()
	return doc, finalURL, false, nil
}

func (a *ProgramathorAdapter) Fetch(ctx context.Context, cfg SourceConfig, keywords []string) ([]DiscoveredJob, error) {
	limit := effectiveLimit(cfg.Capabilities.MaxResultsPerRun, 200)
	query := slugifyKeyword(strings.Join(keywords, " "))
	spec := cardSpec{
		CardClasses:     []string{"job", "job-card", "card-job", "job-item"},
		TitleClasses:    []string{"job-title", "title", "job-name"},
		CompanyClasses:  []string{"job-company", "company", "job-company-name"},
		LocationClasses: []string{"job-location", "location", "job-local"},
		LinkClasses:     []string{"job-title", "title"},
	}

	var jobs []DiscoveredJob
	seen := map[string]bool{}
	for page := 1; len(jobs) < limit; page++ {
		rawURL := buildQuery(strings.TrimRight(a.BaseURL, "/")+"/jobs", map[string]string{
			"search": query,
			"page":   fmt.Sprint(page),
		})
		doc, finalURL, cached, err := a.cachedPage(ctx, rawURL, cfg.RateLimiter)
		if err != nil {
			return jobs, fmt.Errorf("programathor: %w", err)
		}
		if cfg.Usage != nil && !cached {
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
