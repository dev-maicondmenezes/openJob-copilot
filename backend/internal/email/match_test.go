package email

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/maicondmenezes/openjob/internal/store"
)

func TestMatchCascadeFixtures(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "openjob.db")
	repo, err := store.Open(dbPath)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer repo.Close()

	ctx := context.Background()

	// Setup applications
	app1JobID, _ := repo.CreateJob(ctx, store.Job{
		Title:   "Staff Go Engineer",
		Company: "Stripe",
		URL:     "https://stripe.com/jobs/1",
	})
	app1ID, _ := repo.CreateApplication(ctx, store.Application{JobID: app1JobID, Status: "pending"})

	app2JobID, _ := repo.CreateJob(ctx, store.Job{
		Title:   "Site Reliability Engineer",
		Company: "Nubank",
		URL:     "https://nubank.com.br/carreiras/2",
	})
	app2ID, _ := repo.CreateApplication(ctx, store.Application{JobID: app2JobID, Status: "pending"})

	app3JobID, _ := repo.CreateJob(ctx, store.Job{
		Title:   "Frontend Lead",
		Company: "Vercel",
		URL:     "https://vercel.com/careers/3",
	})
	app3ID, _ := repo.CreateApplication(ctx, store.Application{JobID: app3JobID, Status: "pending"})

	candidates := []MatchCandidate{
		{
			Application: store.Application{ID: app1ID, JobID: app1JobID},
			Job:         store.Job{ID: app1JobID, Title: "Staff Go Engineer", Company: "Stripe", URL: "https://stripe.com/jobs/1"},
		},
		{
			Application: store.Application{ID: app2ID, JobID: app2JobID},
			Job:         store.Job{ID: app2JobID, Title: "Site Reliability Engineer", Company: "Nubank", URL: "https://nubank.com.br/carreiras/2"},
		},
		{
			Application: store.Application{ID: app3ID, JobID: app3JobID},
			Job:         store.Job{ID: app3JobID, Title: "Frontend Lead", Company: "Vercel", URL: "https://vercel.com/careers/3"},
		},
	}

	matcher := NewMatcher(repo, nil, nil)

	tests := []struct {
		name           string
		msg            RawMessage
		wantAppID      *int64
		wantMinConf    float64
		wantMaxConf    float64
		wantReasonPart string
	}{
		{
			name: "Greenhouse relay with company name in subject",
			msg: RawMessage{
				MessageID:   "<gh-123@us.greenhouse-mail.io>",
				FromAddress: "no-reply@us.greenhouse-mail.io",
				FromName:    "Stripe Recruiting",
				Subject:     "Thank you for applying to Stripe",
				BodyText:    "We received your application for Staff Go Engineer.",
				ReceivedAt:  time.Now(),
			},
			wantAppID:      &app1ID,
			wantMinConf:    0.90,
			wantMaxConf:    1.00,
			wantReasonPart: "company_in_subject_or_from",
		},
		{
			name: "Lever relay with company in body",
			msg: RawMessage{
				MessageID:   "<lever-456@hire.lever.co>",
				FromAddress: "jobs@hire.lever.co",
				FromName:    "Talent Team",
				Subject:     "Your application status",
				BodyText:    "Thank you for your interest in Nubank. We are reviewing your profile.",
				ReceivedAt:  time.Now(),
			},
			wantAppID:      &app2ID,
			wantMinConf:    0.85,
			wantMaxConf:    1.00,
			wantReasonPart: "company_in_body",
		},
		{
			name: "Ashby relay with company name in subject",
			msg: RawMessage{
				MessageID:   "<ashby-789@ashbyhq.com>",
				FromAddress: "notifications@ashbyhq.com",
				FromName:    "Vercel",
				Subject:     "Vercel: Next steps for Frontend Lead",
				BodyText:    "We would love to schedule a screening call.",
				ReceivedAt:  time.Now(),
			},
			wantAppID:      &app3ID,
			wantMinConf:    0.90,
			wantMaxConf:    1.00,
			wantReasonPart: "company_in_subject_or_from",
		},
		{
			name: "Workday relay with company in from name",
			msg: RawMessage{
				MessageID:   "<workday-111@myworkday.com>",
				FromAddress: "recruiting@myworkday.com",
				FromName:    "Stripe Careers",
				Subject:     "Application Submission",
				BodyText:    "Your submission was received.",
				ReceivedAt:  time.Now(),
			},
			wantAppID:      &app1ID,
			wantMinConf:    0.90,
			wantMaxConf:    1.00,
			wantReasonPart: "company_in_subject_or_from",
		},
		{
			name: "Corporate domain direct match (not ATS)",
			msg: RawMessage{
				MessageID:   "<direct-222@stripe.com>",
				FromAddress: "recruiter.alice@stripe.com",
				FromName:    "Alice Recruiter",
				Subject:     "Quick chat about engineering roles",
				BodyText:    "Hi! Let's talk about your profile.",
				ReceivedAt:  time.Now(),
			},
			wantAppID:      &app1ID,
			wantMinConf:    0.85,
			wantMaxConf:    1.00,
			wantReasonPart: "company", // could match company name or corporate_domain_match
		},
		{
			name: "Corporate domain direct match without company name in subject or body",
			msg: RawMessage{
				MessageID:   "<direct-333@nubank.com.br>",
				FromAddress: "people@nubank.com.br",
				FromName:    "Talent Acquisition",
				Subject:     "Processo seletivo",
				BodyText:    "Queremos conversar com você.",
				ReceivedAt:  time.Now(),
			},
			wantAppID:      &app2ID,
			wantMinConf:    0.80,
			wantMaxConf:    1.00,
			wantReasonPart: "corporate_domain_match",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := matcher.Match(ctx, tt.msg, candidates)
			if tt.wantAppID != nil {
				if res.ApplicationID == nil || *res.ApplicationID != *tt.wantAppID {
					t.Fatalf("Match ApplicationID = %v, want %v (reason: %s, conf: %f)", res.ApplicationID, *tt.wantAppID, res.Reason, res.Confidence)
				}
			} else {
				if res.ApplicationID != nil {
					t.Fatalf("Match ApplicationID = %v, want nil", *res.ApplicationID)
				}
			}

			if res.Confidence < tt.wantMinConf || res.Confidence > tt.wantMaxConf {
				t.Errorf("Match Confidence = %f, want between %f and %f", res.Confidence, tt.wantMinConf, tt.wantMaxConf)
			}
		})
	}
}
