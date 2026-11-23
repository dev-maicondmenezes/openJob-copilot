package email

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/maicondmenezes/openjob/internal/events"
	"github.com/maicondmenezes/openjob/internal/store"
)

func TestEngineMonotonicityAndDedup(t *testing.T) {
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

	err = repo.SaveEmailAccount(ctx, store.EmailAccount{
		ID:         "account_1",
		Provider:   "imap",
		Address:    "test@example.com",
		AuthConfig: []byte("enc"),
		Folder:     "INBOX",
		Enabled:    true,
	})
	if err != nil {
		t.Fatalf("SaveEmailAccount: %v", err)
	}

	// Create job and application
	jobID, _ := repo.CreateJob(ctx, store.Job{
		Title:   "Backend Developer",
		Company: "Datadog",
		URL:     "https://datadog.com/jobs/1",
	})

	appID, _ := repo.CreateApplication(ctx, store.Application{
		JobID:  jobID,
		Status: "pending",
	})

	engine := NewEngine(repo, bus, nil, nil, nil)

	// 1. Process ack email from Datadog -> should advance pending -> in_process
	raw1 := RawMessage{
		MessageID:   "<msg-1@datadog.com>",
		FromAddress: "recruiting@datadog.com",
		FromName:    "Datadog Recruiting",
		Subject:     "Application received for Backend Developer at Datadog",
		BodyText:    "Thank you for applying. We will review your application.",
		ReceivedAt:  time.Now(),
	}

	stats, err := engine.ProcessRawMessages(ctx, "account_1", []RawMessage{raw1})
	if err != nil {
		t.Fatalf("ProcessRawMessages: %v", err)
	}
	if stats.Advanced != 1 || stats.Matched != 1 {
		t.Errorf("expected 1 matched and 1 advanced, got %+v", stats)
	}

	app, _ := repo.GetApplication(ctx, appID)
	if app.StageGroup != "in_process" {
		t.Errorf("StageGroup = %q, want in_process", app.StageGroup)
	}
	if app.StageSource != "email_auto" {
		t.Errorf("StageSource = %q, want email_auto", app.StageSource)
	}

	// 2. Dedup test: same email processed again should be skipped
	stats2, err := engine.ProcessRawMessages(ctx, "account_1", []RawMessage{raw1})
	if err != nil {
		t.Fatalf("ProcessRawMessages dedup: %v", err)
	}
	if stats2.Advanced != 0 || stats2.Matched != 0 {
		t.Errorf("expected 0 matched on duplicate, got %+v", stats2)
	}

	// 3. Rejection email -> should advance in_process -> rejected
	raw2 := RawMessage{
		MessageID:   "<msg-2@datadog.com>",
		FromAddress: "recruiting@datadog.com",
		FromName:    "Datadog Recruiting",
		Subject:     "Update on your application at Datadog",
		BodyText:    "Unfortunately, we decided to pursue other candidates at this time.",
		ReceivedAt:  time.Now(),
	}

	stats3, err := engine.ProcessRawMessages(ctx, "account_1", []RawMessage{raw2})
	if err != nil {
		t.Fatalf("ProcessRawMessages rejection: %v", err)
	}
	if stats3.Advanced != 1 {
		t.Errorf("expected 1 advanced to rejected, got %+v", stats3)
	}

	app, _ = repo.GetApplication(ctx, appID)
	if app.StageGroup != "rejected" {
		t.Errorf("StageGroup = %q, want rejected", app.StageGroup)
	}

	// 4. Monotonicity rule: a later ack or interview email MUST NEVER reopen from rejected!
	raw3 := RawMessage{
		MessageID:   "<msg-3@datadog.com>",
		FromAddress: "recruiting@datadog.com",
		FromName:    "Datadog Recruiting",
		Subject:     "Invitation to interview at Datadog",
		BodyText:    "Please schedule your interview at https://calendly.com/interview",
		ReceivedAt:  time.Now(),
	}

	stats4, err := engine.ProcessRawMessages(ctx, "account_1", []RawMessage{raw3})
	if err != nil {
		t.Fatalf("ProcessRawMessages after rejection: %v", err)
	}
	if stats4.Advanced != 0 {
		t.Errorf("monotonicity violation: application advanced out of rejected! stats: %+v", stats4)
	}

	app, _ = repo.GetApplication(ctx, appID)
	if app.StageGroup != "rejected" {
		t.Errorf("StageGroup changed from rejected to %q! Monotonicity violated", app.StageGroup)
	}
}

func TestEngineLowConfidenceCreatesSuggestion(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "openjob.db")

	repo, err := store.Open(dbPath)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer repo.Close()

	ctx := context.Background()

	err = repo.SaveEmailAccount(ctx, store.EmailAccount{
		ID:         "account_1",
		Provider:   "imap",
		Address:    "test@example.com",
		AuthConfig: []byte("enc"),
		Folder:     "INBOX",
		Enabled:    true,
	})
	if err != nil {
		t.Fatalf("SaveEmailAccount: %v", err)
	}

	// Only 1 job & application
	jobID, _ := repo.CreateJob(ctx, store.Job{
		Title:   "Frontend Developer",
		Company: "Random Company",
		URL:     "https://random.com",
	})
	appID, _ := repo.CreateApplication(ctx, store.Application{
		JobID:  jobID,
		Status: "pending",
	})

	engine := NewEngine(repo, nil, nil, nil, nil)

	// Email from ATS without company name in subject/body -> will have ~0.5 confidence (single candidate suggestion)
	raw := RawMessage{
		MessageID:   "<msg-ambiguous@lever.co>",
		FromAddress: "no-reply@hire.lever.co",
		Subject:     "Your application status",
		BodyText:    "Status changed.",
		ReceivedAt:  time.Now(),
	}

	stats, err := engine.ProcessRawMessages(ctx, "account_1", []RawMessage{raw})
	if err != nil {
		t.Fatalf("ProcessRawMessages: %v", err)
	}

	if stats.Advanced != 0 {
		t.Errorf("low confidence should NOT advance stage, but advanced = %d", stats.Advanced)
	}
	if stats.Suggested != 1 {
		t.Errorf("expected 1 suggestion, got %+v", stats)
	}

	// Verify application stayed pending
	app, _ := repo.GetApplication(ctx, appID)
	if app.StageGroup != "pending" {
		t.Errorf("application stage changed to %q, want pending", app.StageGroup)
	}

	// Verify email message saved with link_source = 'suggested'
	suggested, err := repo.ListSuggestedEmailMessages(ctx)
	if err != nil {
		t.Fatalf("ListSuggestedEmailMessages: %v", err)
	}
	if len(suggested) != 1 {
		t.Fatalf("expected 1 suggested email, got %d", len(suggested))
	}
	if suggested[0].LinkSource != "suggested" {
		t.Errorf("LinkSource = %q, want suggested", suggested[0].LinkSource)
	}
}
