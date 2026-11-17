// Package notifier is the in-app notification channel of the event bus
// (CP10): it maps bus events to persisted `notifications` rows that feed the
// header bell and the /notifications page. Like every bus consumer it is
// best-effort — a persistence failure is logged, never propagated to the
// producer.
package notifier

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/maicondmenezes/openjob/internal/events"
	"github.com/maicondmenezes/openjob/internal/store"
)

// SettingKey is the settings key holding the JSON notification config.
const SettingKey = "notifications.config"

// Config controls the in-app channel (§3.3). Stored as JSON in settings.
type Config struct {
	InAppEnabled         bool `json:"in_app_enabled"`
	RetentionDays        int  `json:"retention_days"`
	BrowserNotifications bool `json:"browser_notifications"`
}

// DefaultConfig mirrors §3.3: in-app on, 90 days retention, browser off.
func DefaultConfig() Config {
	return Config{InAppEnabled: true, RetentionDays: 90, BrowserNotifications: false}
}

// LoadConfig reads the config from settings, falling back to defaults when
// absent or unreadable.
func LoadConfig(ctx context.Context, repo store.Repository) (Config, error) {
	raw, err := repo.GetSetting(ctx, SettingKey)
	if err != nil {
		return DefaultConfig(), nil
	}
	var cfg Config
	if err := json.Unmarshal([]byte(raw), &cfg); err != nil {
		return DefaultConfig(), nil
	}
	if cfg.RetentionDays <= 0 {
		cfg.RetentionDays = DefaultConfig().RetentionDays
	}
	return cfg, nil
}

// SaveConfig validates and persists the config.
func SaveConfig(ctx context.Context, repo store.Repository, cfg Config) error {
	if cfg.RetentionDays < 1 || cfg.RetentionDays > 3650 {
		return fmt.Errorf("retention_days deve estar entre 1 e 3650")
	}
	b, err := json.Marshal(cfg)
	if err != nil {
		return err
	}
	return repo.SetSetting(ctx, SettingKey, string(b))
}

const (
	maxBodyLen    = 500
	dedupWindow   = 10 * time.Minute
	dedupMaxKeys  = 256
	housekeepTick = 24 * time.Hour
)

// InApp is the bus subscriber that persists notifications. Start wires it to
// a bus; Stop shuts the drain goroutine and the housekeeping ticker down.
type InApp struct {
	repo   store.Repository
	logger *slog.Logger

	sub *events.Subscription
	wg  sync.WaitGroup
	mu  sync.Mutex
	// dedup tracks the last time each DedupKey was persisted so repeated
	// events (source_error on every discovery tick) do not spam the feed.
	dedup map[string]time.Time
	stop  chan struct{}
}

// NewInApp creates the notifier; call Start to begin consuming.
func NewInApp(repo store.Repository, logger *slog.Logger) *InApp {
	if logger == nil {
		logger = slog.Default()
	}
	return &InApp{
		repo:   repo,
		logger: logger,
		dedup:  map[string]time.Time{},
		stop:   make(chan struct{}),
	}
}

// Start subscribes to the bus, drains events into the store and kicks off
// the retention housekeeping loop.
func (n *InApp) Start(bus *events.Bus) {
	n.sub = bus.Subscribe("notifier.inapp", 64)
	n.wg.Add(1)
	go n.drain()
	n.wg.Add(1)
	go n.housekeep()
}

// Stop unsubscribes and waits for the goroutines to finish.
func (n *InApp) Stop() {
	close(n.stop)
	if n.sub != nil {
		n.sub.Close()
	}
	n.wg.Wait()
}

func (n *InApp) drain() {
	defer n.wg.Done()
	for e := range n.sub.C() {
		// A single broken insert must not kill the drain loop.
		func() {
			defer func() {
				if r := recover(); r != nil {
					n.logger.Error("notifier: panic handling event", "kind", e.Kind, "panic", r)
				}
			}()
			n.handle(e)
		}()
	}
}

func (n *InApp) housekeep() {
	defer n.wg.Done()
	n.runHousekeeping()
	t := time.NewTicker(housekeepTick)
	defer t.Stop()
	for {
		select {
		case <-n.stop:
			return
		case <-t.C:
			n.runHousekeeping()
		}
	}
}

// runHousekeeping deletes read notifications older than the retention window
// (unread ones are always kept so nothing actionable is lost silently).
func (n *InApp) runHousekeeping() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cfg, err := LoadConfig(ctx, n.repo)
	if err != nil {
		return
	}
	cutoff := time.Now().UTC().AddDate(0, 0, -cfg.RetentionDays)
	deleted, err := n.repo.DeleteReadNotificationsBefore(ctx, cutoff)
	if err != nil {
		n.logger.Warn("notifier: housekeeping failed", "err", err)
		return
	}
	if deleted > 0 {
		n.logger.Info("notifier: retention cleanup", "deleted", deleted, "retention_days", cfg.RetentionDays)
	}
}

