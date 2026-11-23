package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/maicondmenezes/openjob/internal/config"
	"github.com/maicondmenezes/openjob/internal/llm"
	"github.com/maicondmenezes/openjob/internal/store"
)

func setupTestServer(t *testing.T) (http.Handler, *Runtime, store.Repository, []byte) {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "openjob.db")
	repo, err := store.Open(dbPath)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	masterKey := []byte("01234567890123456789012345678901")

	cfg := config.Config{
		Port:     "0",
		DBPath:   dbPath,
		Hostname: "localhost",
		Features: config.Features{
			EmailSync: true,
		},
	}

	handler, runtime := New(cfg, nil, repo, masterKey, nil)
	return handler, runtime, repo, masterKey
}

func TestEmailSettingsAndSyncAPI(t *testing.T) {
	handler, runtime, repo, _ := setupTestServer(t)
	defer runtime.Stop()
	defer repo.Close()

	srv := httptest.NewServer(handler)
	defer srv.Close()

	// 1. Initial GET /api/settings/email should return empty account
	res, err := http.Get(srv.URL + "/api/settings/email")
	if err != nil {
		t.Fatalf("GET /api/settings/email: %v", err)
	}
	if res.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200", res.StatusCode)
	}
	var getResp struct {
		Account  any    `json:"account"`
		Interval string `json:"interval"`
	}
	_ = json.NewDecoder(res.Body).Decode(&getResp)
	res.Body.Close()

	// 2. PUT /api/settings/email to configure IMAP account
	putPayload := map[string]any{
		"provider": "imap",
		"address":  "candidate@example.com",
		"folder":   "INBOX",
		"enabled":  true,
		"interval": "10m",
		"auth": map[string]any{
			"host":     "imap.example.com",
			"port":     993,
			"username": "candidate@example.com",
			"password": "secret_password",
			"use_tls":  true,
		},
	}
	body, _ := json.Marshal(putPayload)
	req, _ := http.NewRequest(http.MethodPut, srv.URL+"/api/settings/email", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	res, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("PUT /api/settings/email: %v", err)
	}
	if res.StatusCode != http.StatusOK {
		t.Fatalf("PUT status = %d, want 200", res.StatusCode)
	}
	res.Body.Close()

	// Verify account saved
	res, err = http.Get(srv.URL + "/api/settings/email")
	if err != nil {
		t.Fatalf("GET /api/settings/email after PUT: %v", err)
	}
	var afterPut struct {
		Account struct {
			ID            string `json:"id"`
			Provider      string `json:"provider"`
			Address       string `json:"address"`
			Enabled       bool   `json:"enabled"`
			HasAuthConfig bool   `json:"has_auth_config"`
		} `json:"account"`
		Interval string `json:"interval"`
	}
	_ = json.NewDecoder(res.Body).Decode(&afterPut)
	res.Body.Close()

	if afterPut.Account.Address != "candidate@example.com" || !afterPut.Account.HasAuthConfig {
		t.Errorf("unexpected account after PUT: %+v", afterPut)
	}
	if afterPut.Interval != "10m" {
		t.Errorf("interval = %q, want 10m", afterPut.Interval)
	}

	// 3. POST /api/settings/email/sync triggers task
	res, err = http.Post(srv.URL+"/api/settings/email/sync", "application/json", nil)
	if err != nil {
		t.Fatalf("POST /api/settings/email/sync: %v", err)
	}
	if res.StatusCode != http.StatusAccepted {
		t.Errorf("sync status = %d, want 202 Accepted", res.StatusCode)
	}
	var syncResp struct {
		TaskID string `json:"task_id"`
		Status string `json:"status"`
	}
	_ = json.NewDecoder(res.Body).Decode(&syncResp)
	res.Body.Close()

	if syncResp.TaskID == "" {
		t.Errorf("expected non-empty task_id")
	}
}

