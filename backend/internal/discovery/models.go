package discovery

import (
	"context"
	"fmt"
	"time"
)

// Access types of a source (CP14). API sources are preferred; scraping
// sources are first-class citizens but always guarded by a defensive rate
// limiter, daily quota and robots.txt constraints.
const (
	AccessAPI      = "api"
	AccessScraping = "scraping"
)

// RateLimitConfig specifies the per-request pace capability of a source.
// Period accepts "second", "minute", "hour" and "day" (CP14 adds "second"
// for scraping sources, e.g. 1 request every 2s).
type RateLimitConfig struct {
	Requests int    `json:"requests"`
	Period   string `json:"period"`
	Strategy string `json:"strategy,omitempty"` // "token_bucket" (default)
}

// Capabilities describes the declarative features of a vacancy source adapter.
type Capabilities struct {
	AccessType            string          `json:"access_type"` // "api" | "scraping"
	SupportsKeywordSearch bool            `json:"supports_keyword_search"`
	SupportsPagination    bool            `json:"supports_pagination"`
	MaxResultsPerRun      int             `json:"max_results_per_run"`
	RateLimit             RateLimitConfig `json:"rate_limit"`
	AttributionRequired   bool            `json:"attribution_required"`
	Experimental          bool            `json:"experimental"`
	RecentWindowHours     int             `json:"recent_window_hours"`

	// CP14 — defensive scraping controls.
	RequiresDailyQuota    bool     `json:"requires_daily_quota"`
	DailyQuota            int      `json:"daily_quota"`
	QuotaUnit             string   `json:"quota_unit,omitempty"` // "requests" (default) | "details"
	MaxDetailsPerRun      int      `json:"max_details_per_run"`
	RobotsDisallowedPaths []string `json:"robots_disallowed_paths,omitempty"`
}

// IsScraping reports whether the source relies on HTML scraping.
func (c Capabilities) IsScraping() bool { return c.AccessType == AccessScraping }

// RequestWaiter serializes requests of a source according to its token
// bucket. Implemented by the persistent rate limiter (CP14).
type RequestWaiter interface {
	Wait(ctx context.Context) error
}

// UsageTracker is shared between the orchestrator and an adapter: adapters
// increment DetailRequests for every detail page actually fetched, so the
// orchestrator can persist the daily quota consumption after the run.
type UsageTracker struct {
	Requests       int // every listing/detail request made by the adapter
	DetailRequests int // detail-page requests (Vagas.com.br quota)
}

// SourceConfig wraps database source record with decrypted credentials.
type SourceConfig struct {
	ID           string       `json:"id"`
	Name         string       `json:"name"`
	Adapter      string       `json:"adapter"`
	Enabled      bool         `json:"enabled"`
	Priority     int          `json:"priority"`
	AuthType     string       `json:"auth_type"` // "none" | "api_key"
	APIKey       string       `json:"-"`         // decrypted in memory, never serialized
	Capabilities Capabilities `json:"capabilities"`

	// CP14 — orchestration-only fields (never serialized to the API).
	RateLimiter    RequestWaiter `json:"-"`
	DailyRemaining int           `json:"-"`
	Usage          *UsageTracker `json:"-"`
}

// BlockedError signals that a source rejected a request with 429/403 (or an
// equivalent block signal). The orchestrator reacts by scheduling an
// exponential backoff instead of retrying immediately.
type BlockedError struct {
	StatusCode int
	RetryAfter time.Duration
}

func (e *BlockedError) Error() string {
	if e.RetryAfter > 0 {
		return fmt.Sprintf("fonte bloqueou a requisição (HTTP %d); nova tentativa em %s", e.StatusCode, e.RetryAfter)
	}
	return fmt.Sprintf("fonte bloqueou a requisição (HTTP %d)", e.StatusCode)
}

// DiscoveredJob is a job item returned by an adapter before filtration and persistence.
type DiscoveredJob struct {
	Title       string `json:"title"`
	Company     string `json:"company"`
	URL         string `json:"url"`
	Description string `json:"description"`
	Source      string `json:"source"`
	ExternalID  string `json:"external_id"`

	// CP14 listing metadata (kept in the description when persisted, since the
	// jobs schema is intentionally stable).
	Location string `json:"location,omitempty"`
	Modality string `json:"modality,omitempty"`
	Salary   string `json:"salary,omitempty"`
}

// ResumeFilterSettings configures curriculum-based pre-filtering.
type ResumeFilterSettings struct {
	Enabled          bool   `json:"enabled"`
	MinMatches       int    `json:"min_matches"`        // default 3
	MatchMode        string `json:"match_mode"`         // "any" | "all"
	SourceOfKeywords string `json:"source_of_keywords"` // "resume" | "manual"
}

// Config represents global discovery settings stored in the settings table.
type Config struct {
	Keywords          []string             `json:"keywords"` // max 5
	ResumeFilter      ResumeFilterSettings `json:"resume_filter"`
	DiscoveryInterval string               `json:"discovery_interval"` // "off" | "1h" | "6h" | "24h"
}

// DefaultConfig provides sensible defaults (ticker disabled by default).
func DefaultConfig() Config {
	return Config{
		Keywords: []string{},
		ResumeFilter: ResumeFilterSettings{
			Enabled:          true,
			MinMatches:       3,
			MatchMode:        "any",
			SourceOfKeywords: "resume",
		},
		DiscoveryInterval: "off",
	}
}

// SourceRunStat tracks per-source results during a discovery run.
type SourceRunStat struct {
	SourceID     string `json:"source_id"`
	Adapter      string `json:"adapter"`
	AccessType   string `json:"access_type,omitempty"`
	Fetched      int    `json:"fetched"`
	DetailsUsed  int    `json:"details_used,omitempty"`
	DailyUsed    int    `json:"daily_used,omitempty"`
	DailyQuota   int    `json:"daily_quota,omitempty"`
	Skipped      bool   `json:"skipped,omitempty"`
	Status       string `json:"status,omitempty"` // "ok" | "degraded" | "blocked" | "quota"
	BackoffUntil string `json:"backoff_until,omitempty"`
	Note         string `json:"note,omitempty"`
	Error        string `json:"error,omitempty"`
}

// RunStats holds aggregate and per-stage counters of a discovery run.
type RunStats struct {
	Fetched          int             `json:"fetched"`
	FilteredKeywords int             `json:"filtered_keywords"`
	FilteredResume   int             `json:"filtered_resume"`
	NewJobs          int             `json:"new_jobs"`
	Duplicates       int             `json:"duplicates"`
	Scored           int             `json:"scored"`
	Errors           []string        `json:"errors,omitempty"`
	Sources          []SourceRunStat `json:"sources"`
	CompletedAt      time.Time       `json:"completed_at,omitempty"`
}
