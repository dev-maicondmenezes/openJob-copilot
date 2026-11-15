package discovery

import (
	"time"
)

// RateLimitConfig specifies the rate limit capability of a source.
type RateLimitConfig struct {
	Requests int    `json:"requests"`
	Period   string `json:"period"` // "day" | "hour" | "minute"
}

// Capabilities describes the declarative features of a vacancy source adapter.
type Capabilities struct {
	SupportsKeywordSearch bool            `json:"supports_keyword_search"`
	SupportsPagination    bool            `json:"supports_pagination"`
	MaxResultsPerRun      int             `json:"max_results_per_run"`
	RateLimit             RateLimitConfig `json:"rate_limit"`
	AttributionRequired   bool            `json:"attribution_required"`
	Experimental          bool            `json:"experimental"`
	RecentWindowHours     int             `json:"recent_window_hours"`
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
}

// DiscoveredJob is a job item returned by an adapter before filtration and persistence.
type DiscoveredJob struct {
	Title       string `json:"title"`
	Company     string `json:"company"`
	URL         string `json:"url"`
	Description string `json:"description"`
	Source      string `json:"source"`
	ExternalID  string `json:"external_id"`
}

// ResumeFilterSettings configures curriculum-based pre-filtering.
type ResumeFilterSettings struct {
	Enabled          bool   `json:"enabled"`
	MinMatches       int    `json:"min_matches"`       // default 3
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
	SourceID string `json:"source_id"`
	Adapter  string `json:"adapter"`
	Fetched  int    `json:"fetched"`
	Error    string `json:"error,omitempty"`
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
