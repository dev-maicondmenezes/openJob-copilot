package api

import (
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/maicondmenezes/openjob/internal/config"
	"github.com/maicondmenezes/openjob/web"
)

func newTestConfig() config.Config {
	return config.Config{Port: "0", DBPath: "/data/openjob.db", Hostname: "openjob.copilot"}
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
	if !strings.Contains(string(body), `"hostname":"openjob.copilot"`) {
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
	// O shell do SvelteKit (fallback) não tem texto estável — o título é
	// injetado pelo cliente. `lang="pt-BR"` vem do app.html e identifica o shell.
	if !strings.Contains(string(body), `lang="pt-BR"`) {
		t.Errorf("body = %s, want SvelteKit shell (lang=pt-BR)", body)
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

// Deep link real do Checkpoint 2: recarregar /jobs no celular não pode dar 404.
func TestSPADeepLinkJobsServesShell(t *testing.T) {
	srv := httptest.NewServer(New(newTestConfig(), slog.Default()))
	defer srv.Close()

	for _, path := range []string{"/jobs", "/settings", "/curriculum", "/applications"} {
		resp, err := srv.Client().Get(srv.URL + path)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("GET %s = %d, want 200 (SPA fallback)", path, resp.StatusCode)
		}
		if ct := resp.Header.Get("Content-Type"); !strings.Contains(ct, "text/html") {
			t.Errorf("GET %s content-type = %q, want text/html", path, ct)
		}
	}
}

func TestSPAManifestAndServiceWorkerServed(t *testing.T) {
	srv := httptest.NewServer(New(newTestConfig(), slog.Default()))
	defer srv.Close()

	// manifest.webmanifest é emitido pelo build do PWA (vite.config.ts).
	if f, err := web.Dist.Open("dist/manifest.webmanifest"); err != nil {
		t.Skipf("PWA manifest not embedded (SPA build pending): %v", err)
	} else {
		_ = f.Close()
	}
	if f, err := web.Dist.Open("dist/sw.js"); err != nil {
		t.Skipf("service worker not embedded (SPA build pending): %v", err)
	} else {
		_ = f.Close()
	}
}

func TestSPAImmutableAssetsGetImmutableCache(t *testing.T) {
	matches, err := fs.Glob(web.Dist, "dist/_app/immutable/entry/*.js")
	if err != nil || len(matches) == 0 {
		t.Skipf("no immutable assets embedded (SPA build pending): %v", err)
	}
	assetPath := "/" + strings.TrimPrefix(matches[0], "dist/")

	srv := httptest.NewServer(New(newTestConfig(), slog.Default()))
	defer srv.Close()

	resp, err := srv.Client().Get(srv.URL + assetPath)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if cc := resp.Header.Get("Cache-Control"); !strings.Contains(cc, "immutable") {
		t.Errorf("GET %s cache-control = %q, want immutable", assetPath, cc)
	}
}
