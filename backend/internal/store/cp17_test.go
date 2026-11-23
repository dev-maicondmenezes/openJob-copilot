package store

import (
	"context"
	"path/filepath"
	"testing"
	"time"
)

func TestCP17StoreEmailAndTags(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "openjob.db")

	repo, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer repo.Close()

	ctx := context.Background()

	// 1. Jobs & Applications with stage_group
	jobID, err := repo.CreateJob(ctx, Job{
		Title:       "Software Engineer",
		Company:     "Acme Corp",
		Description: "Go developer needed",
		URL:         "https://acme.com/jobs/1",
	})
	if err != nil {
		t.Fatalf("CreateJob: %v", err)
	}

	appID, err := repo.CreateApplication(ctx, Application{
		JobID:  jobID,
		Status: "pending",
	})
	if err != nil {
		t.Fatalf("CreateApplication: %v", err)
	}

	app, err := repo.GetApplication(ctx, appID)
	if err != nil {
		t.Fatalf("GetApplication: %v", err)
	}
	if app.StageGroup != "pending" {
		t.Errorf("StageGroup = %q, want pending", app.StageGroup)
	}
	if app.StageSource != "manual" {
		t.Errorf("StageSource = %q, want manual", app.StageSource)
	}

	// Update stage group and label
	if err := repo.UpdateApplicationStage(ctx, appID, "in_process", "Triagem de currículo", "email_auto"); err != nil {
		t.Fatalf("UpdateApplicationStage: %v", err)
	}
	app, _ = repo.GetApplication(ctx, appID)
	if app.StageGroup != "in_process" {
		t.Errorf("StageGroup = %q, want in_process", app.StageGroup)
	}
	if app.StageLabel != "Triagem de currículo" {
		t.Errorf("StageLabel = %q, want Triagem de currículo", app.StageLabel)
	}
	if app.StageSource != "email_auto" {
		t.Errorf("StageSource = %q, want email_auto", app.StageSource)
	}
	if app.SentAt == nil {
		t.Errorf("expected SentAt to be stamped on moving to in_process")
	}

	// 2. Email Account
	acc := EmailAccount{
		ID:         "imap",
		Provider:   "imap",
		Address:    "user@example.com",
		AuthConfig: []byte("encrypted_secret"),
		Folder:     "INBOX",
		Enabled:    true,
	}
	if err := repo.SaveEmailAccount(ctx, acc); err != nil {
		t.Fatalf("SaveEmailAccount: %v", err)
	}
	loadedAcc, err := repo.GetEmailAccount(ctx, "imap")
	if err != nil {
		t.Fatalf("GetEmailAccount: %v", err)
	}
	if loadedAcc.Address != "user@example.com" || !loadedAcc.Enabled {
		t.Errorf("unexpected account data: %+v", loadedAcc)
	}

	now := time.Now().UTC().Truncate(time.Second)
	if err := repo.UpdateEmailAccountCursor(ctx, "imap", "100", "", now, ""); err != nil {
		t.Fatalf("UpdateEmailAccountCursor: %v", err)
	}
	loadedAcc, _ = repo.GetEmailAccount(ctx, "imap")
	if loadedAcc.LastUID != "100" {
		t.Errorf("LastUID = %q, want 100", loadedAcc.LastUID)
	}

	// 3. Email Messages
	msgID := "msg_ulid_1"
	rfcMsgID := "<unique-msg-1@greenhouse.io>"
	threadID := "thread_123"
	msg := EmailMessage{
		ID:                   msgID,
		AccountID:            "imap",
		MessageID:            rfcMsgID,
		ThreadID:             threadID,
		FromAddress:          "no-reply@greenhouse.io",
		FromName:             "Acme Recruiting",
		Subject:              "Recebemos sua candidatura para Software Engineer",
		ReceivedAt:           now,
		BodyText:             "Olá, recebemos sua candidatura na Acme Corp.",
		MatchedApplicationID: &appID,
		MatchConfidence:      0.95,
		StageSignal:          "ack",
		ClassifiedAt:         &now,
		LinkSource:           "auto",
	}
	if err := repo.SaveEmailMessage(ctx, msg); err != nil {
		t.Fatalf("SaveEmailMessage: %v", err)
	}

	// Find by thread
	appFromThread, err := repo.FindApplicationByThreadID(ctx, threadID)
	if err != nil {
		t.Fatalf("FindApplicationByThreadID: %v", err)
	}
	if appFromThread.ID != appID {
		t.Errorf("appFromThread.ID = %d, want %d", appFromThread.ID, appID)
	}

	// List by application
	msgs, err := repo.ListEmailMessagesByApplication(ctx, appID)
	if err != nil {
		t.Fatalf("ListEmailMessagesByApplication: %v", err)
	}
	if len(msgs) != 1 || msgs[0].MessageID != rfcMsgID {
		t.Errorf("unexpected msgs: %+v", msgs)
	}

	// 4. Tags
	tagPhase, err := repo.GetOrCreateTag(ctx, "phase", "technical_interview", "Entrevista Técnica")
	if err != nil {
		t.Fatalf("GetOrCreateTag: %v", err)
	}
	tagPriority, err := repo.GetOrCreateTag(ctx, "priority", "high", "Alta Prioridade")
	if err != nil {
		t.Fatalf("GetOrCreateTag: %v", err)
	}

	if err := repo.SetApplicationTags(ctx, appID, []int64{tagPhase.ID, tagPriority.ID}); err != nil {
		t.Fatalf("SetApplicationTags: %v", err)
	}
	appTags, err := repo.GetApplicationTags(ctx, appID)
	if err != nil {
		t.Fatalf("GetApplicationTags: %v", err)
	}
	if len(appTags) != 2 {
		t.Fatalf("expected 2 tags, got %d", len(appTags))
	}

	if err := repo.RemoveApplicationTag(ctx, appID, tagPriority.ID); err != nil {
		t.Fatalf("RemoveApplicationTag: %v", err)
	}
	appTags, _ = repo.GetApplicationTags(ctx, appID)
	if len(appTags) != 1 || appTags[0].Value != "technical_interview" {
		t.Fatalf("expected 1 tag technical_interview, got %+v", appTags)
	}
}
