// Package config loads runtime configuration from OPENJOB_* environment
// variables. Defaults follow the naming conventions of the spec (§2).
package config

import (
	"os"
	"strconv"
)

// Features are the MVP release flags. They default to off so a merge never
// exposes unfinished behavior; enable them per environment as each phase ships.
type Features struct {
	// UIFeed enables the vertical feed (CP16).
	UIFeed bool
	// EmailSync enables email tracking and the in_process pipeline (CP17).
	EmailSync bool
	// CurriculumExport enables resume export (B10).
	CurriculumExport bool
	// ScoreDeterministic enables the deterministic scoring refactor (B11).
	ScoreDeterministic bool
}

// Config holds the runtime configuration of the openjob server.
type Config struct {
	// Port is the HTTP listen port (OPENJOB_PORT, default "8080").
	Port string
	// DBPath is the SQLite database location (OPENJOB_DB_PATH, default
	// "/data/openjob.db" inside the container volume).
	DBPath string
	// Hostname is the local DNS name announced via mDNS (OPENJOB_HOSTNAME,
	// default "openjob.copilot").
	Hostname string
	// Features are the release flags read from OPENJOB_FEATURE_* variables.
	Features Features
}

// Load reads configuration from the environment, applying defaults.
func Load() Config {
	return Config{
		Port:     envOr("OPENJOB_PORT", "8080"),
		DBPath:   envOr("OPENJOB_DB_PATH", "/data/openjob.db"),
		Hostname: envOr("OPENJOB_HOSTNAME", "openjob.copilot"),
		Features: Features{
			UIFeed:             envBool("OPENJOB_FEATURE_UI_FEED"),
			EmailSync:          envBool("OPENJOB_FEATURE_EMAIL_SYNC"),
			CurriculumExport:   envBool("OPENJOB_FEATURE_CURRICULUM_EXPORT"),
			ScoreDeterministic: envBool("OPENJOB_FEATURE_SCORE_DETERMINISTIC"),
		},
	}
}

func envBool(key string) bool {
	v, err := strconv.ParseBool(os.Getenv(key))
	return err == nil && v
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
