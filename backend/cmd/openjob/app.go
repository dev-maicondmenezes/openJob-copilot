package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/netip"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/maicondmenezes/openjob/internal/api"
	"github.com/maicondmenezes/openjob/internal/bootstrap"
	"github.com/maicondmenezes/openjob/internal/browser"
	"github.com/maicondmenezes/openjob/internal/config"
	"github.com/maicondmenezes/openjob/internal/llm"
	"github.com/maicondmenezes/openjob/internal/netinfo"
	"github.com/maicondmenezes/openjob/internal/store"
	"github.com/maicondmenezes/openjob/internal/version"
)

const usage = `openJob — local-first job application copilot

Usage:
  openjob [flags]              prepare, start the server and open the browser (desktop)
  openjob serve [flags]        start the server only (SSH / headless / containers)
  openjob version              print the version
  openjob healthcheck          probe the local server (used by the Docker HEALTHCHECK)

Flags (override the matching environment variable):
  --port <n>         HTTP port             (OPENJOB_PORT,     default 8080)
  --db-path <file>   SQLite database path  (OPENJOB_DB_PATH,  default: per-user data dir; /data/openjob.db in the image)
  --hostname <name>  LAN hostname announced (OPENJOB_HOSTNAME, default openjob.copilot)

Precedence: flag > environment variable > default.
Product setup (AI provider, resume, sources, Telegram) happens in the web app.
`

// app bundles the process-level dependencies so every run mode is testable
// without a real browser, network interfaces or terminal.
type app struct {
	stdout, stderr io.Writer
	logger         *slog.Logger

	openBrowser func(url string) error
	headless    func() bool
	inContainer func() bool
	lanAddrs    func() []netinfo.Addr
	resolves    func(host string, lan []netinfo.Addr) bool // does the LAN hostname point at this machine?
}

func newApp(stdout, stderr io.Writer) *app {
	return &app{
		stdout:      stdout,
		stderr:      stderr,
		logger:      slog.New(slog.NewTextHandler(stdout, nil)),
		openBrowser: browser.Open,
		headless: func() bool {
			return browser.IsHeadless(runtime.GOOS, os.Getenv)
		},
		inContainer: runningInContainer,
		lanAddrs:    netinfo.Discover,
		resolves:    hostResolves,
	}
}

// run dispatches the subcommand and returns the process exit code.
func (a *app) run(ctx context.Context, args []string) int {
	cmd, rest := "", args
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		cmd, rest = args[0], args[1:]
	}

	switch cmd {
	case "":
		// `openjob --version` / `-h` are handled by parseFlags below.
		return a.cmdServe(ctx, rest, true)
	case "serve":
		return a.cmdServe(ctx, rest, false)
	case "version":
		fmt.Fprintln(a.stdout, version.String())
		return 0
	case "healthcheck":
		return a.cmdHealthcheck(ctx, rest)
	case "help":
		fmt.Fprint(a.stdout, usage)
		return 0
	default:
		fmt.Fprintf(a.stderr, "openjob: unknown command %q\n\n%s", cmd, usage)
		return 2
	}
}

// parseFlags resolves flag > env > default into a Config. Returns done=true
// (with an exit code) when the invocation was fully handled (-h, --version, or
// a flag error).
func (a *app) parseFlags(name string, args []string) (cfg config.Config, done bool, code int) {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(a.stderr)
	fs.Usage = func() { fmt.Fprint(a.stderr, usage) }

	var o config.Overrides
	var showVersion bool
	fs.StringVar(&o.Port, "port", "", "HTTP port")
	fs.StringVar(&o.DBPath, "db-path", "", "SQLite database path")
	fs.StringVar(&o.Hostname, "hostname", "", "LAN hostname")
	fs.BoolVar(&showVersion, "version", false, "print the version")

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return cfg, true, 0
		}
		return cfg, true, 2
	}
	if showVersion {
		fmt.Fprintln(a.stdout, version.String())
		return cfg, true, 0
	}
	if fs.NArg() > 0 {
		fmt.Fprintf(a.stderr, "openjob: unexpected argument %q\n", fs.Arg(0))
		return cfg, true, 2
	}
	return config.LoadWith(o), false, 0
}

