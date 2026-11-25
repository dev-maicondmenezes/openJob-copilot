package bootstrap

import (
	"bytes"
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"github.com/maicondmenezes/openjob/internal/config"
)

func cfgIn(dir string) config.Config {
	return config.Config{Port: "8080", DBPath: filepath.Join(dir, "openjob.db"), Hostname: "openjob.copilot"}
}

func TestPrepareFirstRunCreatesPrivateDirAndKey(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "fresh", "openjob")
	res, err := Prepare(cfgIn(dir))
	if err != nil {
		t.Fatal(err)
	}
	if !res.KeyCreated || len(res.MasterKey) != 32 {
		t.Fatalf("first run: KeyCreated=%v keyLen=%d", res.KeyCreated, len(res.MasterKey))
	}
	if runtime.GOOS != "windows" {
		if fi, _ := os.Stat(dir); fi.Mode().Perm() != 0o700 {
			t.Errorf("dir mode %v, want 0700", fi.Mode().Perm())
		}
		if fi, _ := os.Stat(res.KeyPath); fi.Mode().Perm() != 0o600 {
			t.Errorf("key mode %v, want 0600", fi.Mode().Perm())
		}
	}
}

func TestPrepareIsIdempotentAndPreservesExistingData(t *testing.T) {
	dir := t.TempDir()
	cfg := cfgIn(dir)

	first, err := Prepare(cfg)
	if err != nil {
		t.Fatal(err)
	}
	// Pretend a DB already exists: Prepare must not touch it.
	if err := os.WriteFile(cfg.DBPath, []byte("sqlite-bytes"), 0o600); err != nil {
		t.Fatal(err)
	}

	second, err := Prepare(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if second.KeyCreated {
		t.Error("second run reported KeyCreated")
	}
	if !bytes.Equal(first.MasterKey, second.MasterKey) {
		t.Error("master key changed between runs")
	}
	if b, _ := os.ReadFile(cfg.DBPath); string(b) != "sqlite-bytes" {
		t.Errorf("existing DB was modified: %q", b)
	}

	// Leaves only db + key behind (no probe/temp files).
	entries, _ := os.ReadDir(dir)
	if len(entries) != 2 {
		names := []string{}
		for _, e := range entries {
			names = append(names, e.Name())
		}
		t.Errorf("unexpected leftovers: %v", names)
	}
}

func TestPrepareTightensLooseKeyMode(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX modes")
	}
	dir := t.TempDir()
	cfg := cfgIn(dir)
	res, _ := Prepare(cfg)
	if err := os.Chmod(res.KeyPath, 0o644); err != nil {
		t.Fatal(err)
	}
	res2, err := Prepare(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if fi, _ := os.Stat(res2.KeyPath); fi.Mode().Perm() != 0o600 {
		t.Errorf("mode not restored: %v", fi.Mode().Perm())
	}
	if len(res2.Warnings) != 1 {
		t.Errorf("expected one warning, got %v", res2.Warnings)
	}
}

func TestPrepareRefusesCorruptKeyWithoutTouchingIt(t *testing.T) {
	dir := t.TempDir()
	cfg := cfgIn(dir)
	keyPath := filepath.Join(dir, ".master.key")
	if err := os.WriteFile(keyPath, []byte("bad"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Prepare(cfg); err == nil {
		t.Fatal("expected error")
	}
	if b, _ := os.ReadFile(keyPath); string(b) != "bad" {
		t.Error("corrupt key was overwritten")
	}
}

func TestPrepareUnwritableDirIsActionable(t *testing.T) {
	if runtime.GOOS == "windows" || os.Geteuid() == 0 {
		t.Skip("needs a non-root POSIX user")
	}
	dir := filepath.Join(t.TempDir(), "ro")
	if err := os.Mkdir(dir, 0o500); err != nil {
		t.Fatal(err)
	}
	_, err := Prepare(cfgIn(dir))
	if err == nil {
		t.Fatal("expected error for read-only dir")
	}
	if !bytes.Contains([]byte(err.Error()), []byte("--db-path")) {
		t.Errorf("error lacks the actionable hint: %v", err)
	}
	// Retry after fixing permissions works: no partial state left behind.
	if err := os.Chmod(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if _, err := Prepare(cfgIn(dir)); err != nil {
		t.Errorf("retry after fix failed: %v", err)
	}
}

func TestListenExplicitPortIsStrict(t *testing.T) {
	busy, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatal(err)
	}
	defer busy.Close()

	if ln, err := Listen(PortOf(busy), true); err == nil {
		ln.Close()
		t.Fatal("explicit busy port must fail, not move to another port")
	}
}

func TestListenDefaultFallsForward(t *testing.T) {
	busy, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatal(err)
	}
	defer busy.Close()
	start, _ := strconv.Atoi(PortOf(busy))

	ln, err := Listen(strconv.Itoa(start), false)
	if err != nil {
		t.Skipf("no free neighbour port available: %v", err)
	}
	defer ln.Close()
	got, _ := strconv.Atoi(PortOf(ln))
	if got <= start || got >= start+fallbackSpan {
		t.Errorf("fallback port %d not in (%d, %d)", got, start, start+fallbackSpan)
	}
}

func TestWaitHealthyWaitsForOK(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if calls.Add(1) < 3 {
			http.Error(w, "starting", http.StatusServiceUnavailable)
			return
		}
		w.Write([]byte(`{"status":"ok"}`))
	}))
	defer srv.Close()

	if err := WaitHealthy(context.Background(), srv.URL, 5*time.Second); err != nil {
		t.Fatal(err)
	}
	if calls.Load() < 3 {
		t.Errorf("returned before the server was healthy (calls=%d)", calls.Load())
	}
}

func TestWaitHealthyTimesOut(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte(`{"status":"degraded"}`))
	}))
	defer srv.Close()
	if err := WaitHealthy(context.Background(), srv.URL, 300*time.Millisecond); err == nil {
		t.Fatal("expected timeout error")
	}
}

func TestCheckSingleProbe(t *testing.T) {
	ok := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.Write([]byte(`{"status":"ok"}`)) }))
	defer ok.Close()
	if err := Check(context.Background(), ok.URL, time.Second); err != nil {
		t.Errorf("healthy server: %v", err)
	}
	if err := Check(context.Background(), "http://127.0.0.1:1", 500*time.Millisecond); err == nil {
		t.Error("closed port must fail")
	}
}
