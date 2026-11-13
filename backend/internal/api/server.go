// Package api assembles the HTTP surface of openjob: the /api routes
// (chi router) and the SPA served from the embedded filesystem at the root.
package api

import (
	"encoding/json"
	"io/fs"
	"log/slog"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/maicondmenezes/openjob/internal/config"
	"github.com/maicondmenezes/openjob/web"
)

// New builds the root http.Handler.
//   - /api/health → {"status":"ok"} (Checkpoint 1)
//   - everything else → embedded SPA with index.html fallback so deep links
//     (e.g. /jobs, /settings after Checkpoint 2) survive a reload.
func New(cfg config.Config, logger *slog.Logger) http.Handler {
	r := chi.NewRouter()

	r.Route("/api", func(r chi.Router) {
		r.Get("/health", handleHealth(cfg))
		r.NotFound(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
		})
	})

	r.Get("/*", spaHandler(logger))

	return r
}

func handleHealth(cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{
			"status":   "ok",
			"hostname": cfg.Hostname,
		})
	}
}

// spaHandler serves files from the embedded SPA bundle. Any path that does
// not match a real file falls back to index.html (SPA client routing).
func spaHandler(logger *slog.Logger) http.HandlerFunc {
	dist, err := fs.Sub(web.Dist, "dist")
	if err != nil {
		logger.Error("embedded spa bundle missing", "err", err)
	}
	fileServer := http.FileServer(http.FS(dist))

	return func(w http.ResponseWriter, r *http.Request) {
		if dist == nil {
			http.Error(w, "spa assets unavailable", http.StatusServiceUnavailable)
			return
		}
		path := strings.TrimPrefix(r.URL.Path, "/")
		if path == "" {
			path = "index.html"
		}
		if f, err := dist.Open(path); err == nil {
			_ = f.Close()
			fileServer.ServeHTTP(w, r)
			return
		}
		// SPA fallback: serve index.html for unknown paths.
		data, err := fs.ReadFile(dist, "index.html")
		if err != nil {
			http.Error(w, "index.html missing", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(data)
	}
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