// cmdServe implements both `openjob` (desktop=true) and `openjob serve`.
//
// Two independent switches:
//   - binary mode (everything except the container image): a per-user data dir
//     and a free port are chosen only when the person didn't set them.
//   - desktop (`openjob` without a subcommand): additionally opens the browser
//     once the server is healthy, and only on a GUI session.
//
// Inside a container nothing is guessed: configuration is explicit
// (OPENJOB_* env vars) and strict, and no browser or LAN discovery is tried.
func (a *app) cmdServe(ctx context.Context, args []string, desktop bool) int {
	name := "openjob"
	if !desktop {
		name = "openjob serve"
	}
	cfg, done, code := a.parseFlags(name, args)
	if done {
		return code
	}
	binaryMode := !a.inContainer()
	if binaryMode {
		cfg.ApplyDesktopDefaults()
	}
	if err := cfg.Validate(); err != nil {
		fmt.Fprintf(a.stderr, "openjob: invalid configuration:\n%v\n", err)
		return 2
	}

	// 1. Minimal technical bootstrap: data dir + master key (idempotent).
	prep, err := bootstrap.Prepare(cfg)
	if err != nil {
		fmt.Fprintf(a.stderr, "openjob: bootstrap failed: %v\n", err)
		return 1
	}
	for _, w := range prep.Warnings {
		a.logger.Warn(w)
	}
	a.logger.Info("master key ready", "path", prep.KeyPath, "created", prep.KeyCreated)

	// 2. Bind the port (kept open: no check-then-bind race).
	strict := cfg.PortExplicit || !binaryMode
	ln, err := bootstrap.Listen(cfg.Port, strict)
	if err != nil {
		fmt.Fprintf(a.stderr, "openjob: %v\n", err)
		return 1
	}
	cfg.Port = bootstrap.PortOf(ln)

	// 3. Store (SQLite + embedded migrations — the existing mechanism).
	repo, err := store.Open(cfg.DBPath)
	if err != nil {
		ln.Close()
		fmt.Fprintf(a.stderr, "openjob: opening store: %v\n", err)
		return 1
	}

	// CP10: New also starts the event bus, the in-app notifier and the task
	// worker pool; runtime.Stop tears them down before the store closes.
	handler, rt := api.New(cfg, a.logger, repo, prep.MasterKey, llm.NewOpenAIClient())

	srv := &http.Server{
		Handler:     handler,
		ReadTimeout: 10 * time.Second,
		// WriteTimeout must exceed the LLM call budget (llm.requestTimeout =
		// 120s): completions legitimately take 20–60s+. With the old 15s the
		// server killed the connection mid-completion and the client saw
		// "empty reply from server" / RemoteDisconnected.
		WriteTimeout: 150 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		a.logger.Info("openjob starting",
			"version", version.Version,
			"port", cfg.Port,
			"db_path", cfg.DBPath,
			"hostname", cfg.Hostname,
		)
		if err := srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	shutdown := func() int {
		sctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		code := 0
		if err := srv.Shutdown(sctx); err != nil {
			a.logger.Error("graceful shutdown failed", "err", err)
			code = 1
		}
		rt.Stop() // task pool (drains ctx), notifier, bus, discovery ticker
		if err := repo.Close(); err != nil {
			a.logger.Warn("store close", "err", err)
		}
		if code == 0 {
			a.logger.Info("shutdown complete")
		}
		return code
	}

	// 4. Announce "ready" only after the health check really passes.
	healthURL := "http://127.0.0.1:" + cfg.Port + "/api/health"
	if err := bootstrap.WaitHealthy(ctx, healthURL, 15*time.Second); err != nil {
		fmt.Fprintf(a.stderr, "openjob: server did not become healthy: %v\n", err)
		shutdown()
		return 1
	}
	a.announce(cfg, prep, desktop)

	// 5. Browser: desktop mode on a GUI session only, after the health check.
	if desktop && !a.headless() && !a.inContainer() {
		if err := a.openBrowser("http://localhost:" + cfg.Port); err != nil {
			fmt.Fprintf(a.stdout, "  (could not open the browser automatically: %v — open the URL above manually)\n", err)
		}
	}

	select {
	case err := <-errCh:
		a.logger.Error("server error", "err", err)
		shutdown()
		return 1
	case <-ctx.Done():
		a.logger.Info("shutting down")
		return shutdown()
	}
}

// announce prints the human-facing "ready" banner: the real URLs, nothing
// promised that wasn't verified.
func (a *app) announce(cfg config.Config, prep bootstrap.Result, desktop bool) {
	w := a.stdout
	fmt.Fprintf(w, "\nopenJob %s is ready\n", version.Version)

	// Inside a container, interface addresses are the bridge network's, not
	// something a phone can reach: the published host port decides the URL.
	if a.inContainer() {
		fmt.Fprintf(w, "  Listening on :%s (container) — open it through the host port you published\n", cfg.Port)
		fmt.Fprintf(w, "  Data:  %s\n\n", prep.DataDir)
		return
	}

	fmt.Fprintf(w, "  Local: http://localhost:%s\n", cfg.Port)
	lan := a.lanAddrs()
	for _, ad := range lan {
		fmt.Fprintf(w, "  LAN:   http://%s:%s   (%s)\n", ad.IP, cfg.Port, ad.Interface)
	}
	if len(lan) > 0 && a.resolves(cfg.Hostname, lan) {
		fmt.Fprintf(w, "  Name:  http://%s:%s   (resolves on this machine)\n", cfg.Hostname, cfg.Port)
	}
	fmt.Fprintf(w, "  Data:  %s\n", prep.DataDir)

	if len(lan) > 0 {
		fmt.Fprintf(w, "\n  Phone access needs the same Wi-Fi/LAN and a firewall rule allowing port %s.\n", cfg.Port)
		fmt.Fprintln(w, "  Android/Chrome can't resolve .local names — use the LAN URL by IP.")
	} else {
		fmt.Fprintln(w, "\n  No LAN address found (offline or only VPN/virtual interfaces): only localhost is available.")
	}
	if !desktop || a.headless() {
		fmt.Fprintln(w, "  Not opening a browser (serve / SSH / headless). Open the URL above from a machine with a GUI.")
	}
	fmt.Fprintln(w, "  First-time setup (AI provider, resume, sources) happens in the web app.")
	fmt.Fprintln(w)
}

// cmdHealthcheck is the container HEALTHCHECK: the distroless runtime has no
// curl/wget, so the binary probes itself. Port comes from --port or
// OPENJOB_PORT (the same variable the server reads).
func (a *app) cmdHealthcheck(ctx context.Context, args []string) int {
	cfg, done, code := a.parseFlags("openjob healthcheck", args)
	if done {
		return code
	}
	url := "http://127.0.0.1:" + cfg.Port + "/api/health"
	if err := bootstrap.Check(ctx, url, 4*time.Second); err != nil {
		fmt.Fprintf(a.stderr, "unhealthy: %v\n", err)
		return 1
	}
	return 0
}

// runningInContainer reports whether we run as the official container image,
// which sets OPENJOB_CONTAINER=true. It is deliberately an explicit opt-in and
// not a heuristic like /.dockerenv: the binary/.deb is also used inside
// devcontainers and distrobox, where it must keep the per-user defaults.
func runningInContainer() bool {
	v, err := strconv.ParseBool(os.Getenv("OPENJOB_CONTAINER"))
	return err == nil && v
}

// hostResolves reports whether host resolves, within a short timeout, to one
// of this machine's own addresses — the only case where the friendly name is
// printed. A name that resolves elsewhere (or not at all) is never promised.
func hostResolves(host string, lan []netinfo.Addr) bool {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	addrs, err := net.DefaultResolver.LookupHost(ctx, host)
	if err != nil {
		return false
	}
	for _, r := range addrs {
		ip, err := netip.ParseAddr(r)
		if err != nil {
			continue
		}
		if ip.IsLoopback() {
			return true
		}
		for _, l := range lan {
			if l.IP == ip.Unmap() {
				return true
			}
		}
	}
	return false
}
