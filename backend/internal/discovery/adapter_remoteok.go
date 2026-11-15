package discovery

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type RemoteOKAdapter struct {
	BaseURL    string
	HTTPClient *http.Client
}

func NewRemoteOKAdapter() *RemoteOKAdapter {
	return &RemoteOKAdapter{
		BaseURL: "https://remoteok.com/api",
		HTTPClient: &http.Client{
			Timeout: 20 * time.Second,
		},
	}
}

func (a *RemoteOKAdapter) Slug() string { return "remoteok" }
func (a *RemoteOKAdapter) Name() string { return "Remote OK" }

func (a *RemoteOKAdapter) DefaultCapabilities() Capabilities {
	return Capabilities{
		SupportsKeywordSearch: false,
		SupportsPagination:    false,
		MaxResultsPerRun:      100,
		RateLimit: RateLimitConfig{
			Requests: 30,
			Period:   "minute",
		},
		AttributionRequired: true,
		Experimental:        false,
		RecentWindowHours:   24,
	}
}

type remoteOKItem struct {
	ID          string `json:"id"`
	URL         string `json:"url"`
	Position    string `json:"position"`
	Company     string `json:"company"`
	Description string `json:"description"`
}

func (a *RemoteOKAdapter) Fetch(ctx context.Context, cfg SourceConfig, keywords []string) ([]DiscoveredJob, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, a.BaseURL, nil)
	if err != nil {
		return nil, fmt.Errorf("remoteok: build request: %w", err)
	}
	// Remote OK requires a browser-like User-Agent
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) openJob/1.0 (https://github.com/maicondmenezes/openjob)")
	req.Header.Set("Accept", "application/json")

	client := a.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("remoteok: http request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("remoteok: unexpected status HTTP %d", resp.StatusCode)
	}

	var rawItems []remoteOKItem
	if err := json.NewDecoder(resp.Body).Decode(&rawItems); err != nil {
		return nil, fmt.Errorf("remoteok: decode response: %w", err)
	}

	maxResults := cfg.Capabilities.MaxResultsPerRun
	if maxResults <= 0 {
		maxResults = 100
	}

	jobs := make([]DiscoveredJob, 0, len(rawItems))
	for _, item := range rawItems {
		// First item or non-job items have empty Position/URL
		if strings.TrimSpace(item.Position) == "" || strings.TrimSpace(item.URL) == "" {
			continue
		}
		jobs = append(jobs, DiscoveredJob{
			Title:       strings.TrimSpace(item.Position),
			Company:     strings.TrimSpace(item.Company),
			URL:         strings.TrimSpace(item.URL),
			Description: CleanHTML(item.Description),
			Source:      a.Slug(),
			ExternalID:  item.ID,
		})
		if len(jobs) >= maxResults {
			break
		}
	}

	return jobs, nil
}