func TestEmailMessagesAndLinkingAPI(t *testing.T) {
	handler, runtime, repo, _ := setupTestServer(t)
	defer runtime.Stop()
	defer repo.Close()

	srv := httptest.NewServer(handler)
	defer srv.Close()

	ctx := context.Background()

	// Create job and application
	jobID, _ := repo.CreateJob(ctx, store.Job{
		Title:   "Go Engineer",
		Company: "Acme",
		URL:     "https://acme.com",
	})
	appID, _ := repo.CreateApplication(ctx, store.Application{
		JobID:  jobID,
		Status: "pending",
	})

	_ = repo.SaveEmailAccount(ctx, store.EmailAccount{
		ID:         "imap",
		Provider:   "imap",
		Address:    "user@example.com",
		AuthConfig: []byte("enc"),
		Folder:     "INBOX",
		Enabled:    true,
	})

	now := time.Now().UTC()
	msg := store.EmailMessage{
		ID:                   "msg_test_123",
		AccountID:            "imap",
		MessageID:            "<test-123@greenhouse.io>",
		Subject:              "Interview Invitation at Acme",
		ReceivedAt:           now,
		MatchedApplicationID: &appID,
		MatchConfidence:      0.65, // low confidence -> suggested
		StageSignal:          "interview",
		ClassifiedAt:         &now,
		LinkSource:           "suggested",
	}
	if err := repo.SaveEmailMessage(ctx, msg); err != nil {
		t.Fatalf("SaveEmailMessage: %v", err)
	}

	// 1. GET /api/email/messages?suggested=true
	res, err := http.Get(srv.URL + "/api/email/messages?suggested=true")
	if err != nil {
		t.Fatalf("GET /api/email/messages: %v", err)
	}
	var listResp struct {
		Messages []struct {
			ID         string `json:"id"`
			Subject    string `json:"subject"`
			LinkSource string `json:"link_source"`
		} `json:"messages"`
	}
	_ = json.NewDecoder(res.Body).Decode(&listResp)
	res.Body.Close()

	if len(listResp.Messages) != 1 || listResp.Messages[0].ID != "msg_test_123" {
		t.Fatalf("unexpected suggested messages: %+v", listResp.Messages)
	}

	// 2. POST /api/email/messages/msg_test_123/link with advance_stage = true
	linkBody, _ := json.Marshal(map[string]any{
		"application_id": appID,
		"advance_stage":  true,
	})
	res, err = http.Post(srv.URL+"/api/email/messages/msg_test_123/link", "application/json", bytes.NewReader(linkBody))
	if err != nil {
		t.Fatalf("link request: %v", err)
	}
	if res.StatusCode != http.StatusOK {
		t.Errorf("link status = %d, want 200", res.StatusCode)
	}
	res.Body.Close()

	// Verify application updated to interview
	app, _ := repo.GetApplication(ctx, appID)
	if app.StageGroup != "interview" {
		t.Errorf("StageGroup = %q, want interview", app.StageGroup)
	}

	// 3. POST /api/email/messages/msg_test_123/unlink
	res, err = http.Post(srv.URL+"/api/email/messages/msg_test_123/unlink", "application/json", nil)
	if err != nil {
		t.Fatalf("unlink request: %v", err)
	}
	if res.StatusCode != http.StatusOK {
		t.Errorf("unlink status = %d, want 200", res.StatusCode)
	}
	res.Body.Close()

	linkedMsgs, _ := repo.ListEmailMessagesByApplication(ctx, appID)
	if len(linkedMsgs) != 0 {
		t.Errorf("expected 0 linked messages after unlink, got %d", len(linkedMsgs))
	}
}

