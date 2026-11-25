package config

import (
	"path/filepath"
	"strings"
	"testing"
)

func clearEnv(t *testing.T) {
	t.Helper()
	for _, k := range []string{"OPENJOB_PORT", "OPENJOB_DB_PATH", "OPENJOB_HOSTNAME"} {
		t.Setenv(k, "")
	}
}

func TestPrecedenceFlagOverEnvOverDefault(t *testing.T) {
	clearEnv(t)

	def := LoadWith(Overrides{})
	if def.Port != "8080" || def.PortExplicit || def.DBPathExplicit {
		t.Fatalf("defaults wrong or marked explicit: %+v", def)
	}

	t.Setenv("OPENJOB_PORT", "9000")
	t.Setenv("OPENJOB_DB_PATH", "/env/openjob.db")
	env := LoadWith(Overrides{})
	if env.Port != "9000" || env.DBPath != "/env/openjob.db" || !env.PortExplicit || !env.DBPathExplicit {
		t.Fatalf("env should beat default: %+v", env)
	}

	flag := LoadWith(Overrides{Port: "9100", DBPath: "/flag/openjob.db", Hostname: "flag.test"})
	if flag.Port != "9100" || flag.DBPath != "/flag/openjob.db" || flag.Hostname != "flag.test" {
		t.Fatalf("flag should beat env: %+v", flag)
	}
}

func TestValidate(t *testing.T) {
	ok := Config{Port: "8080", DBPath: "/data/openjob.db", Hostname: "openjob.copilot"}
	if err := ok.Validate(); err != nil {
		t.Fatalf("valid config rejected: %v", err)
	}

	cases := map[string]Config{
		"port not a number": {Port: "abc", DBPath: "/d/x.db", Hostname: "h"},
		"port zero":         {Port: "0", DBPath: "/d/x.db", Hostname: "h"},
		"port too large":    {Port: "70000", DBPath: "/d/x.db", Hostname: "h"},
		"empty db path":     {Port: "8080", DBPath: "  ", Hostname: "h"},
		"hostname scheme":   {Port: "8080", DBPath: "/d/x.db", Hostname: "http://x"},
		"hostname space":    {Port: "8080", DBPath: "/d/x.db", Hostname: "a b"},
	}
	for name, c := range cases {
		if err := c.Validate(); err == nil {
			t.Errorf("%s: expected validation error", name)
		}
	}

	// All problems are reported together, not just the first.
	err := (Config{Port: "x", DBPath: "", Hostname: ""}).Validate()
	if err == nil || strings.Count(err.Error(), "\n") < 2 {
		t.Errorf("expected 3 joined errors, got %v", err)
	}
}

func TestDefaultDataDir(t *testing.T) {
	env := func(m map[string]string) func(string) string {
		return func(k string) string { return m[k] }
	}
	home := filepath.FromSlash("/home/u")

	cases := []struct {
		name string
		goos string
		env  map[string]string
		want string
	}{
		{"linux default", "linux", nil, filepath.Join(home, ".local", "share", "openjob")},
		{"linux xdg", "linux", map[string]string{"XDG_DATA_HOME": "/xdg"}, filepath.Join("/xdg", "openjob")},
		{"darwin", "darwin", nil, filepath.Join(home, "Library", "Application Support", "openjob")},
		{"windows appdata", "windows", map[string]string{"AppData": `C:\Users\u\AppData\Roaming`}, filepath.Join(`C:\Users\u\AppData\Roaming`, "openjob")},
		{"windows fallback", "windows", nil, filepath.Join(home, "AppData", "Roaming", "openjob")},
	}
	for _, c := range cases {
		if got := DefaultDataDir(c.goos, env(c.env), home); got != c.want {
			t.Errorf("%s: got %q want %q", c.name, got, c.want)
		}
	}
}

func TestApplyDesktopDefaults(t *testing.T) {
	clearEnv(t)
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", "")

	c := Load()
	c.ApplyDesktopDefaults()
	if c.DBPath == "/data/openjob.db" || filepath.Base(c.DBPath) != "openjob.db" {
		t.Errorf("desktop default not applied: %q", c.DBPath)
	}

	// An explicit path is never replaced (no silent data migration).
	t.Setenv("OPENJOB_DB_PATH", "/custom/openjob.db")
	c = Load()
	c.ApplyDesktopDefaults()
	if c.DBPath != "/custom/openjob.db" {
		t.Errorf("explicit DBPath overridden: %q", c.DBPath)
	}
	if got := c.DataDir(); got != filepath.Dir("/custom/openjob.db") {
		t.Errorf("DataDir = %q", got)
	}
}
