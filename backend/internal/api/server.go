// Package api assembles the HTTP surface of openjob: the /api routes
// (chi router) and the SPA served from the embedded filesystem at the root.
package api

import (
	"context"
	"encoding/json"
	"io/fs"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/maicondmenezes/openjob/internal/config"
	"github.com/maicondmenezes/openjob/internal/deterministic"
	"github.com/maicondmenezes/openjob/internal/discovery"
	"github.com/maicondmenezes/openjob/internal/events"
	"github.com/maicondmenezes/openjob/internal/llm"
	"github.com/maicondmenezes/openjob/internal/notifier"
	"github.com/maicondmenezes/openjob/internal/ranking"
	"github.com/maicondmenezes/openjob/internal/store"
	"github.com/maicondmenezes/openjob/internal/tasks"
	"github.com/maicondmenezes/openjob/internal/version"
	"github.com/maicondmenezes/openjob/web"
	"golang.org/x/time/rate"
)

// Runtime holds the background services started by New so main can stop them
// gracefully on shutdown.
type Runtime struct {
	Bus      *events.Bus
	Pool     *tasks.Pool
	Notifier *notifier.InApp
	Orch     *discovery.Orchestrator
	Email    *tasks.EmailSyncScheduler
}

// Stop terminates the background services in dependency order (HTTP server
// shutdown happens before this, in main).
func (rt *Runtime) Stop() {
	if rt.Email != nil {
		rt.Email.Stop()
	}
	if rt.Orch != nil {
		rt.Orch.Stop()
	}
	if rt.Pool != nil {
		rt.Pool.Stop()
	}
	if rt.Notifier != nil {
		rt.Notifier.Stop()
	}
	if rt.Bus != nil {
		rt.Bus.Close()
	}
}