func (n *InApp) handle(e events.Event) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cfg, err := LoadConfig(ctx, n.repo)
	if err != nil {
		n.logger.Warn("notifier: load config failed", "err", err)
	}
	if !cfg.InAppEnabled {
		return // feed desligado: só os outros canais (ex.: Telegram) recebem
	}

	nb, ok := mapEvent(e)
	if !ok {
		return // kind sem representação in-app (ex.: task.progress)
	}

	if e.DedupKey != "" && n.recentlySeen(e.DedupKey) {
		return
	}

	body := nb.body
	if len(body) > maxBodyLen {
		body = body[:maxBodyLen-3] + "..."
	}

	title, persistedBody := nb.title, body
	if nb.i18nKey != "" {
		// CP13/ADR 0020: notificações novas gravam só a chave + params;
		// o texto é resolvido no frontend no locale atual. title/body
		// ficam vazios para evitar texto fixo em pt-BR no histórico.
		title, persistedBody = "", ""
	}
	if _, err := n.repo.InsertNotification(ctx, store.Notification{
		Type:       nb.notificationType,
		Severity:   nb.severity,
		Title:      title,
		Body:       persistedBody,
		I18nKey:    nb.i18nKey,
		I18nParams: nb.i18nParams,
		Link:       nb.link,
	}); err != nil {
		n.logger.Error("notifier: insert notification failed", "kind", e.Kind, "err", err)
	}
}

// recentlySeen reports whether the same dedup key was persisted inside the
// dedup window, recording the hit otherwise.
func (n *InApp) recentlySeen(key string) bool {
	now := time.Now()
	n.mu.Lock()
	defer n.mu.Unlock()
	if last, ok := n.dedup[key]; ok && now.Sub(last) < dedupWindow {
		return true
	}
	if len(n.dedup) >= dedupMaxKeys { // bound the map: drop expired entries
		for k, ts := range n.dedup {
			if now.Sub(ts) >= dedupWindow {
				delete(n.dedup, k)
			}
		}
		if len(n.dedup) >= dedupMaxKeys { // still full: reset (best-effort)
			n.dedup = map[string]time.Time{}
		}
	}
	n.dedup[key] = now
	return false
}

type notificationBuild struct {
	notificationType string
	severity         string
	title            string // mantido para compatibilidade de testes/legado
	body             string // mantido para compatibilidade de testes/legado
	link             string
	i18nKey          string // CP13: ex. "notifications.discovery.completed"
	i18nParams       string // CP13: JSON dos parâmetros ICU
}

// i18nParams marshals ICU params into the additive JSON column.
func i18nParams(values map[string]any) string {
	if len(values) == 0 {
		return ""
	}
	data, err := json.Marshal(values)
	if err != nil {
		return ""
	}
	return string(data)
}

