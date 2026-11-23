package tasks

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	"github.com/maicondmenezes/openjob/internal/email"
	"github.com/maicondmenezes/openjob/internal/email/connector"
	"github.com/maicondmenezes/openjob/internal/events"
	"github.com/maicondmenezes/openjob/internal/store"
)

func TestSyncEmailHandler(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "openjob.db")
	repo, err := store.Open(dbPath)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer repo.Close()

	bus := events.New()
	defer bus.Close()

	ctx := context.Background()

	// 1. Setup job & application
	jobID, _ := repo.CreateJob(ctx, store.Job{
		Title:   "Go Engineer",
		Company: "Google",
		URL:     "https://google.com/jobs/1",
	})
	appID, _ := repo.CreateApplication(ctx, store.Application{
		JobID:  jobID,
		Status: "pending",
	})

	// 2. Setup account
	acc := store.EmailAccount{
		ID:         "imap_test",
		Provider:   "imap",
		Address:    "me@example.com",
		AuthConfig: []byte("encrypted_dummy"),
		Folder:     "INBOX",
		Enabled:    true,
	}
	if err := repo.SaveEmailAccount(ctx, acc); err != nil {
		t.Fatalf("SaveEmailAccount: %v", err)
	}

	// 3. Mock connector returning a confirmation email
	mockConn := &connector.MockConnector{
		Messages: []email.RawMessage{
			{
				MessageID:   "<msg-google-1@google.com>",
				FromAddress: "recruiting@google.com",
				FromName:    "Google Careers",
				Subject:     "Thank you for applying to Google",
				BodyText:    "We received your application for Go Engineer at Google.",
				ReceivedAt:  time.Now(),
			},
		},
		NewCursor: "105",
	}

	handler := NewSyncEmailHandler(repo, []byte("01234567890123456789012345678901"), nil, bus, nil, func(provider string, authConfig, masterKey []byte) (email.Connector, error) {
		return mockConn, nil
	})

	var reportedProgress []int
	progressFn := func(p int) {
		reportedProgress = append(reportedProgress, p)
	}

	resRaw, err := handler.Run(ctx, json.RawMessage(`{}`), progressFn)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	var res struct {
		EmailsFetched int `json:"emails_fetched"`
		Matched       int `json:"matched"`
		Advanced      int `json:"advanced"`
		Suggested     int `json:"suggested"`
		AccountsRun   int `json:"accounts_run"`
	}
	if err := json.Unmarshal(resRaw, &res); err != nil {
		t.Fatalf("Unmarshal result: %v", err)
	}

	if res.EmailsFetched != 1 || res.Matched != 1 || res.Advanced != 1 {
		t.Errorf("unexpected result: %+v", res)
	}

	// Check application advanced to in_process
	app, _ := repo.GetApplication(ctx, appID)
	if app.StageGroup != "in_process" {
		t.Errorf("StageGroup = %q, want in_process", app.StageGroup)
	}

	// Check account cursor updated
	loadedAcc, _ := repo.GetEmailAccount(ctx, "imap_test")
	if loadedAcc.LastUID != "105" {
		t.Errorf("LastUID = %q, want 105", loadedAcc.LastUID)
	}

	if len(reportedProgress) == 0 || reportedProgress[len(reportedProgress)-1] != 100 {
		t.Errorf("progress did not reach 100: %v", reportedProgress)
	}
}
