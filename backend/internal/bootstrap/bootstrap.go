// Package bootstrap is the minimal technical preparation the openjob binary
// (and the .deb) performs before serving: data directory, master key, port.
// It deliberately configures NOTHING about the product — providers, resumes,
// Telegram and the like belong to the web onboarding (CP18/CP19). See
// docs/adr/0012-bootstrap-minimo-binario.md.
package bootstrap

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"runtime"

	"github.com/maicondmenezes/openjob/internal/config"
	"github.com/maicondmenezes/openjob/internal/store"
)

// Result describes what Prepare found or created.
type Result struct {
	DataDir string
	DBPath  string
	KeyPath string
	// MasterKey is the 32-byte AES-GCM key (never log it).
	MasterKey []byte
	// KeyCreated is true only on the very first run.
	KeyCreated bool
	// Warnings are non-fatal findings (e.g. a loose key mode that was fixed).
	Warnings []string
}

// Prepare makes the data directory and the master key exist with restrictive
// permissions. It is idempotent and never overwrites an existing key or DB.
// On failure it leaves no partial state that would block a retry: the key is
// published atomically (store.LoadOrCreateMasterKey) and the writability probe
// file is always removed.
func Prepare(cfg config.Config) (Result, error) {
	res := Result{
		DataDir: cfg.DataDir(),
		DBPath:  cfg.DBPath,
		KeyPath: store.MasterKeyPath(cfg.DBPath),
	}

	// 0700 applies only to directories created here; a pre-existing directory
	// (e.g. a mounted volume) keeps its owner-chosen mode.
	if err := os.MkdirAll(res.DataDir, 0o700); err != nil {
		return res, hint(fmt.Errorf("creating data directory %s: %w", res.DataDir, err), cfg.DBPath)
	}
	if err := probeWritable(res.DataDir); err != nil {
		return res, hint(fmt.Errorf("data directory %s is not writable: %w", res.DataDir, err), cfg.DBPath)
	}

	_, statErr := os.Stat(res.KeyPath)
	res.KeyCreated = errors.Is(statErr, fs.ErrNotExist)

	key, err := store.LoadOrCreateMasterKey(res.KeyPath)
	if err != nil {
		return res, fmt.Errorf("master key %s: %w", res.KeyPath, err)
	}
	res.MasterKey = key

	if w := tightenKeyMode(res.KeyPath); w != "" {
		res.Warnings = append(res.Warnings, w)
	}
	return res, nil
}

// tightenKeyMode fixes a master key readable by group/others (e.g. restored
// from a backup with the wrong umask). Returns a warning when it changed or
// could not change the mode. No-op on Windows, where POSIX modes don't apply.
func tightenKeyMode(path string) string {
	if runtime.GOOS == "windows" {
		return ""
	}
	fi, err := os.Stat(path)
	if err != nil || fi.Mode().Perm()&0o077 == 0 {
		return ""
	}
	if err := os.Chmod(path, 0o600); err != nil {
		return fmt.Sprintf("master key %s is accessible by other users (mode %o) and could not be restricted: %v", path, fi.Mode().Perm(), err)
	}
	return fmt.Sprintf("master key %s had mode %o; restricted to 600", path, fi.Mode().Perm())
}

func probeWritable(dir string) error {
	f, err := os.CreateTemp(dir, ".openjob-write-test-*")
	if err != nil {
		return err
	}
	name := f.Name()
	f.Close()
	return os.Remove(name)
}

// hint appends an actionable next step to permission/path failures.
func hint(err error, dbPath string) error {
	if errors.Is(err, fs.ErrPermission) || errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("%w\n  → choose another location with --db-path or OPENJOB_DB_PATH (current: %s), or fix the directory permissions",
			err, dbPath)
	}
	return err
}