// mapEvent translates a bus event into the persisted notification shape.
// Kinds without an in-app representation (task.progress, task.cancelled, ...)
// return ok=false and are ignored.
func mapEvent(e events.Event) (notificationBuild, bool) {
	meta := e.Metadata

	switch e.Kind {
	case events.KindApplicationStatusChanged:
		sev := events.SeverityInfo
		title := "Candidatura atualizada"
		switch metaStr(meta, "to") {
		case "sent":
			title = "Candidatura enviada"
		case "interview":
			title = "Candidatura em entrevista"
		case "offer":
			sev = events.SeveritySuccess
			title = "Oferta recebida!"
		case "rejected":
			sev = events.SeverityWarning
			title = "Candidatura rejeitada"
		case "pending":
			title = "Candidatura criada"
		}
		body := fmt.Sprintf("%s: %s → %s", metaStr(meta, "job_label"), metaStr(meta, "from"), metaStr(meta, "to"))
		if body == ":  → " {
			body = e.Body
		}
		return notificationBuild{
			notificationType: "application_status",
			severity:         sev,
			title:            title,
			body:             firstNonEmpty(body, e.Body),
			link:             firstNonEmpty(e.Link, "/applications"),
			i18nKey:          "notifications.application.status_changed",
			i18nParams: i18nParams(map[string]any{
				"job_label": metaStr(meta, "job_label"),
				"from":      metaStr(meta, "from"),
				"to":        metaStr(meta, "to"),
			}),
		}, true

	case events.KindDiscoveryCompleted:
		sev := events.SeveritySuccess
		if len(errorsOf(meta)) > 0 {
			sev = events.SeverityWarning
		}
		return notificationBuild{
			notificationType: "discovery_result",
			severity:         sev,
			title:            "Busca de vagas concluída",
			body: fmt.Sprintf("%d novas vagas cadastradas, %d na fila de scoring (%d analisadas).",
				metaInt(meta, "new_jobs"), metaInt(meta, "queued_scoring"), metaInt(meta, "fetched")),
			link:    firstNonEmpty(e.Link, "/jobs"),
			i18nKey: "notifications.discovery.completed",
			i18nParams: i18nParams(map[string]any{
				"count": metaInt(meta, "new_jobs"),
			}),
		}, true

	case events.KindDiscoveryFailed:
		return notificationBuild{
			notificationType: "discovery_result",
			severity:         events.SeverityError,
			title:            "Busca de vagas falhou",
			body:             firstNonEmpty(e.Body, "A busca automática terminou com erro."),
			link:             firstNonEmpty(e.Link, "/jobs"),
			i18nKey:          "notifications.discovery.failed",
		}, true

	case events.KindSourceFetchFailed:
		source := metaStr(meta, "source")
		return notificationBuild{
			notificationType: "source_error",
			severity:         events.SeverityWarning,
			title:            "Falha em fonte de vagas",
			body:             firstNonEmpty(e.Body, source),
			link:             firstNonEmpty(e.Link, "/settings/sources"),
			i18nKey:          "notifications.source.fetch_failed",
			i18nParams:       i18nParams(map[string]any{"source": source}),
		}, true

	case events.KindTaskCompleted:
		typ := metaStr(meta, "task_type")
		return notificationBuild{
			notificationType: "task_completed",
			severity:         events.SeveritySuccess,
			title:            taskCompletedTitle(typ),
			body:             firstNonEmpty(e.Body, taskCompletedBody(typ, meta)),
			link:             firstNonEmpty(e.Link, "/tasks"),
			i18nKey:          "notifications.task.completed",
			i18nParams:       i18nParams(map[string]any{"task": typ}),
		}, true

	case events.KindTaskFailed:
		typ := metaStr(meta, "task_type")
		return notificationBuild{
			notificationType: "task_failed",
			severity:         events.SeverityError,
			title:            "Tarefa de IA falhou",
			body: fmt.Sprintf("%s: %s", taskHumanName(typ),
				firstNonEmpty(strings.TrimSpace(e.Body), "erro desconhecido")),
			link:       firstNonEmpty(e.Link, "/tasks"),
			i18nKey:    "notifications.task.failed",
			i18nParams: i18nParams(map[string]any{"task": typ}),
		}, true

	case events.KindTest:
		return notificationBuild{
			notificationType: "test",
			severity:         firstNonEmpty(e.Severity, events.SeverityInfo),
			title:            firstNonEmpty(e.Title, "Notificação de teste"),
			body:             e.Body,
			link:             e.Link,
		}, true
	}
	return notificationBuild{}, false
}

func taskHumanName(typ string) string {
	switch typ {
	case "score_job":
		return "Score de vaga"
	case "extract_resume_keywords":
		return "Extração de palavras-chave"
	case "tailor_resume":
		return "Adaptação de currículo"
	}
	return typ
}

func taskCompletedTitle(typ string) string {
	switch typ {
	case "score_job":
		return "Score de vaga calculado"
	case "extract_resume_keywords":
		return "Palavras-chave extraídas"
	case "tailor_resume":
		return "Currículo adaptado"
	}
	return "Tarefa concluída"
}

func taskCompletedBody(typ string, meta map[string]any) string {
	label := metaStr(meta, "job_label")
	switch typ {
	case "score_job":
		score := metaInt(meta, "score_fit")
		if score > 0 || label != "" {
			if label != "" {
				return fmt.Sprintf("%s — score %d/100.", label, score)
			}
			return fmt.Sprintf("Score calculado: %d/100.", score)
		}
	case "extract_resume_keywords":
		if n := metaInt(meta, "count"); n > 0 {
			return fmt.Sprintf("%d palavras-chave extraídas do currículo.", n)
		}
	}
	return taskHumanName(typ) + " concluída."
}

// helpers to read metadata maps
func metaStr(meta map[string]any, key string) string {
	if v, ok := meta[key].(string); ok {
		return v
	}
	return ""
}

func metaInt(meta map[string]any, key string) int {
	switch v := meta[key].(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	}
	return 0
}

func errorsOf(meta map[string]any) []string {
	if v, ok := meta["errors"].([]string); ok {
		return v
	}
	return nil
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
