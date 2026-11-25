package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

// Validate checks the process-level settings so a bad OPENJOB_* value fails
// fast with an actionable message instead of a cryptic listen/open error
// (CP11 §5.3: "documentar nome, default, validação e precedência").
func (c Config) Validate() error {
	var errs []error

	p, err := strconv.Atoi(c.Port)
	if err != nil || p < 1 || p > 65535 {
		errs = append(errs, fmt.Errorf("invalid port %q (OPENJOB_PORT / --port): use a number between 1 and 65535", c.Port))
	}
	if strings.TrimSpace(c.DBPath) == "" {
		errs = append(errs, errors.New("empty database path (OPENJOB_DB_PATH / --db-path)"))
	}
	if h := c.Hostname; h == "" || len(h) > 253 || strings.ContainsAny(h, " /\\:@") {
		errs = append(errs, fmt.Errorf("invalid hostname %q (OPENJOB_HOSTNAME / --hostname): use a bare DNS name like openjob.copilot", h))
	}
	return errors.Join(errs...)
}

// DataDir is the directory that holds the database and the master key. The
// master key always lives next to the DB (docs/adr/0004).
func (c Config) DataDir() string { return filepath.Dir(c.DBPath) }

// ApplyDesktopDefaults gives the desktop binary (and the .deb) a per-user data
// directory when the person did not set OPENJOB_DB_PATH / --db-path. The
// container never calls this: its image sets OPENJOB_DB_PATH=/data/openjob.db
// explicitly, and /data is not writable for a regular user anyway.
func (c *Config) ApplyDesktopDefaults() {
	if c.DBPathExplicit {
		return
	}
	home, _ := os.UserHomeDir()
	c.DBPath = filepath.Join(DefaultDataDir(runtime.GOOS, os.Getenv, home), "openjob.db")
}

// DefaultDataDir returns the per-user data directory for the given OS. It is a
// pure function (OS, env lookup and home are injected) so every platform is
// testable on one machine. Layout:
//
//	linux   $XDG_DATA_HOME/openjob  or  ~/.local/share/openjob
//	darwin  ~/Library/Application Support/openjob
//	windows %AppData%\openjob       or  ~\AppData\Roaming\openjob
func DefaultDataDir(goos string, getenv func(string) string, home string) string {
	switch goos {
	case "windows":
		if appData := getenv("AppData"); appData != "" {
			return filepath.Join(appData, "openjob")
		}
		return filepath.Join(home, "AppData", "Roaming", "openjob")
	case "darwin":
		return filepath.Join(home, "Library", "Application Support", "openjob")
	default:
		if xdg := getenv("XDG_DATA_HOME"); xdg != "" {
			return filepath.Join(xdg, "openjob")
		}
		return filepath.Join(home, ".local", "share", "openjob")
	}
}
