package discovery

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type RemotiveAdapter struct {
	BaseURL    string
	HTTPClient *http.Client
}

func NewRemotiveAdapter() *RemotiveAdapter {
	return &RemotiveAdapter{
		BaseURL: "https://remotive.com/api/remote-jobs",
		HTTPClient: &http.Client{
			Timeout: 20 * time.Second,
		},
	}
}

func (a *RemotiveAdapter) Slug() string { return "remotive" }
func (a *RemotiveAdapter) Name() string { return "Remotive" }

func (a *RemotiveAdapter) DefaultCapabilities() Capabilities {
	return Capabilities{
		SupportsKeywordSearch: true,
		SupportsPagination:    false,
		MaxResultsPerRun:      50,
		RateLimit: RateLimitConfig{
			Requests: 4,
			Period:   "day",
		},
		AttributionRequired: true,
		Experimental:        false,
		RecentWindowHours:   24,
	}
}

type remotiveJobItem struct {
	ID          int64  `json:"id"`
	URL         string `json:"url"`
	Title       string `json:"title"`
	CompanyName string `json:"company_name"`
	Description string `json:"description"`
}

type remotiveResponse struct {
	Jobs []remotiveJobItem `json:"jobs"`
}

func (a *RemotiveAdapter) Fetch(ctx context.Context, cfg SourceConfig, keywords []string) ([]DiscoveredJob, error) {
	limit := cfg.Capabilities.MaxResultsPerRun
	if limit <= 0 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}

	reqURL := fmt.Sprintf("%s?limit=%d", a.BaseURL, limit)

	cleanedKws := make([]string, 0, len(keywords))
	for _, kw := range keywords {
		t := strings.TrimSpace(kw)
		if t != "" {
			cleanedKws = append(cleanedKws, t)
		}
	}
	if len(cleanedKws) > 0 {
		reqURL += "&search=" + url.QueryEscape(strings.Join(cleanedKws, " "))
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("remotive: build request: %w", err)
	}
	req.Header.Set("User-Agent", "openJob/1.0 (https://github.com/maicondmenezes/openjob)")

	client := a.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("remotive: http request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("remotive: unexpected status HTTP %d", resp.StatusCode)
	}

	var payload remotiveResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("remotive: decode response: %w", err)
	}

	jobs := make([]DiscoveredJob, 0, len(payload.Jobs))
	for _, j := range payload.Jobs {
		if j.URL == "" || j.Title == "" {
			continue
		}
		jobs = append(jobs, DiscoveredJob{
			Title:       strings.TrimSpace(j.Title),
			Company:     strings.TrimSpace(j.CompanyName),
			URL:         strings.TrimSpace(j.URL),
			Description: CleanHTML(j.Description),
			Source:      a.Slug(),
			ExternalID:  strconv.FormatInt(j.ID, 10),
		})
	}

	return jobs, nil
}
