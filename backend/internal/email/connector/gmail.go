package connector

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/maicondmenezes/openjob/internal/email"
)

// GmailConnector interacts with the Gmail REST API (read-only).
type GmailConnector struct {
	cfg        email.GmailAuthConfig
	httpClient *http.Client
}

// NewGmailConnector creates a new Gmail REST API connector.
func NewGmailConnector(cfg email.GmailAuthConfig, httpClient *http.Client) *GmailConnector {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}
	return &GmailConnector{
		cfg:        cfg,
		httpClient: httpClient,
	}
}

func (c *GmailConnector) ensureAccessToken(ctx context.Context) (string, error) {
	if c.cfg.AccessToken != "" && time.Now().Before(c.cfg.Expiry.Add(-60*time.Second)) {
		return c.cfg.AccessToken, nil
	}

	if c.cfg.RefreshToken == "" {
		if c.cfg.AccessToken != "" {
			return c.cfg.AccessToken, nil
		}
		return "", fmt.Errorf("gmail: missing refresh token and access token")
	}

	data := url.Values{
		"client_id":     {c.cfg.ClientID},
		"client_secret": {c.cfg.ClientSecret},
		"refresh_token": {c.cfg.RefreshToken},
		"grant_type":    {"refresh_token"},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://oauth2.googleapis.com/token", strings.NewReader(data.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("gmail: refresh token request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return "", fmt.Errorf("gmail: refresh token failed (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var tokenResp struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return "", fmt.Errorf("gmail: decode token response: %w", err)
	}

	c.cfg.AccessToken = tokenResp.AccessToken
	c.cfg.Expiry = time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)
	return c.cfg.AccessToken, nil
}

func (c *GmailConnector) TestConnection(ctx context.Context) error {
	token, err := c.ensureAccessToken(ctx)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://gmail.googleapis.com/gmail/v1/users/me/profile", nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("gmail: test connection request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return fmt.Errorf("gmail: API returned HTTP %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

func (c *GmailConnector) FetchNewMessages(ctx context.Context, cursor string, folder string, limit int) ([]email.RawMessage, string, error) {
	token, err := c.ensureAccessToken(ctx)
	if err != nil {
		return nil, cursor, err
	}

	if limit <= 0 {
		limit = 50
	}

	// First, check profile to get current history ID
	profileReq, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://gmail.googleapis.com/gmail/v1/users/me/profile", nil)
	if err != nil {
		return nil, cursor, err
	}
	profileReq.Header.Set("Authorization", "Bearer "+token)

	profileResp, err := c.httpClient.Do(profileReq)
	if err != nil {
		return nil, cursor, fmt.Errorf("gmail: profile check failed: %w", err)
	}
	defer profileResp.Body.Close()

	var profile struct {
		HistoryID string `json:"historyId"`
	}
	_ = json.NewDecoder(profileResp.Body).Decode(&profile)

	var messageIDs []string

	if cursor != "" {
		// Incremental sync using users.history.list
		historyURL := fmt.Sprintf("https://gmail.googleapis.com/gmail/v1/users/me/history?startHistoryId=%s&historyTypes=messageAdded&maxResults=%d", url.QueryEscape(cursor), limit)
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, historyURL, nil)
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := c.httpClient.Do(req)
		if err == nil && resp.StatusCode == http.StatusOK {
			defer resp.Body.Close()
			var hist struct {
				History []struct {
					MessagesAdded []struct {
						Message struct {
							ID string `json:"id"`
						} `json:"message"`
					} `json:"messagesAdded"`
				} `json:"history"`
				HistoryID string `json:"historyId"`
			}
			if err := json.NewDecoder(resp.Body).Decode(&hist); err == nil {
				seen := make(map[string]bool)
				for _, h := range hist.History {
					for _, ma := range h.MessagesAdded {
						if ma.Message.ID != "" && !seen[ma.Message.ID] {
							seen[ma.Message.ID] = true
							messageIDs = append(messageIDs, ma.Message.ID)
						}
					}
				}
				if hist.HistoryID != "" {
					cursor = hist.HistoryID
				}
			}
		} else if resp != nil && resp.StatusCode == http.StatusNotFound {
			// History ID is too old / invalid: fall back to recent message listing
			cursor = ""
		}
	}

	if cursor == "" || len(messageIDs) == 0 {
		// Full/initial sync: list recent messages
		listURL := fmt.Sprintf("https://gmail.googleapis.com/gmail/v1/users/me/messages?maxResults=%d", limit)
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, listURL, nil)
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return nil, cursor, fmt.Errorf("gmail: list messages failed: %w", err)
		}
		defer resp.Body.Close()

		var listResp struct {
			Messages []struct {
				ID string `json:"id"`
			} `json:"messages"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&listResp); err != nil {
			return nil, cursor, fmt.Errorf("gmail: decode list: %w", err)
		}

		for _, m := range listResp.Messages {
			messageIDs = append(messageIDs, m.ID)
		}
	}

	var rawMessages []email.RawMessage
	for _, id := range messageIDs {
		msgURL := fmt.Sprintf("https://gmail.googleapis.com/gmail/v1/users/me/messages/%s?format=raw", url.PathEscape(id))
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, msgURL, nil)
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := c.httpClient.Do(req)
		if err != nil || resp.StatusCode != http.StatusOK {
			continue
		}

		var rawData struct {
			Raw string `json:"raw"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&rawData); err != nil {
			resp.Body.Close()
			continue
		}
		resp.Body.Close()

		decoded, err := base64.URLEncoding.DecodeString(rawData.Raw)
		if err != nil {
			continue
		}

		parsed, err := ParseMIME(bytes.NewReader(decoded))
		if err != nil {
			continue
		}
		rawMessages = append(rawMessages, *parsed)
	}

	newCursor := cursor
	if profile.HistoryID != "" {
		newCursor = profile.HistoryID
	}

	return rawMessages, newCursor, nil
}
