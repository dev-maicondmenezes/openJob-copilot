package tasks

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/maicondmenezes/openjob/internal/email"
	"github.com/maicondmenezes/openjob/internal/email/connector"
	"github.com/maicondmenezes/openjob/internal/events"
	"github.com/maicondmenezes/openjob/internal/llm"
	"github.com/maicondmenezes/openjob/internal/store"
)

// ConnectorFactory creates an email.Connector from account settings.
type ConnectorFactory func(provider string, authConfig []byte, masterKey []byte) (email.Connector, error)

// DefaultConnectorFactory creates Gmail or IMAP connectors with decrypted credentials.
func DefaultConnectorFactory(provider string, authConfig []byte, masterKey []byte) (email.Connector, error) {
	switch provider {
	case "gmail":
		var cfg email.GmailAuthConfig
		if len(authConfig) > 0 {
			if err := email.DecryptAuthConfig(authConfig, masterKey, &cfg); err != nil {
				return nil, fmt.Errorf("decrypt gmail config: %w", err)
			}
		}
		return connector.NewGmailConnector(cfg, nil), nil
	case "imap":
		var cfg email.IMAPAuthConfig
		if len(authConfig) > 0 {
			if err := email.DecryptAuthConfig(authConfig, masterKey, &cfg); err != nil {
				return nil, fmt.Errorf("decrypt imap config: %w", err)
			}
		}
		return connector.NewIMAPConnector(cfg), nil
	default:
		return nil, fmt.Errorf("unsupported email provider: %s", provider)
	}
}

// SyncEmailHandler coordinates email inbox polling, parsing, and pipeline updates.
type SyncEmailHandler struct {
	repo             store.Repository
	masterKey        []byte
	llmClient        llm.Client
	bus              *events.Bus
	logger           *slog.Logger
	connectorFactory ConnectorFactory
}

// NewSyncEmailHandler creates a handler for TypeSyncEmail.
func NewSyncEmailHandler(
	repo store.Repository,
	masterKey []byte,
	llmClient llm.Client,
	bus *events.Bus,
	logger *slog.Logger,
	connectorFactory ConnectorFactory,
) *SyncEmailHandler {
	if logger == nil {
		logger = slog.Default()
	}
	if connectorFactory == nil {
		connectorFactory = DefaultConnectorFactory
	}
	return &SyncEmailHandler{
		repo:             repo,
		masterKey:        masterKey,
		llmClient:        llmClient,
		bus:              bus,
		logger:           logger,
		connectorFactory: connectorFactory,
	}
}

func (h *SyncEmailHandler) Type() string { return TypeSyncEmail }

func (h *SyncEmailHandler) Run(ctx context.Context, payload json.RawMessage, progress func(int)) (json.RawMessage, error) {
	progress(10)

	var p struct {
		AccountID string `json:"account_id"`
	}
	if len(payload) > 0 {
		_ = json.Unmarshal(payload, &p)
	}

	var accounts []store.EmailAccount
	if p.AccountID != "" {
		acc, err := h.repo.GetEmailAccount(ctx, p.AccountID)
		if err != nil {
			return nil, fmt.Errorf("get email account %s: %w", p.AccountID, err)
		}
		if acc != nil {
			accounts = append(accounts, *acc)
		}
	} else {
		all, err := h.repo.ListEmailAccounts(ctx)
		if err != nil {
			return nil, fmt.Errorf("list email accounts: %w", err)
		}
		for _, a := range all {
			if a.Enabled {
				accounts = append(accounts, a)
			}
		}
	}

	progress(25)

	var llmCfg *llm.Config
	if h.llmClient != nil {
		if cfg, err := h.resolveLLMConfig(ctx); err == nil && cfg.APIKey != "" {
			llmCfg = &cfg
		}
	}

	engine := email.NewEngine(h.repo, h.bus, h.llmClient, llmCfg, h.logger)

	var totalStats email.SyncStats
	stepProgress := 25
	stepIncrement := 0
	if len(accounts) > 0 {
		stepIncrement = 70 / len(accounts)
	}

	for _, acc := range accounts {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		conn, err := h.connectorFactory(acc.Provider, acc.AuthConfig, h.masterKey)
		if err != nil {
			h.logger.Error("sync_email: connector factory error", "account", acc.ID, "err", err)
			_ = h.repo.UpdateEmailAccountCursor(ctx, acc.ID, acc.LastUID, acc.LastHistoryID, time.Now().UTC(), err.Error())
			continue
		}

		cursor := acc.LastUID
		if acc.Provider == "gmail" {
			cursor = acc.LastHistoryID
		}

		rawMsgs, newCursor, err := conn.FetchNewMessages(ctx, cursor, acc.Folder, 50)
		if err != nil {
			h.logger.Error("sync_email: fetch messages failed", "account", acc.ID, "err", err)
			_ = h.repo.UpdateEmailAccountCursor(ctx, acc.ID, acc.LastUID, acc.LastHistoryID, time.Now().UTC(), err.Error())
			continue
		}

		stats, err := engine.ProcessRawMessages(ctx, acc.ID, rawMsgs)
		if err != nil {
			h.logger.Error("sync_email: process messages failed", "account", acc.ID, "err", err)
			_ = h.repo.UpdateEmailAccountCursor(ctx, acc.ID, acc.LastUID, acc.LastHistoryID, time.Now().UTC(), err.Error())
			continue
		}

		lastUID := acc.LastUID
		lastHistID := acc.LastHistoryID
		if acc.Provider == "gmail" {
			lastHistID = newCursor
		} else {
			lastUID = newCursor
		}

		if err := h.repo.UpdateEmailAccountCursor(ctx, acc.ID, lastUID, lastHistID, time.Now().UTC(), ""); err != nil {
			h.logger.Warn("sync_email: update cursor failed", "account", acc.ID, "err", err)
		}

		totalStats.EmailsFetched += stats.EmailsFetched
		totalStats.Matched += stats.Matched
		totalStats.Advanced += stats.Advanced
		totalStats.Suggested += stats.Suggested

		stepProgress += stepIncrement
		progress(stepProgress)
	}

	progress(100)
	return json.Marshal(map[string]any{
		"emails_fetched": totalStats.EmailsFetched,
		"matched":        totalStats.Matched,
		"advanced":       totalStats.Advanced,
		"suggested":      totalStats.Suggested,
		"accounts_run":   len(accounts),
		"synced_at":      time.Now().UTC().Format(time.RFC3339),
	})
}

func (h *SyncEmailHandler) resolveLLMConfig(ctx context.Context) (llm.Config, error) {
	sealedKey, err := h.repo.GetSetting(ctx, "llm.api_key")
	if err != nil || sealedKey == "" {
		return llm.Config{}, errors.New("not configured")
	}
	apiKey, err := store.DecryptString(h.masterKey, sealedKey)
	if err != nil || apiKey == "" {
		return llm.Config{}, errors.New("cannot decrypt key")
	}
	baseURL, _ := h.repo.GetSetting(ctx, "llm.base_url")
	model, _ := h.repo.GetSetting(ctx, "llm.model")
	return llm.Config{BaseURL: baseURL, Model: model, APIKey: apiKey}.Normalize(), nil
}
