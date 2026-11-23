package email

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/maicondmenezes/openjob/internal/email/classify"
	"github.com/maicondmenezes/openjob/internal/events"
	"github.com/maicondmenezes/openjob/internal/llm"
	"github.com/maicondmenezes/openjob/internal/store"
)

// SyncStats records the counters from an email sync run.
type SyncStats struct {
	EmailsFetched int `json:"emails_fetched"`
	Matched       int `json:"matched"`
	Advanced      int `json:"advanced"`
	Suggested     int `json:"suggested"`
}

// Stage ranking for monotonicity (ADR 0035).
var stageRank = map[string]int{
	"pending":    0,
	"in_process": 1,
	"interview":  2,
	"offer":      3,
	"rejected":   4,
	"closed":     5,
}

// Engine coordinates email fetching, matching, classification and monotonic updates.
type Engine struct {
	repo       store.Repository
	bus        *events.Bus
	matcher    *Matcher
	classifier *classify.Classifier
	logger     *slog.Logger
}

// NewEngine creates a new pipeline engine.
func NewEngine(repo store.Repository, bus *events.Bus, llmClient llm.Client, llmCfg *llm.Config, logger *slog.Logger) *Engine {
	if logger == nil {
		logger = slog.Default()
	}
	return &Engine{
		repo:       repo,
		bus:        bus,
		matcher:    NewMatcher(repo, llmClient, llmCfg),
		classifier: classify.NewClassifier(llmClient, llmCfg),
		logger:     logger,
	}
}

// ProcessRawMessages ingests a batch of raw messages for an account.
func (e *Engine) ProcessRawMessages(ctx context.Context, accountID string, rawMsgs []RawMessage) (SyncStats, error) {
	var stats SyncStats
	stats.EmailsFetched = len(rawMsgs)

	if len(rawMsgs) == 0 {
		return stats, nil
	}

	// Load candidates (all applications and their jobs)
	apps, err := e.repo.ListApplications(ctx)
	if err != nil {
		return stats, fmt.Errorf("engine: list applications: %w", err)
	}

	var candidates []MatchCandidate
	for _, app := range apps {
		job, err := e.repo.GetJob(ctx, app.JobID)
		if err == nil {
			candidates = append(candidates, MatchCandidate{
				Application: app,
				Job:         job,
			})
		}
	}

	for _, raw := range rawMsgs {
		if raw.MessageID == "" {
			continue
		}

		// Dedup by Message-ID (RFC 5322)
		if existing, _ := e.repo.GetEmailMessageByMessageID(ctx, raw.MessageID); existing != nil {
			continue
		}

		// 1. Match against open applications
		matchRes := e.matcher.Match(ctx, raw, candidates)

		// 2. Classify signal
		classRes := e.classifier.Classify(ctx, raw.Subject, raw.BodyText)

		var matchedAppID *int64
		var linkSource string
		confidence := matchRes.Confidence

		if matchRes.ApplicationID != nil {
			matchedAppID = matchRes.ApplicationID
			stats.Matched++

			if confidence >= 0.80 {
				linkSource = "auto"
			} else if confidence > 0.0 {
				linkSource = "suggested"
				stats.Suggested++
			}
		}

		now := time.Now().UTC()
		emailRecord := store.EmailMessage{
			ID:                   generateULID(),
			AccountID:            accountID,
			MessageID:            raw.MessageID,
			ThreadID:             raw.ThreadID,
			FromAddress:          raw.FromAddress,
			FromName:             raw.FromName,
			Subject:              raw.Subject,
			ReceivedAt:           raw.ReceivedAt,
			BodyText:             TruncateBody(raw.BodyText),
			MatchedApplicationID: matchedAppID,
			MatchConfidence:      confidence,
			StageSignal:          classRes.Signal,
			ClassifiedAt:         &now,
			LinkSource:           linkSource,
		}

		if err := e.repo.SaveEmailMessage(ctx, emailRecord); err != nil {
			e.logger.Error("engine: save email message failed", "msg_id", raw.MessageID, "err", err)
			continue
		}

		// 3. Monotonic advance if linkSource == "auto"
		if linkSource == "auto" && matchedAppID != nil {
			advanced, err := e.applyMonotonicAdvance(ctx, *matchedAppID, classRes.Signal)
			if err != nil {
				e.logger.Warn("engine: apply transition failed", "app_id", *matchedAppID, "err", err)
			} else if advanced {
				stats.Advanced++
			}
		}
	}

	return stats, nil
}

