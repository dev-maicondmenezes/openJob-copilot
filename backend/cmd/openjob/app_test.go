package main

import (
	"bytes"
	"context"
	"io"
	"net"
	"net/http"
	"net/netip"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/maicondmenezes/openjob/internal/netinfo"
)

// syncBuf is a goroutine-safe buffer: the slog handler writes from server
// goroutines while the test reads.
type syncBuf struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (s *syncBuf) Write(p []byte) (int, error) { s.mu.Lock(); defer s.mu.Unlock(); return s.b.Write(p) }
func (s *syncBuf) String() string              { s.mu.Lock(); defer s.mu.Unlock(); return s.b.String() }

func freePort(t *testing.T) string {
	t.Helper()
	ln, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	return strconv.Itoa(ln.Addr().(*net.TCPAddr).Port)
}

// testEnv isolates every OPENJOB_* variable and the per-user data dirs.
func testEnv(t *testing.T) string {
	t.Helper()
	for _, k := range []string{"OPENJOB_PORT", "OPENJOB_DB_PATH", "OPENJOB_HOSTNAME", "OPENJOB_CONTAINER"} {
		t.Setenv(k, "")
	}
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_DATA_HOME", filepath.Join(home, "xdg"))
	t.Setenv("AppData", filepath.Join(home, "appdata"))
	return home
}

type running struct {
	out  *syncBuf
	errb *syncBuf
	stop func() int
}

// start runs the app in a goroutine and waits until it announces readiness.
func start(t *testing.T, a *app, out, errb *syncBuf, args ...string) *running {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan int, 1)
	go func() { done <- a.run(ctx, args) }()

	deadline := time.After(20 * time.Second)
	for !strings.Contains(out.String(), "is ready") {
		select {
		case code := <-done:
			cancel()
			t.Fatalf("exited early (code %d)\nstdout: %s\nstderr: %s", code, out.String(), errb.String())
		case <-deadline:
			cancel()
			t.Fatalf("never became ready\nstdout: %s\nstderr: %s", out.String(), errb.String())
		case <-time.After(20 * time.Millisecond):
		}
	}
	var once sync.Once
	var code int
	stop := func() int {
		once.Do(func() { cancel(); code = <-done })
		return code
	}
	t.Cleanup(func() { stop() })
	return &running{out: out, errb: errb, stop: stop}
}

func newTestApp() (*app, *syncBuf, *syncBuf) {
	out, errb := &syncBuf{}, &syncBuf{}
	a := newApp(out, errb)
	a.openBrowser = func(string) error { return nil }
	a.headless = func() bool { return false }
	a.inContainer = func() bool { return false }
	a.lanAddrs = func() []netinfo.Addr {
		return []netinfo.Addr{{Interface: "wlp3s0", IP: netip.MustParseAddr("192.168.15.20")}}
	}
	a.resolves = func(string, []netinfo.Addr) bool { return false }
	return a, out, errb
}

func TestVersionCommand(t *testing.T) {
	a, out, _ := newTestApp()
	if code := a.run(context.Background(), []string{"version"}); code != 0 {
		t.Fatalf("exit %d", code)
	}
	if !strings.HasPrefix(out.String(), "openjob ") {
		t.Errorf("unexpected output %q", out.String())
	}
	out.b.Reset()
	if code := a.run(context.Background(), []string{"--version"}); code != 0 || !strings.HasPrefix(out.String(), "openjob ") {
		t.Errorf("--version: code=%d out=%q", code, out.String())
	}
}

func TestUnknownCommandAndBadFlags(t *testing.T) {
	a, _, errb := newTestApp()
	if code := a.run(context.Background(), []string{"setup"}); code != 2 {
		t.Errorf("unknown command exit %d, want 2 (no `setup` subcommand exists by design)", code)
	}
	if !strings.Contains(errb.String(), "unknown command") {
		t.Errorf("stderr: %q", errb.String())
	}
	if code := a.run(context.Background(), []string{"serve", "--nope"}); code != 2 {
		t.Errorf("bad flag exit %d, want 2", code)
	}
	if code := a.run(context.Background(), []string{"serve", "extra"}); code != 2 {
		t.Errorf("stray arg exit %d, want 2", code)
	}
}

