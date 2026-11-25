package bootstrap

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"time"
)

// Port range scanned when the person did not choose a port and the default is
// taken (e.g. another openjob or a dev server already owns :8080).
const (
	defaultPort  = 8080
	fallbackSpan = 20 // 8080..8099
)

// Listen binds the HTTP port and returns the live listener, so the port that
// was checked is the port that gets served (no check-then-bind race).
//
//   - explicit=true (flag/env): bind exactly that port or fail with an
//     actionable message — never silently move to another port.
//   - explicit=false (default): try 8080, then the next ports up to 8099.
func Listen(port string, explicit bool) (net.Listener, error) {
	if explicit {
		ln, err := net.Listen("tcp", ":"+port)
		if err != nil {
			return nil, fmt.Errorf("port %s is unavailable: %w\n  → free it, or pick another with --port or OPENJOB_PORT", port, err)
		}
		return ln, nil
	}

	start, err := strconv.Atoi(port)
	if err != nil {
		start = defaultPort
	}
	var lastErr error
	for p := start; p < start+fallbackSpan && p <= 65535; p++ {
		ln, err := net.Listen("tcp", ":"+strconv.Itoa(p))
		if err == nil {
			return ln, nil
		}
		lastErr = err
	}
	return nil, fmt.Errorf("no free port in %d-%d: %w\n  → free one, or pick a port with --port or OPENJOB_PORT",
		start, start+fallbackSpan-1, lastErr)
}

// PortOf extracts the numeric port from a listener address.
func PortOf(ln net.Listener) string {
	if a, ok := ln.Addr().(*net.TCPAddr); ok {
		return strconv.Itoa(a.Port)
	}
	_, p, _ := net.SplitHostPort(ln.Addr().String())
	return p
}

// WaitHealthy polls url (GET /api/health) until it answers 200 with
// {"status":"ok"} or the timeout/ctx expires. The binary only announces
// "ready" — and only then opens the browser — after this returns nil.
func WaitHealthy(ctx context.Context, url string, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	client := &http.Client{Timeout: 2 * time.Second}
	tick := time.NewTicker(100 * time.Millisecond)
	defer tick.Stop()

	var last error
	for {
		last = checkOnce(ctx, client, url)
		if last == nil {
			return nil
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("health check %s did not pass within %s: %w", url, timeout, last)
		case <-tick.C:
		}
	}
}

// Check performs a single health probe (used by `openjob healthcheck`).
func Check(ctx context.Context, url string, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	return checkOnce(ctx, &http.Client{Timeout: timeout}, url)
}

func checkOnce(ctx context.Context, c *http.Client, url string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := c.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("status %d", resp.StatusCode)
	}
	var body struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return fmt.Errorf("decoding health body: %w", err)
	}
	if body.Status != "ok" {
		return errors.New("health status is not ok: " + body.Status)
	}
	return nil
}
