package api

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/maicondmenezes/openjob/internal/config"
)

func newTestConfig() config.Config {
	return config.Config{Port: "0", DBPath: "/data/openjob.db", Hostname: "openjob.local"}
}

func TestHealthEndpoint(t *testing.T) {
	srv := httptest.NewServer(New(newTestConfig(), slog.Default()))
	defer srv.Close()

	resp, err := srv.Client().Get(srv.URL + "/api/health")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), `"status":"ok"`) {
		t.Errorf("body = %s, want it to contain \"status\":\"ok\"", body)
	}
	if !strings.Contains(string(body), `"hostname":"openjob.local"`) {
		t.Errorf("body = %s, want hostname echoed", body)
	}
}

func TestAPIUnknownRouteReturnsJSON404(t *testing.T) {
	srv := httptest.NewServer(New(newTestConfig(), slog.Default()))
	defer srv.Close()

	resp, err := srv.Client().Get(srv.URL + "/api/nope")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); !strings.Contains(ct, "application/json") {
		t.Errorf("content-type = %q, want application/json", ct)
	}
}

func TestSPAFallbackServesIndexForUnknownPath(t *testing.T) {
	srv := httptest.NewServer(New(newTestConfig(), slog.Default()))
	defer srv.Close()

	resp, err := srv.Client().Get(srv.URL + "/some/deep/link")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200 (SPA fallback)", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); !strings.Contains(ct, "text/html") {
		t.Errorf("content-type = %q, want text/html", ct)
	}
	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "openJob") {
		t.Errorf("body = %s, want placeholder index.html", body)
	}
}

func TestSPARootServesIndex(t *testing.T) {
	srv := httptest.NewServer(New(newTestConfig(), slog.Default()))
	defer srv.Close()

	resp, err := srv.Client().Get(srv.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
}