func TestInvalidConfigFailsFast(t *testing.T) {
	testEnv(t)
	a, _, errb := newTestApp()
	code := a.run(context.Background(), []string{"serve", "--port", "99999", "--db-path", filepath.Join(t.TempDir(), "x.db")})
	if code != 2 || !strings.Contains(errb.String(), "invalid port") {
		t.Errorf("code=%d stderr=%q", code, errb.String())
	}
}

func TestServeStartsHealthyAndDoesNotOpenBrowser(t *testing.T) {
	testEnv(t)
	a, out, errb := newTestApp()
	var opened atomic.Int32
	a.openBrowser = func(string) error { opened.Add(1); return nil }

	port := freePort(t)
	db := filepath.Join(t.TempDir(), "data", "openjob.db")
	r := start(t, a, out, errb, "serve", "--port", port, "--db-path", db)

	resp, err := http.Get("http://127.0.0.1:" + port + "/api/health")
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != 200 || !strings.Contains(string(body), `"version"`) {
		t.Errorf("health: %d %s", resp.StatusCode, body)
	}

	if opened.Load() != 0 {
		t.Error("`serve` must never open a browser")
	}
	for _, want := range []string{"http://localhost:" + port, "http://192.168.15.20:" + port, "wlp3s0", "web app"} {
		if !strings.Contains(r.out.String(), want) {
			t.Errorf("banner missing %q:\n%s", want, r.out.String())
		}
	}
	if strings.Contains(r.out.String(), "Name:") {
		t.Error("friendly hostname promised although it did not resolve")
	}
	if code := r.stop(); code != 0 {
		t.Errorf("graceful shutdown exit %d", code)
	}
}

func TestDesktopOpensBrowserOnlyAfterHealthy(t *testing.T) {
	testEnv(t)
	a, out, errb := newTestApp()
	port := freePort(t)

	var openedURL atomic.Value
	var healthyAtOpen atomic.Bool
	a.openBrowser = func(u string) error {
		openedURL.Store(u)
		resp, err := http.Get("http://127.0.0.1:" + port + "/api/health")
		if err == nil {
			healthyAtOpen.Store(resp.StatusCode == 200)
			resp.Body.Close()
		}
		return nil
	}

	db := filepath.Join(t.TempDir(), "openjob.db")
	r := start(t, a, out, errb, "--port", port, "--db-path", db)

	// The browser call happens right after the banner; give it a moment.
	deadline := time.Now().Add(5 * time.Second)
	for openedURL.Load() == nil && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if got, _ := openedURL.Load().(string); got != "http://localhost:"+port {
		t.Errorf("opened %q", got)
	}
	if !healthyAtOpen.Load() {
		t.Error("browser was opened before /api/health answered 200")
	}
	r.stop()
}

func TestDesktopHeadlessDoesNotOpenBrowser(t *testing.T) {
	testEnv(t)
	a, out, errb := newTestApp()
	a.headless = func() bool { return true }
	var opened atomic.Int32
	a.openBrowser = func(string) error { opened.Add(1); return nil }

	r := start(t, a, out, errb, "--port", freePort(t), "--db-path", filepath.Join(t.TempDir(), "openjob.db"))
	time.Sleep(100 * time.Millisecond)
	if opened.Load() != 0 {
		t.Error("headless/SSH session must not try to launch a browser")
	}
	if !strings.Contains(r.out.String(), "Not opening a browser") {
		t.Errorf("headless banner should explain what to do:\n%s", r.out.String())
	}
	r.stop()
}

func TestDesktopUsesPerUserDataDirByDefault(t *testing.T) {
	home := testEnv(t)
	a, out, errb := newTestApp()
	r := start(t, a, out, errb, "--port", freePort(t))
	r.stop()

	var want string
	switch runtime.GOOS {
	case "windows":
		want = filepath.Join(home, "appdata", "openjob")
	case "darwin":
		want = filepath.Join(home, "Library", "Application Support", "openjob")
	default:
		want = filepath.Join(home, "xdg", "openjob")
	}
	for _, f := range []string{"openjob.db", ".master.key"} {
		if _, err := os.Stat(filepath.Join(want, f)); err != nil {
			t.Errorf("%s not created in the per-user dir %s: %v", f, want, err)
		}
	}
}

