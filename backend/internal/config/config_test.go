package config

import "testing"

func TestLoadDefaults(t *testing.T) {
	for _, key := range []string{"OPENJOB_PORT", "OPENJOB_DB_PATH", "OPENJOB_HOSTNAME"} {
		t.Setenv(key, "")
	}
	// t.Setenv with "" still sets an empty value, which envOr treats as unset.
	cfg := Load()
	if cfg.Port != "8080" {
		t.Errorf("Port = %q, want 8080", cfg.Port)
	}
	if cfg.DBPath != "/data/openjob.db" {
		t.Errorf("DBPath = %q, want /data/openjob.db", cfg.DBPath)
	}
	if cfg.Hostname != "openjob.copilot" {
		t.Errorf("Hostname = %q, want openjob.copilot", cfg.Hostname)
	}
}

func TestLoadOverrides(t *testing.T) {
	t.Setenv("OPENJOB_PORT", "9090")
	t.Setenv("OPENJOB_DB_PATH", "/tmp/test.db")
	t.Setenv("OPENJOB_HOSTNAME", "test.local")

	cfg := Load()
	if cfg.Port != "9090" || cfg.DBPath != "/tmp/test.db" || cfg.Hostname != "test.local" {
		t.Errorf("overrides not applied: %+v", cfg)
	}
}
