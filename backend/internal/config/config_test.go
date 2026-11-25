package config

import "testing"

func TestLoadDefaults(t *testing.T) {
	for _, key := range []string{
		"OPENJOB_PORT", "OPENJOB_DB_PATH", "OPENJOB_HOSTNAME",
		"OPENJOB_FEATURE_UI_FEED", "OPENJOB_FEATURE_EMAIL_SYNC",
		"OPENJOB_FEATURE_CURRICULUM_EXPORT", "OPENJOB_FEATURE_SCORE_DETERMINISTIC",
		"OPENJOB_FEATURE_KEYWORDS_DETERMINISTIC", "OPENJOB_FEATURE_FILTER_DETERMINISTIC",
	} {
		t.Setenv(key, "")
	}
	// t.Setenv with "" still sets an empty value, which pick treats as unset.
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
	want := Features{
		UIFeed: true, EmailSync: true, CurriculumExport: true,
		ScoreDeterministic: true, KeywordsDeterministic: true, FilterDeterministic: true,
	}
	if cfg.Features != want {
		t.Errorf("Features = %+v, want every release flag enabled by default", cfg.Features)
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

func TestLoadFeatures(t *testing.T) {
	t.Setenv("OPENJOB_FEATURE_UI_FEED", "true")
	t.Setenv("OPENJOB_FEATURE_EMAIL_SYNC", "1")
	t.Setenv("OPENJOB_FEATURE_CURRICULUM_EXPORT", "TRUE")
	t.Setenv("OPENJOB_FEATURE_SCORE_DETERMINISTIC", "not-a-bool")

	cfg := Load()
	if !cfg.Features.UIFeed || !cfg.Features.EmailSync || !cfg.Features.CurriculumExport {
		t.Errorf("Features = %+v, want UI feed, email sync and export enabled", cfg.Features)
	}
	if !cfg.Features.ScoreDeterministic {
		t.Errorf("ScoreDeterministic = false, want true (invalid value falls back to the safe default)")
	}
}

func TestReleaseFlagsCanBeDisabledForRollback(t *testing.T) {
	t.Setenv("OPENJOB_FEATURE_UI_FEED", "false")
	t.Setenv("OPENJOB_FEATURE_EMAIL_SYNC", "0")
	t.Setenv("OPENJOB_FEATURE_CURRICULUM_EXPORT", "0")

	cfg := Load()
	if cfg.Features.UIFeed || cfg.Features.EmailSync || cfg.Features.CurriculumExport {
		t.Errorf("Features = %+v, want release flags off for rollback", cfg.Features)
	}
}

func TestDeterministicFeatureGating(t *testing.T) {
	t.Setenv("OPENJOB_FEATURE_SCORE_DETERMINISTIC", "false")
	t.Setenv("OPENJOB_FEATURE_KEYWORDS_DETERMINISTIC", "true")
	t.Setenv("OPENJOB_FEATURE_FILTER_DETERMINISTIC", "true")

	cfg := Load()
	if cfg.Features.KeywordsEnabled() || cfg.Features.FilterEnabled() {
		t.Fatalf("specific flags must not bypass the master switch: %+v", cfg.Features)
	}

	t.Setenv("OPENJOB_FEATURE_SCORE_DETERMINISTIC", "true")
	cfg = Load()
	if !cfg.Features.KeywordsEnabled() || !cfg.Features.FilterEnabled() {
		t.Fatalf("flags should be enabled with the master switch: %+v", cfg.Features)
	}
}
