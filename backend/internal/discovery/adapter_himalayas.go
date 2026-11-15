package discovery

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type HimalayasAdapter struct {
	BaseURL    string
	HTTPClient *http.Client
}

func NewHimalayasAdapter() *HimalayasAdapter {
	return &HimalayasAdapter{
		BaseURL: "https://himalayas.app/jobs/api",
		HTTPClient: &http.Client{
			Timeout: 20 * time.Second,
		},
	}
}

func (a *HimalayasAdapter) Slug() string { return "himalayas" }
func (a *HimalayasAdapter) Name() string { return "Himalayas" }

func (a *HimalayasAdapter) DefaultCapabilities() Capabilities {
	return Capabilities{
		SupportsKeywordSearch: true,
		SupportsPagination:    true,
		MaxResultsPerRun:      50,
		RateLimit: RateLimitConfig{
			Requests: 60,
			Period:   "minute",
		},
		AttributionRequired: false,
		Experimental:        false,
		RecentWindowHours:   0,
	}
}

type himalayasJobItem struct {
	Title           string `json:"title"`
	CompanyName     string `json:"companyName"`
	ApplicationLink string `json:"applicationLink"`
	GUID            string `json:"guid"`
	Excerpt         string `json:"excerpt"`
	Description     string `json:"description"`
}

type himalayasResponse struct {
	Jobs []himalayasJobItem `json:"jobs"`
}

func (a *HimalayasAdapter) Fetch(ctx context.Context, cfg SourceConfig, keywords []string) ([]DiscoveredJob, error) {
	limit := cfg.Capabilities.MaxResultsPerRun
	if limit <= 0 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}

	endpoint := a.BaseURL
	// If searching by keywords
	cleanedKws := make([]string, 0, len(keywords))
	for _, kw := range keywords {
		t := strings.TrimSpace(kw)
		if t != "" {
			cleanedKws = append(cleanedKws, t)
		}
	}

	var reqURL string
	if len(cleanedKws) > 0 {
		// Use search endpoint if BaseURL is default, or query param
		if strings.HasSuffix(endpoint, "/api") {
			reqURL = endpoint + "/search?keyword=" + url.QueryEscape(strings.Join(cleanedKws, " ")) + fmt.Sprintf("&limit=%d", limit)
		} else {
			reqURL = endpoint + "?keyword=" + url.QueryEscape(strings.Join(cleanedKws, " ")) + fmt.Sprintf("&limit=%d", limit)
		}
	} else {
		reqURL = endpoint + fmt.Sprintf("?limit=%d", limit)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("himalayas: build request: %w", err)
	}
	req.Header.Set("User-Agent", "openJob/1.0 (https://github.com/maicondmenezes/openjob)")
	if cfg.APIKey != "" {
		req.Header.Set("X-Himalayas-Key", cfg.APIKey)
	}

	client := a.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("himalayas: http request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("himalayas: unexpected status HTTP %d", resp.StatusCode)
	}

	var payload himalayasResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("himalayas: decode response: %w", err)
	}

	jobs := make([]DiscoveredJob, 0, len(payload.Jobs))
	for _, j := range payload.Jobs {
		jobURL := j.ApplicationLink
		if jobURL == "" {
			jobURL = j.GUID
		}
		if jobURL == "" || j.Title == "" {
			continue
		}

		desc := CleanHTML(j.Description)
		if desc == "" {
			desc = j.Excerpt
		}

		jobs = append(jobs, DiscoveredJob{
			Title:       strings.TrimSpace(j.Title),
			Company:     strings.TrimSpace(j.CompanyName),
			URL:         strings.TrimSpace(jobURL),
			Description: desc,
			Source:      a.Slug(),
			ExternalID:  j.GUID,
		})
	}

	return jobs, nil
}
