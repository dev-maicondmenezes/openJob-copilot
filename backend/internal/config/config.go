// Package config loads runtime configuration from OPENJOB_* environment
// variables. Defaults follow the naming conventions of the spec (§2).
package config

import "os"

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
}

// Load reads configuration from the environment, applying defaults.
func Load() Config {
	return Config{
		Port:     envOr("OPENJOB_PORT", "8080"),
		DBPath:   envOr("OPENJOB_DB_PATH", "/data/openjob.db"),
		Hostname: envOr("OPENJOB_HOSTNAME", "openjob.copilot"),
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