// New builds the root http.Handler and starts the background services.
//   - /api/health → {"status":"ok"} (Checkpoint 1)
//   - /api/features → release flags from config (Phase 0)
//   - /api/settings/* → BYOK configuration (Checkpoint 3)
//   - /api/curriculum/* → base resume + AI tailoring (Checkpoint 4)
//   - /api/jobs/* + /api/applications → manual jobs, scoring, pipeline (CP5)
//   - /api/settings/notifications + /api/settings/tasks (CP10)
//   - everything else → embedded SPA with index.html fallback so deep links
//     (e.g. /jobs, /settings after Checkpoint 2) survive a reload.
func New(cfg config.Config, logger *slog.Logger, repo store.Repository, masterKey []byte, llmClient llm.Client) (http.Handler, *Runtime) {
	r := chi.NewRouter()

	// CP10: in-process event bus (docs/adr/0009) + in-app subscriber.
	bus := events.New()
	appNotifier := notifier.NewInApp(repo, logger)
	appNotifier.Start(bus)

	orch := discovery.NewOrchestrator(repo, nil, masterKey, llmClient, logger)

	// Fase 5 / B11: deterministic infrastructure (NLP core, keywords, ranking,
	// filtering). It is always constructed so feature flags can enable it at
	// runtime without changing wiring; the legacy paths stay available.
	detSvc, detErr := deterministic.New(deterministic.DefaultOptions(repo))
	if detErr != nil {
		logger.Error("deterministic service unavailable", "err", detErr)
	}
	if detSvc != nil {
		if cfg.Features.KeywordsEnabled() {
			orch.SetKeywordExtractor(func(md string, topN int) []string {
				kws, err := detSvc.ExtractKeywordStrings(context.Background(), md, topN)
				if err != nil {
					return nil
				}
				return kws
			})
		}
		if cfg.Features.FilterEnabled() {
			orch.SetResumeFilterer(detSvc)
		}
		if cfg.Features.ScoreDeterministic {
			orch.SetJobIndexer(detSvc)
			orch.SetDeterministicScorer(func(ctx context.Context, curriculumMD, title, description string) (int, string, error) {
				res, err := detSvc.ScoreJob(ctx, curriculumMD, title, description)
				if err != nil {
					return 0, "", err
				}
				return res.Score, res.Reason, nil
			})
			rebuildJobIndex(context.Background(), repo, detSvc, logger)
		}
	}

	// CP10: durable task queue (docs/adr/0010) with the migrated AI handlers.
	// CP14 adds discover_sources so repeated "Buscar agora" clicks are queued
	// and serialized instead of being rejected/lost.
	registry := tasks.NewRegistry()
	scoreHandler := tasks.NewScoreJobHandler(repo, masterKey, llmClient, rate.NewLimiter(rate.Every(300*time.Millisecond), 1), logger)
	if detSvc != nil {
		scoreHandler.SetDeterministic(detSvc, cfg.Features.ScoreDeterministic)
	}
	registry.Register(scoreHandler)
	keywordHandler := tasks.NewExtractKeywordsHandler(repo)
	if detSvc != nil {
		keywordHandler.SetDeterministic(detSvc, cfg.Features.KeywordsEnabled())
	}
	registry.Register(keywordHandler)
	registry.Register(tasks.NewDiscoverHandler(orch, logger))
	registry.Register(tasks.NewSyncEmailHandler(repo, masterKey, llmClient, bus, logger, nil))
	taskCfg, _ := tasks.LoadConfig(context.Background(), repo)
	pool := tasks.NewPool(repo, registry, bus, taskCfg, logger)
	pool.Start()

	orch.SetScoreEnqueuer(tasks.DiscoveryEnqueuer{Pool: pool})
	orch.SetPublisher(bus)
	if discCfg, err := orch.LoadConfig(context.Background()); err == nil {
		orch.ConfigureTicker(discCfg.DiscoveryInterval)
	}

	// CP17: opt-in email polling. Idle until an account is enabled.
	emailScheduler := tasks.NewEmailSyncScheduler(repo, pool, logger)
	emailScheduler.Start(context.Background())

	runtime := &Runtime{Bus: bus, Pool: pool, Notifier: appNotifier, Orch: orch, Email: emailScheduler}

	r.Route("/api", func(r chi.Router) {
		r.Get("/health", handleHealth(cfg, repo))
		r.Get("/features", handleFeatures(cfg))
		registerSettings(r, &settingsAPI{
			repo:      repo,
			masterKey: masterKey,
			llm:       llmClient,
			logger:    logger,
		})
		registerCurriculum(r, &curriculumAPI{
			repo:      repo,
			masterKey: masterKey,
			llm:       llmClient,
			logger:    logger,
		})
		registerJobs(r, &jobsAPI{
			repo:          repo,
			masterKey:     masterKey,
			llm:           llmClient,
			logger:        logger,
			bus:           bus,
			deterministic: detSvc,
			features:      cfg.Features,
			scoreEnqueuer: tasks.DiscoveryEnqueuer{Pool: pool},
		})
		registerSources(r, &sourcesAPI{
			repo:         repo,
			orchestrator: orch,
			masterKey:    masterKey,
			logger:       logger,
		})
		registerDiscovery(r, &discoveryAPI{
			orchestrator: orch,
			repo:         repo,
			logger:       logger,
			pool:         pool,
		})
		registerNotifications(r, &notificationsAPI{
			repo:   repo,
			logger: logger,
		})
		registerEmailAPI(r, &emailAPI{
			repo:      repo,
			masterKey: masterKey,
			llm:       llmClient,
			logger:    logger,
			pool:      pool,
			bus:       bus,
			scheduler: emailScheduler,
		})
		registerTasksAPI(r, &tasksAPI{
			repo:   repo,
			pool:   pool,
			bus:    bus,
			logger: logger,
		})
		r.NotFound(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
		})
	})

	r.Get("/*", spaHandler(logger))

	return r, runtime
}

// handleHealth answers GET /api/health with liveness plus the CP10 queue and
// bell counters (useful for scripts and the status UI).
func handleHealth(cfg config.Config, repo store.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()

		payload := map[string]any{
			"status":   "ok",
			"version":  version.Version,
			"hostname": cfg.Hostname,
			"tasks": map[string]any{
				"queued":          0,
				"running":         0,
				"failed_last_24h": 0,
			},
			"notifications": map[string]any{"unread": 0},
		}
		if counts, err := repo.CountTasksByStatus(ctx); err == nil {
			payload["tasks"] = map[string]any{
				"queued":          counts["queued"],
				"running":         counts["running"],
				"failed_last_24h": failedSince(ctx, repo),
			}
		}
		if unread, err := repo.CountUnreadNotifications(ctx); err == nil {
			payload["notifications"] = map[string]any{"unread": unread}
		}
		writeJSON(w, http.StatusOK, payload)
	}
}

