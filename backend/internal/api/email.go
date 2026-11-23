package api

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/maicondmenezes/openjob/internal/email"
	"github.com/maicondmenezes/openjob/internal/email/connector"
	"github.com/maicondmenezes/openjob/internal/events"
	"github.com/maicondmenezes/openjob/internal/llm"
	"github.com/maicondmenezes/openjob/internal/store"
	"github.com/maicondmenezes/openjob/internal/tasks"
)

type emailAPI struct {
	repo      store.Repository
	masterKey []byte
	llm       llm.Client
	logger    *slog.Logger
	pool      *tasks.Pool
	bus       *events.Bus
	scheduler *tasks.EmailSyncScheduler
}

const (
	settingEmailSyncInterval = "email.sync_interval"
	defaultSyncInterval      = "5m"
)

func registerEmailAPI(r chi.Router, e *emailAPI) {
	// Settings
	r.Get("/settings/email", e.getEmailSettings)
	r.Put("/settings/email", e.saveEmailSettings)
	r.Post("/settings/email/test", e.testEmailConnection)
	r.Post("/settings/email/sync", e.triggerEmailSync)

	// Messages & linking
	r.Get("/email/messages", e.listEmailMessages)
	r.Post("/email/messages/{id}/link", e.linkEmailMessage)
	r.Post("/email/messages/{id}/unlink", e.unlinkEmailMessage)

	// Tags
	r.Get("/tags", e.listTags)
	r.Post("/tags", e.createTag)
}

// ---- GET /api/settings/email ----------------------------------------------

