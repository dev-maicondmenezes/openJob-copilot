package api

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/maicondmenezes/openjob/internal/notifier"
	"github.com/maicondmenezes/openjob/internal/store"
)

// notificationsAPI serves the in-app notification feed (CP10).
type notificationsAPI struct {
	repo   store.Repository
	logger *slog.Logger
}

// registerNotifications mounts the notification routes + the channel config.
func registerNotifications(r chi.Router, n *notificationsAPI) {
	r.Get("/notifications", n.list)
	r.Get("/notifications/unread-count", n.unreadCount)
	r.Post("/notifications/read-all", n.readAll)
	r.Post("/notifications/{id}/read", n.markRead)
	r.Delete("/notifications/{id}", n.delete)
	r.Delete("/notifications", n.clearRead)

	r.Get("/settings/notifications", n.getConfig)
	r.Put("/settings/notifications", n.updateConfig)
}

// nullableString sends empty strings as JSON null so the frontend can use
// `i18n_key ? ... : title + body` without extra checks.
func nullableString(v string) any {
	if v == "" {
		return nil
	}
	return v
}

// notificationPayload renders a notification row for the UI (snake_case).
func notificationPayload(n store.Notification) map[string]any {
	item := map[string]any{
		"id":          n.ID,
		"type":        n.Type,
		"severity":    n.Severity,
		"title":       n.Title,
		"body":        n.Body,
		"link":        n.Link,
		"metadata":    n.Metadata,
		"i18n_key":    nullableString(n.I18nKey),
		"i18n_params": nullableString(n.I18nParams),
		"read_at":     nil,
		"created_at":  n.CreatedAt,
	}
	if n.ReadAt != nil {
		item["read_at"] = n.ReadAt
	}
	return item
}

// ---- GET /api/notifications -------------------------------------------------

func (n *notificationsAPI) list(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	limit, _ := strconv.Atoi(q.Get("limit"))
	if limit <= 0 {
		limit = 50
	}
	f := store.NotificationFilter{
		UnreadOnly: q.Get("unread") == "1",
		Type:       q.Get("type"),
		Severity:   q.Get("severity"),
		Limit:      limit,
		Cursor:     q.Get("cursor"),
	}
	items, next, err := n.repo.ListNotifications(r.Context(), f)
	if err != nil {
		n.logger.Error("list notifications failed", "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "falha ao listar notificações"})
		return
	}
	out := make([]map[string]any, 0, len(items))
	for _, item := range items {
		out = append(out, notificationPayload(item))
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"notifications": out,
		"next_cursor":   next,
	})
}

// ---- GET /api/notifications/unread-count ------------------------------------

// unreadCount is the cheap endpoint the bell badge polls (~30s).
func (n *notificationsAPI) unreadCount(w http.ResponseWriter, r *http.Request) {
	count, err := n.repo.CountUnreadNotifications(r.Context())
	if err != nil {
		n.logger.Error("count unread failed", "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "falha ao contar não lidas"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"unread": count})
}

// ---- POST /api/notifications/{id}/read --------------------------------------

func (n *notificationsAPI) markRead(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		writeAPIError(w, http.StatusBadRequest, "invalid_id", "invalid id", nil)
		return
	}
	if err := n.repo.MarkNotificationRead(r.Context(), id); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeAPIError(w, http.StatusNotFound, "notification.not_found", "notification not found", map[string]any{"id": id})
			return
		}
		n.logger.Error("mark notification read failed", "id", id, "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "falha ao marcar como lida"})
		return
	}
	notif, err := n.repo.GetNotification(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]any{"id": id, "read": true})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"id": id, "read": true, "notification": notificationPayload(notif)})
}

// ---- POST /api/notifications/read-all ----------------------------------------

func (n *notificationsAPI) readAll(w http.ResponseWriter, r *http.Request) {
	affected, err := n.repo.MarkAllNotificationsRead(r.Context())
	if err != nil {
		n.logger.Error("mark all read failed", "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "falha ao marcar todas como lidas"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"updated": affected})
}

// ---- DELETE /api/notifications/{id} ------------------------------------------

func (n *notificationsAPI) delete(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		writeAPIError(w, http.StatusBadRequest, "invalid_id", "invalid id", nil)
		return
	}
	if err := n.repo.DeleteNotification(r.Context(), id); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeAPIError(w, http.StatusNotFound, "notification.not_found", "notification not found", map[string]any{"id": id})
			return
		}
		n.logger.Error("delete notification failed", "id", id, "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "falha ao remover notificação"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"id": id, "deleted": true})
}

// ---- DELETE /api/notifications ("limpar lidas") -------------------------------

func (n *notificationsAPI) clearRead(w http.ResponseWriter, r *http.Request) {
	deleted, err := n.repo.DeleteReadNotifications(r.Context())
	if err != nil {
		n.logger.Error("clear read notifications failed", "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "falha ao limpar notificações lidas"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"deleted": deleted})
}

// ---- GET/PUT /api/settings/notifications --------------------------------------

type notificationSettingsRequest struct {
	InAppEnabled         *bool `json:"in_app_enabled"`
	RetentionDays        *int  `json:"retention_days"`
	BrowserNotifications *bool `json:"browser_notifications"`
}

func (n *notificationsAPI) getConfig(w http.ResponseWriter, r *http.Request) {
	cfg, err := notifier.LoadConfig(r.Context(), n.repo)
	if err != nil {
		n.logger.Error("load notification config failed", "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "falha ao carregar configuração"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"in_app_enabled":        cfg.InAppEnabled,
		"retention_days":        cfg.RetentionDays,
		"browser_notifications": cfg.BrowserNotifications,
	})
}

func (n *notificationsAPI) updateConfig(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxSettingsBody)
	var req notificationSettingsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_json", "invalid JSON", nil)
		return
	}

	cfg, err := notifier.LoadConfig(r.Context(), n.repo)
	if err != nil {
		n.logger.Error("load notification config failed", "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "falha ao carregar configuração"})
		return
	}
	if req.InAppEnabled != nil {
		cfg.InAppEnabled = *req.InAppEnabled
	}
	if req.BrowserNotifications != nil {
		cfg.BrowserNotifications = *req.BrowserNotifications
	}
	if req.RetentionDays != nil {
		cfg.RetentionDays = *req.RetentionDays
	}
	if err := notifier.SaveConfig(r.Context(), n.repo, cfg); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"in_app_enabled":        cfg.InAppEnabled,
		"retention_days":        cfg.RetentionDays,
		"browser_notifications": cfg.BrowserNotifications,
	})
}