func TestTagsAndApplicationStageAPI(t *testing.T) {
	handler, runtime, repo, _ := setupTestServer(t)
	defer runtime.Stop()
	defer repo.Close()

	srv := httptest.NewServer(handler)
	defer srv.Close()

	ctx := context.Background()

	// 1. POST /api/tags
	tagPayload, _ := json.Marshal(map[string]any{
		"namespace": "phase",
		"value":     "live_coding",
		"label":     "Live Coding",
	})
	res, err := http.Post(srv.URL+"/api/tags", "application/json", bytes.NewReader(tagPayload))
	if err != nil {
		t.Fatalf("POST /api/tags: %v", err)
	}
	if res.StatusCode != http.StatusOK {
		t.Fatalf("POST /api/tags status = %d, want 200", res.StatusCode)
	}
	var tagResp struct {
		ID        int64  `json:"ID"`
		Namespace string `json:"Namespace"`
		Value     string `json:"Value"`
	}
	_ = json.NewDecoder(res.Body).Decode(&tagResp)
	res.Body.Close()

	if tagResp.ID <= 0 || tagResp.Value != "live_coding" {
		t.Errorf("unexpected tag: %+v", tagResp)
	}

	// 2. GET /api/tags?namespace=phase
	res, err = http.Get(srv.URL + "/api/tags?namespace=phase")
	if err != nil {
		t.Fatalf("GET /api/tags: %v", err)
	}
	var listTagsResp struct {
		Tags []struct {
			ID    int64  `json:"ID"`
			Value string `json:"Value"`
		} `json:"tags"`
	}
	_ = json.NewDecoder(res.Body).Decode(&listTagsResp)
	res.Body.Close()

	if len(listTagsResp.Tags) == 0 {
		t.Fatalf("expected at least 1 tag")
	}

	// 3. Create job & application, then PATCH /api/applications/{id} with stage_group and tag_ids
	jobID, _ := repo.CreateJob(ctx, store.Job{Title: "Dev", Company: "Tech", URL: "https://tech.com"})
	appID, _ := repo.CreateApplication(ctx, store.Application{JobID: jobID, Status: "pending"})

	patchBody, _ := json.Marshal(map[string]any{
		"stage_group": "in_process",
		"stage_label": "Live Coding Test",
		"tag_ids":     []int64{tagResp.ID},
	})
	req, _ := http.NewRequest(http.MethodPatch, fmt.Sprintf("%s/api/applications/%d", srv.URL, appID), bytes.NewReader(patchBody))
	req.Header.Set("Content-Type", "application/json")
	res, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("PATCH /api/applications: %v", err)
	}
	if res.StatusCode != http.StatusOK {
		t.Fatalf("PATCH status = %d, want 200", res.StatusCode)
	}
	var patchResp struct {
		StageGroup string `json:"stage_group"`
		StageLabel string `json:"stage_label"`
		Tags       []struct {
			ID int64 `json:"ID"`
		} `json:"tags"`
	}
	_ = json.NewDecoder(res.Body).Decode(&patchResp)
	res.Body.Close()

	if patchResp.StageGroup != "in_process" || patchResp.StageLabel != "Live Coding Test" {
		t.Errorf("unexpected stage response: %+v", patchResp)
	}
	if len(patchResp.Tags) != 1 || patchResp.Tags[0].ID != tagResp.ID {
		t.Errorf("unexpected tags in response: %+v", patchResp.Tags)
	}
}

func TestB6FriendlyErrors(t *testing.T) {
	// Verify friendly error structure for LLM errors
	fe401 := llm.ClassifyError(fmt.Errorf("HTTP 401 Unauthorized"), "secret_123")
	if fe401.Code != llm.CodeUnauthorized || fe401.HTTPStatus != http.StatusUnauthorized {
		t.Errorf("fe401 = %+v, want unauthorized", fe401)
	}

	fe429 := llm.ClassifyError(fmt.Errorf("HTTP 429 quota exceeded"), "secret_123")
	if fe429.Code != llm.CodeRateLimited || fe429.HTTPStatus != http.StatusTooManyRequests {
		t.Errorf("fe429 = %+v, want rate_limited", fe429)
	}
}