func (e *emailAPI) getEmailSettings(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	accounts, err := e.repo.ListEmailAccounts(ctx)
	if err != nil {
		e.logger.Error("list email accounts failed", "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "falha ao consultar contas de e-mail"})
		return
	}

	interval, _ := e.repo.GetSetting(ctx, settingEmailSyncInterval)
	if interval == "" {
		interval = defaultSyncInterval
	}

	var accountData map[string]any
	if len(accounts) > 0 {
		acc := accounts[0]
		accountData = map[string]any{
			"id":              acc.ID,
			"provider":        acc.Provider,
			"address":         acc.Address,
			"folder":          acc.Folder,
			"enabled":         acc.Enabled,
			"has_auth_config": len(acc.AuthConfig) > 0,
			"last_sync_at":    acc.LastSyncAt,
			"last_error":      acc.LastError,
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"account":  accountData,
		"interval": interval,
	})
}

// ---- PUT /api/settings/email ----------------------------------------------

func (e *emailAPI) saveEmailSettings(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxSettingsBody)
	var req struct {
		Provider string          `json:"provider"`
		Address  string          `json:"address"`
		Folder   string          `json:"folder"`
		Enabled  bool            `json:"enabled"`
		Interval string          `json:"interval"`
		Auth     json.RawMessage `json:"auth"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_json", "invalid JSON payload", nil)
		return
	}

	req.Provider = strings.ToLower(strings.TrimSpace(req.Provider))
	req.Address = strings.TrimSpace(req.Address)
	if req.Provider != "gmail" && req.Provider != "imap" {
		writeAPIError(w, http.StatusBadRequest, "email.invalid_provider", "provedor inválido (suportados: gmail, imap)", nil)
		return
	}
	if req.Address == "" {
		writeAPIError(w, http.StatusBadRequest, "email.address_required", "endereço de e-mail obrigatório", nil)
		return
	}
	if req.Folder == "" {
		req.Folder = "INBOX"
	}

	ctx := r.Context()
	existing, _ := e.repo.GetEmailAccount(ctx, req.Provider)

	var encryptedAuth []byte
	if len(req.Auth) > 0 && string(req.Auth) != "null" && string(req.Auth) != "{}" {
		var err error
		if req.Provider == "gmail" {
			var gAuth email.GmailAuthConfig
			if err := json.Unmarshal(req.Auth, &gAuth); err != nil {
				writeAPIError(w, http.StatusBadRequest, "email.invalid_auth", "dados de autenticação do Gmail inválidos", nil)
				return
			}
			encryptedAuth, err = email.EncryptAuthConfig(gAuth, e.masterKey)
		} else {
			var iAuth email.IMAPAuthConfig
			if err := json.Unmarshal(req.Auth, &iAuth); err != nil {
				writeAPIError(w, http.StatusBadRequest, "email.invalid_auth", "dados de autenticação do IMAP inválidos", nil)
				return
			}
			encryptedAuth, err = email.EncryptAuthConfig(iAuth, e.masterKey)
		}
		if err != nil {
			e.logger.Error("encrypt auth config failed", "err", err)
			writeAPIError(w, http.StatusInternalServerError, "email.encrypt_failed", "falha ao cifrar credenciais", nil)
			return
		}
	} else if existing != nil {
		encryptedAuth = existing.AuthConfig
	}

	account := store.EmailAccount{
		ID:         req.Provider,
		Provider:   req.Provider,
		Address:    req.Address,
		AuthConfig: encryptedAuth,
		Folder:     req.Folder,
		Enabled:    req.Enabled,
	}
	if existing != nil {
		account.LastUID = existing.LastUID
		account.LastHistoryID = existing.LastHistoryID
		account.LastSyncAt = existing.LastSyncAt
	}

	if err := e.repo.SaveEmailAccount(ctx, account); err != nil {
		e.logger.Error("save email account failed", "err", err)
		writeAPIError(w, http.StatusInternalServerError, "email.save_failed", "falha ao salvar conta de e-mail", nil)
		return
	}

	if req.Interval != "" {
		_ = e.repo.SetSetting(ctx, settingEmailSyncInterval, req.Interval)
		if e.scheduler != nil {
			e.scheduler.Configure(req.Interval)
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// ---- POST /api/settings/email/test ----------------------------------------

func (e *emailAPI) testEmailConnection(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxSettingsBody)
	var req struct {
		Provider string          `json:"provider"`
		Auth     json.RawMessage `json:"auth"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && err.Error() != "EOF" {
		writeAPIError(w, http.StatusBadRequest, "invalid_json", "invalid JSON", nil)
		return
	}

	ctx := r.Context()
	provider := strings.ToLower(strings.TrimSpace(req.Provider))

	var conn email.Connector
	if len(req.Auth) > 0 && string(req.Auth) != "null" && string(req.Auth) != "{}" {
		if provider == "gmail" {
			var gAuth email.GmailAuthConfig
			if err := json.Unmarshal(req.Auth, &gAuth); err != nil {
				writeAPIError(w, http.StatusBadRequest, "email.invalid_auth", "configuração Gmail inválida", nil)
				return
			}
			conn = connector.NewGmailConnector(gAuth, nil)
		} else if provider == "imap" {
			var iAuth email.IMAPAuthConfig
			if err := json.Unmarshal(req.Auth, &iAuth); err != nil {
				writeAPIError(w, http.StatusBadRequest, "email.invalid_auth", "configuração IMAP inválida", nil)
				return
			}
			conn = connector.NewIMAPConnector(iAuth)
		} else {
			writeAPIError(w, http.StatusBadRequest, "email.invalid_provider", "provedor inválido", nil)
			return
		}
	} else {
		// Use stored account
		accounts, err := e.repo.ListEmailAccounts(ctx)
		if err != nil || len(accounts) == 0 {
			writeJSON(w, http.StatusOK, map[string]any{
				"ok":    false,
				"error": "Nenhuma conta de e-mail configurada ainda.",
			})
			return
		}
		acc := accounts[0]
		if provider != "" {
			for _, a := range accounts {
				if a.Provider == provider {
					acc = a
					break
				}
			}
		}

		if acc.Provider == "gmail" {
			var gAuth email.GmailAuthConfig
			if err := email.DecryptAuthConfig(acc.AuthConfig, e.masterKey, &gAuth); err != nil {
				fe := llm.ClassifyError(err, "")
				writeJSON(w, http.StatusOK, map[string]any{"ok": false, "error": fe.Message, "code": fe.Code})
				return
			}
			conn = connector.NewGmailConnector(gAuth, nil)
		} else {
			var iAuth email.IMAPAuthConfig
			if err := email.DecryptAuthConfig(acc.AuthConfig, e.masterKey, &iAuth); err != nil {
				fe := llm.ClassifyError(err, "")
				writeJSON(w, http.StatusOK, map[string]any{"ok": false, "error": fe.Message, "code": fe.Code})
				return
			}
			conn = connector.NewIMAPConnector(iAuth)
		}
	}

	testCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	if err := conn.TestConnection(testCtx); err != nil {
		e.logger.Info("email test connection failed", "err", err)
		fe := llm.ClassifyError(err, "")
		writeJSON(w, http.StatusOK, map[string]any{
			"ok":         false,
			"error":      fe.Message,
			"code":       fe.Code,
			"suggestion": fe.Suggestion,
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"ok":      true,
		"message": "Conexão estabelecida com sucesso.",
	})
}

// ---- POST /api/settings/email/sync ----------------------------------------

func (e *emailAPI) triggerEmailSync(w http.ResponseWriter, r *http.Request) {
	if e.pool == nil {
		writeAPIError(w, http.StatusServiceUnavailable, "tasks.pool_unavailable", "fila de tarefas indisponível", nil)
		return
	}

	task, err := e.pool.Enqueue(r.Context(), tasks.TypeSyncEmail, json.RawMessage("{}"), 5)
	if err != nil {
		e.logger.Error("enqueue sync_email failed", "err", err)
		writeAPIError(w, http.StatusInternalServerError, "tasks.enqueue_failed", "falha ao enfileirar sincronização", nil)
		return
	}

	writeJSON(w, http.StatusAccepted, map[string]any{
		"task_id": task.ID,
		"status":  task.Status,
	})
}

// ---- GET /api/email/messages ----------------------------------------------

func (e *emailAPI) listEmailMessages(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	appIDStr := r.URL.Query().Get("application_id")
	suggestedOnly := r.URL.Query().Get("suggested") == "true"

	var messages []store.EmailMessage
	var err error

	if appIDStr != "" {
		appID, parseErr := strconv.ParseInt(appIDStr, 10, 64)
		if parseErr != nil {
			writeAPIError(w, http.StatusBadRequest, "invalid_id", "ID de candidatura inválido", nil)
			return
		}
		messages, err = e.repo.ListEmailMessagesByApplication(ctx, appID)
	} else if suggestedOnly {
		messages, err = e.repo.ListSuggestedEmailMessages(ctx)
	} else {
		messages, err = e.repo.ListSuggestedEmailMessages(ctx)
	}

	if err != nil {
		e.logger.Error("list email messages failed", "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "falha ao listar mensagens de e-mail"})
		return
	}

	items := make([]map[string]any, 0, len(messages))
	for _, m := range messages {
		items = append(items, map[string]any{
			"id":                     m.ID,
			"account_id":             m.AccountID,
			"message_id":             m.MessageID,
			"thread_id":              m.ThreadID,
			"from_address":           m.FromAddress,
			"from_name":              m.FromName,
			"subject":                m.Subject,
			"received_at":            m.ReceivedAt,
			"matched_application_id": m.MatchedApplicationID,
			"match_confidence":       m.MatchConfidence,
			"stage_signal":           m.StageSignal,
			"classified_at":          m.ClassifiedAt,
			"link_source":            m.LinkSource,
		})
	}

	writeJSON(w, http.StatusOK, map[string]any{"messages": items})
}

// ---- POST /api/email/messages/{id}/link -----------------------------------

func (e *emailAPI) linkEmailMessage(w http.ResponseWriter, r *http.Request) {
	msgID := chi.URLParam(r, "id")
	if msgID == "" {
		writeAPIError(w, http.StatusBadRequest, "invalid_id", "ID de mensagem obrigatório", nil)
		return
	}

	var req struct {
		ApplicationID int64 `json:"application_id"`
		AdvanceStage  bool  `json:"advance_stage"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.ApplicationID <= 0 {
		writeAPIError(w, http.StatusBadRequest, "invalid_json", "application_id obrigatório", nil)
		return
	}

	ctx := r.Context()
	if err := e.repo.LinkEmailMessage(ctx, msgID, req.ApplicationID, "manual"); err != nil {
		e.logger.Error("link email message failed", "msg_id", msgID, "app_id", req.ApplicationID, "err", err)
		writeAPIError(w, http.StatusInternalServerError, "email.link_failed", "falha ao vincular e-mail", nil)
		return
	}

	if req.AdvanceStage {
		msg, err := e.repo.GetEmailMessage(ctx, msgID)
		if err == nil && msg != nil && msg.StageSignal != "" {
			stageGroup, stageLabel := mapSignalToStage(msg.StageSignal)
			if stageGroup != "" {
				_ = e.repo.UpdateApplicationStage(ctx, req.ApplicationID, stageGroup, stageLabel, "email_suggested")
			}
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// ---- POST /api/email/messages/{id}/unlink ---------------------------------

func (e *emailAPI) unlinkEmailMessage(w http.ResponseWriter, r *http.Request) {
	msgID := chi.URLParam(r, "id")
	if msgID == "" {
		writeAPIError(w, http.StatusBadRequest, "invalid_id", "ID de mensagem obrigatório", nil)
		return
	}

	if err := e.repo.UnlinkEmailMessage(r.Context(), msgID); err != nil {
		e.logger.Error("unlink email message failed", "msg_id", msgID, "err", err)
		writeAPIError(w, http.StatusInternalServerError, "email.unlink_failed", "falha ao desvincular e-mail", nil)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// ---- GET /api/tags --------------------------------------------------------

func (e *emailAPI) listTags(w http.ResponseWriter, r *http.Request) {
	namespace := r.URL.Query().Get("namespace")
	tagsList, err := e.repo.ListTags(r.Context(), namespace)
	if err != nil {
		e.logger.Error("list tags failed", "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "falha ao listar tags"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"tags": tagsList})
}

// ---- POST /api/tags -------------------------------------------------------

func (e *emailAPI) createTag(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Namespace string `json:"namespace"`
		Value     string `json:"value"`
		Label     string `json:"label"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_json", "JSON inválido", nil)
		return
	}

	req.Namespace = strings.TrimSpace(req.Namespace)
	req.Value = strings.TrimSpace(req.Value)
	req.Label = strings.TrimSpace(req.Label)

	if req.Namespace == "" || req.Value == "" {
		writeAPIError(w, http.StatusBadRequest, "tag.invalid", "namespace e value obrigatórios", nil)
		return
	}
	if req.Label == "" {
		req.Label = req.Value
	}

	tag, err := e.repo.GetOrCreateTag(r.Context(), req.Namespace, req.Value, req.Label)
	if err != nil {
		e.logger.Error("get or create tag failed", "err", err)
		writeAPIError(w, http.StatusInternalServerError, "tag.create_failed", "falha ao criar tag", nil)
		return
	}

	writeJSON(w, http.StatusOK, tag)
}

func mapSignalToStage(signal string) (string, string) {
	switch signal {
	case "ack":
		return "in_process", "Recebida"
	case "screening":
		return "in_process", "Triagem"
	case "assessment":
		return "in_process", "Desafio Técnico"
	case "info_request":
		return "in_process", "Dados Adicionais"
	case "interview":
		return "interview", "Entrevista"
	case "offer":
		return "offer", "Proposta"
	case "rejection":
		return "rejected", "Rejeitada"
	default:
		return "", ""
	}
}
