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

	// PortExplicit / DBPathExplicit record whether the value came from a flag
	// or an environment variable (as opposed to the built-in default). The
	// desktop binary uses them to pick safe defaults — a free port and a
	// per-user data directory — only when the person did not choose (CP11,
	// docs/adr/0012). The container never relies on them.
	PortExplicit   bool
	DBPathExplicit bool
}

// Overrides carries command-line flag values. Empty means "flag not given".
// Precedence, highest first: flag > environment variable > built-in default
// (documented in docs/distribution.md).
type Overrides struct {
	Port     string
	DBPath   string
	Hostname string
}

// Load reads configuration from the environment, applying defaults.
func Load() Config { return LoadWith(Overrides{}) }

// LoadWith resolves the configuration with flag > env > default precedence.
func LoadWith(o Overrides) Config {
	port, portSet := pick(o.Port, "OPENJOB_PORT", "8080")
	dbPath, dbSet := pick(o.DBPath, "OPENJOB_DB_PATH", "/data/openjob.db")
	hostname, _ := pick(o.Hostname, "OPENJOB_HOSTNAME", "openjob.copilot")
	return Config{
		Port:           port,
		PortExplicit:   portSet,
		DBPath:         dbPath,
		DBPathExplicit: dbSet,
		Hostname:       hostname,
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

// pick returns the value and whether it was set explicitly (flag or env).
func pick(flag, key, fallback string) (string, bool) {
	if flag != "" {
		return flag, true
	}
	if v := os.Getenv(key); v != "" {
		return v, true
	}
	return fallback, false
}
