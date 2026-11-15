package discovery

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type JobicyAdapter struct {
	BaseURL    string
	HTTPClient *http.Client
}

func NewJobicyAdapter() *JobicyAdapter {
	return &JobicyAdapter{
		BaseURL: "https://jobicy.com/api/v2/remote-jobs",
		HTTPClient: &http.Client{
			Timeout: 20 * time.Second,
		},
	}
}

func (a *JobicyAdapter) Slug() string { return "jobicy" }
func (a *JobicyAdapter) Name() string { return "Jobicy" }

func (a *JobicyAdapter) DefaultCapabilities() Capabilities {
	return Capabilities{
		SupportsKeywordSearch: false,
		SupportsPagination:    false,
		MaxResultsPerRun:      50,
		RateLimit: RateLimitConfig{
			Requests: 60,
			Period:   "minute",
		},
		AttributionRequired: true,
		Experimental:        false,
		RecentWindowHours:   168,
	}
}

type jobicyJobItem struct {
	ID             any    `json:"id"`
	URL            string `json:"url"`
	JobTitle       string `json:"jobTitle"`
	CompanyName    string `json:"companyName"`
	JobDescription string `json:"jobDescription"`
}

type jobicyResponse struct {
	Jobs []jobicyJobItem `json:"jobs"`
}

func (a *JobicyAdapter) Fetch(ctx context.Context, cfg SourceConfig, keywords []string) ([]DiscoveredJob, error) {
	limit := cfg.Capabilities.MaxResultsPerRun
	if limit <= 0 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}

	reqURL := fmt.Sprintf("%s?count=%d", a.BaseURL, limit)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("jobicy: build request: %w", err)
	}
	req.Header.Set("User-Agent", "openJob/1.0 (https://github.com/maicondmenezes/openjob)")

	client := a.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("jobicy: http request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("jobicy: unexpected status HTTP %d", resp.StatusCode)
	}

	var payload jobicyResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("jobicy: decode response: %w", err)
	}

	jobs := make([]DiscoveredJob, 0, len(payload.Jobs))
	for _, j := range payload.Jobs {
		if j.URL == "" || j.JobTitle == "" {
			continue
		}
		extID := ""
		if j.ID != nil {
			extID = fmt.Sprint(j.ID)
		}
		jobs = append(jobs, DiscoveredJob{
			Title:       strings.TrimSpace(j.JobTitle),
			Company:     strings.TrimSpace(j.CompanyName),
			URL:         strings.TrimSpace(j.URL),
			Description: CleanHTML(j.JobDescription),
			Source:      a.Slug(),
			ExternalID:  extID,
		})
	}

	return jobs, nil
}