// applyMonotonicAdvance checks monotonic progression rules (ADR 0035).
// Returns true if a transition was applied.
func (e *Engine) applyMonotonicAdvance(ctx context.Context, appID int64, signal string) (bool, error) {
	app, err := e.repo.GetApplication(ctx, appID)
	if err != nil {
		return false, err
	}

	targetStage, stageLabel := mapSignalToStage(signal)
	if targetStage == "" {
		return false, nil // no state transition needed for this signal
	}

	currStage := app.StageGroup
	if currStage == "" {
		currStage = "pending"
	}

	// Never reopen or change a terminal state automatically
	if currStage == "rejected" || currStage == "closed" {
		e.logger.Info("engine: monotonic check refused transition: application already in terminal state",
			"app_id", appID, "current", currStage, "attempted", targetStage)
		return false, nil
	}

	// Never allow moving backward in rank
	currRank := stageRank[currStage]
	targetRank := stageRank[targetStage]

	// Rejection can happen from any non-terminal state
	if targetStage == "rejected" {
		if err := e.repo.UpdateApplicationStage(ctx, appID, "rejected", stageLabel, "email_auto"); err != nil {
			return false, err
		}
		e.publishStatusChange(ctx, app, "rejected")
		return true, nil
	}

	// For other stages: targetRank must be strictly greater than currRank
	if targetRank <= currRank {
		e.logger.Info("engine: monotonic check refused transition: target rank not higher than current",
			"app_id", appID, "current", currStage, "attempted", targetStage)
		return false, nil
	}

	if err := e.repo.UpdateApplicationStage(ctx, appID, targetStage, stageLabel, "email_auto"); err != nil {
		return false, err
	}

	// Create/assign phase tag
	tagVal := sanitizeTagValue(signal)
	if tag, err := e.repo.GetOrCreateTag(ctx, "phase", tagVal, stageLabel); err == nil {
		_ = e.repo.AddApplicationTag(ctx, appID, tag.ID)
	}

	e.publishStatusChange(ctx, app, targetStage)
	return true, nil
}

func (e *Engine) publishStatusChange(ctx context.Context, app store.Application, newStage string) {
	if e.bus != nil {
		e.bus.Publish(events.Event{
			Kind:     events.KindApplicationStatusChanged,
			Severity: events.SeverityInfo,
			Title:    fmt.Sprintf("Candidatura avançou para %s", newStage),
			Link:     "/applications",
			Metadata: map[string]any{
				"application_id": app.ID,
				"job_id":         app.JobID,
				"previous":       app.StageGroup,
				"status":         newStage,
				"stage_group":    newStage,
				"source":         "email_auto",
			},
		})
	}
}

func mapSignalToStage(signal string) (stageGroup, stageLabel string) {
	switch signal {
	case classify.SignalAck:
		return "in_process", "Recebida"
	case classify.SignalScreening:
		return "in_process", "Triagem"
	case classify.SignalAssessment:
		return "in_process", "Desafio Técnico"
	case classify.SignalInfoRequest:
		return "in_process", "Dados Adicionais"
	case classify.SignalInterview:
		return "interview", "Entrevista"
	case classify.SignalOffer:
		return "offer", "Proposta"
	case classify.SignalRejection:
		return "rejected", "Rejeitada"
	default:
		return "", ""
	}
}

func sanitizeTagValue(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	return strings.ReplaceAll(s, " ", "_")
}

const crockford = "0123456789ABCDEFGHJKMNPQRSTVWXYZ"

var (
	ulidMu   sync.Mutex
	ulidLast uint64
)

func generateULID() string {
	ulidMu.Lock()
	defer ulidMu.Unlock()

	ms := uint64(time.Now().UnixMilli())
	if ms <= ulidLast {
		ms = ulidLast + 1
	}
	ulidLast = ms

	var entropy [10]byte
	if err := binary.Read(rand.Reader, binary.BigEndian, &entropy); err != nil {
		for i := range entropy {
			entropy[i] = byte(ms >> (uint(i%8) * 8))
		}
	}

	var bytes [16]byte
	bytes[0] = byte(ms >> 40)
	bytes[1] = byte(ms >> 32)
	bytes[2] = byte(ms >> 24)
	bytes[3] = byte(ms >> 16)
	bytes[4] = byte(ms >> 8)
	bytes[5] = byte(ms)
	copy(bytes[6:], entropy[:])

	var sb [26]byte
	acc := uint(0)
	bits := uint(0)
	pos := 0
	for _, b := range bytes {
		acc = acc<<8 | uint(b)
		bits += 8
		for bits >= 5 {
			bits -= 5
			sb[pos] = crockford[(acc>>bits)&31]
			pos++
		}
	}
	if bits > 0 {
		sb[pos] = crockford[(acc<<(5-bits))&31]
	}
	return string(sb[:])
}