// handleFeatures exposes the release flags so the SPA can gate unfinished UI.
// Keys mirror the flag names documented in the README.
func handleFeatures(cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]bool{
			"ui.feed":                cfg.Features.UIFeed,
			"email.sync":             cfg.Features.EmailSync,
			"curriculum.export":      cfg.Features.CurriculumExport,
			"score.deterministic":    cfg.Features.ScoreDeterministic,
			"keywords.deterministic": cfg.Features.KeywordsEnabled(),
			"filter.deterministic":   cfg.Features.FilterEnabled(),
		})
	}
}

func failedSince(ctx context.Context, repo store.Repository) int {
	n, err := repo.CountTasksFailedSince(ctx, time.Now().UTC().Add(-24*time.Hour))
	if err != nil {
		return 0
	}
	return n
}

// rebuildJobIndex repopulates the global FTS index from the jobs table. It is
// best-effort: a failure is logged and the legacy path keeps working.
func rebuildJobIndex(ctx context.Context, repo store.Repository, svc *deterministic.Service, logger *slog.Logger) {
	jobs, err := repo.ListJobs(ctx, "")
	if err != nil {
		logger.Warn("rebuild job index: list jobs failed", "err", err)
		return
	}
	texts := jobTexts(jobs)
	if err := svc.RebuildIndex(ctx, texts); err != nil {
		logger.Warn("rebuild job index failed", "err", err)
		return
	}
	logger.Info("job search index ready", "documents", len(texts))
}

// jobTexts converts stored jobs into index documents, defensively deduplicating
// the corpus. The database already enforces UNIQUE(url); this also collapses
// URL-less manual duplicates by normalized content.
func jobTexts(jobs []store.Job) []ranking.JobText {
	seenURL := make(map[string]struct{}, len(jobs))
	seenContent := make(map[string]struct{}, len(jobs))
	texts := make([]ranking.JobText, 0, len(jobs))
	for _, job := range jobs {
		if url := strings.ToLower(strings.TrimSpace(job.URL)); url != "" {
			if _, ok := seenURL[url]; ok {
				continue
			}
			seenURL[url] = struct{}{}
		} else {
			content := strings.ToLower(strings.Join(strings.Fields(job.Title+"\n"+job.Company+"\n"+job.Description), " "))
			if _, ok := seenContent[content]; ok {
				continue
			}
			seenContent[content] = struct{}{}
		}
		texts = append(texts, ranking.JobText{JobID: job.ID, Title: job.Title, Description: job.Description})
	}
	return texts
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
		// Content-types que o mime padrão do Go não resolve.
		if r.URL.Path == "/manifest.webmanifest" {
			w.Header().Set("Content-Type", "application/manifest+json; charset=utf-8")
		}
		// Assets imutáveis do Vite (hash no nome): cache agressivo.
		if strings.HasPrefix(r.URL.Path, "/_app/immutable/") {
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
			fileServer.ServeHTTP(w, r)
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
		// SPA fallback: serve index.html for unknown paths (deep links).
		// no-cache: o shell precisa revalidar para pegar novos deploys.
		data, err := fs.ReadFile(dist, "index.html")
		if err != nil {
			http.Error(w, "index.html missing", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-cache")
		_, _ = w.Write(data)
	}
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

// writeAPIError emits the additive error contract of CP13 / ADR 0021:
//
//	{ "code": "job.not_found", "params": { "id": 999 }, "message": "job not found" }
//
// The legacy `error` field is kept with the same English message so older
// clients (and internal/debug errors) keep working without a breaking change.
// `message` is always developer-facing English; the frontend translates
// known `code`s and falls back to `message` for unknown ones.
func writeAPIError(w http.ResponseWriter, status int, code, message string, params map[string]any) {
	payload := map[string]any{
		"code":    code,
		"message": message,
		"error":   message, // backward compatibility with pre-CP13 clients
	}
	if len(params) > 0 {
		payload["params"] = params
	}
	writeJSON(w, status, payload)
}