func TestContainerModeSkipsLANDiscoveryAndBrowser(t *testing.T) {
	testEnv(t)
	a, out, errb := newTestApp()
	a.inContainer = func() bool { return true }
	var opened atomic.Int32
	a.openBrowser = func(string) error { opened.Add(1); return nil }
	a.lanAddrs = func() []netinfo.Addr { t.Error("LAN discovery must not run in a container"); return nil }

	r := start(t, a, out, errb, "serve", "--port", freePort(t), "--db-path", filepath.Join(t.TempDir(), "openjob.db"))
	if opened.Load() != 0 {
		t.Error("browser opened inside a container")
	}
	if !strings.Contains(r.out.String(), "container") {
		t.Errorf("banner:\n%s", r.out.String())
	}
	r.stop()
}

func TestRestartPreservesMasterKey(t *testing.T) {
	testEnv(t)
	dir := t.TempDir()
	db := filepath.Join(dir, "openjob.db")

	run := func() {
		a, out, errb := newTestApp()
		r := start(t, a, out, errb, "serve", "--port", freePort(t), "--db-path", db)
		if code := r.stop(); code != 0 {
			t.Fatalf("exit %d", code)
		}
	}
	run()
	k1, err := os.ReadFile(filepath.Join(dir, ".master.key"))
	if err != nil || len(k1) != 32 {
		t.Fatalf("key after first run: %v len=%d", err, len(k1))
	}
	run()
	k2, _ := os.ReadFile(filepath.Join(dir, ".master.key"))
	if !bytes.Equal(k1, k2) {
		t.Error("master key was regenerated on restart")
	}
}

func TestHealthcheckCommand(t *testing.T) {
	testEnv(t)
	a, out, errb := newTestApp()
	port := freePort(t)

	// Nothing listening yet → unhealthy.
	a2, _, _ := newTestApp()
	if code := a2.run(context.Background(), []string{"healthcheck", "--port", port}); code != 1 {
		t.Errorf("no server: exit %d, want 1", code)
	}

	r := start(t, a, out, errb, "serve", "--port", port, "--db-path", filepath.Join(t.TempDir(), "openjob.db"))
	defer r.stop()

	// The container reads the port from OPENJOB_PORT, same as the server.
	t.Setenv("OPENJOB_PORT", port)
	if code := a2.run(context.Background(), []string{"healthcheck"}); code != 0 {
		t.Errorf("server up: exit %d, want 0", code)
	}
}

func TestServeStrictWhenPortBusy(t *testing.T) {
	testEnv(t)
	busy, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatal(err)
	}
	defer busy.Close()
	port := strconv.Itoa(busy.Addr().(*net.TCPAddr).Port)

	a, _, errb := newTestApp()
	code := a.run(context.Background(), []string{"serve", "--port", port, "--db-path", filepath.Join(t.TempDir(), "openjob.db")})
	if code != 1 || !strings.Contains(errb.String(), "--port") {
		t.Errorf("code=%d stderr=%q", code, errb.String())
	}
}

// `openjob serve` outside a container gets the same per-user data dir as the
// desktop mode (SSH/headless on a laptop or a .deb install), just no browser.
func TestServeOutsideContainerUsesPerUserDataDir(t *testing.T) {
	home := testEnv(t)
	a, out, errb := newTestApp()
	r := start(t, a, out, errb, "serve", "--port", freePort(t))
	r.stop()

	var want string
	switch runtime.GOOS {
	case "windows":
		want = filepath.Join(home, "appdata", "openjob")
	case "darwin":
		want = filepath.Join(home, "Library", "Application Support", "openjob")
	default:
		want = filepath.Join(home, "xdg", "openjob")
	}
	if _, err := os.Stat(filepath.Join(want, ".master.key")); err != nil {
		t.Errorf("serve did not bootstrap the per-user dir %s: %v", want, err)
	}
}

// In the container the DB path comes from OPENJOB_DB_PATH only — the per-user
// default must never kick in (no guessing, no silent relocation of data).
func TestContainerNeverGuessesDataDir(t *testing.T) {
	testEnv(t)
	dir := t.TempDir()
	t.Setenv("OPENJOB_DB_PATH", filepath.Join(dir, "vol", "openjob.db"))
	a, out, errb := newTestApp()
	a.inContainer = func() bool { return true }
	r := start(t, a, out, errb, "serve", "--port", freePort(t))
	r.stop()
	if _, err := os.Stat(filepath.Join(dir, "vol", ".master.key")); err != nil {
		t.Errorf("env-configured data dir not used: %v", err)
	}
}
