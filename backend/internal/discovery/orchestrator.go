package discovery

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/maicondmenezes/openjob/internal/aiusage"
	"github.com/maicondmenezes/openjob/internal/discovery/ratelimit"
	"github.com/maicondmenezes/openjob/internal/events"
	"github.com/maicondmenezes/openjob/internal/filtering"
	"github.com/maicondmenezes/openjob/internal/llm"
	"github.com/maicondmenezes/openjob/internal/score"
	"github.com/maicondmenezes/openjob/internal/store"
)

const (
	SettingDiscoveryConfig = "discovery_config"
	SettingResumeKeywords  = "resume_keywords"
	SettingResumeHash      = "resume_hash"
	maxBatchScoringPerRun  = 15
)

// Orchestrator coordinates vacancy discovery across all enabled sources,
// executing keyword and resume filters, deduplicating, and scoring. AI scoring
// is queued (CP10); deterministic scoring runs inline so a discovered job has
// its fit as soon as the run returns.
type Orchestrator struct {
	repo      store.Repository
	registry  *Registry
	masterKey []byte
	llmClient llm.Client
	logger    *slog.Logger
	enqueuer  ScoreEnqueuer  // optional (nil = legacy inline scoring)
	publisher EventPublisher // optional (nil = no events)
	limiter   *ratelimit.Limiter

	// runMu serializes concurrent Discover calls (CP14): repeated "Buscar
	// agora" clicks are queued by the task pool and executed one at a time,
	// never in parallel against the same sources.
	runMu sync.Mutex

	mu           sync.Mutex
	running      bool
	currentRunID int64

	tickerMu   sync.Mutex
	ticker     *time.Ticker
	tickerStop chan struct{}

	// Optional deterministic integrations (Fase 5). When nil the legacy
	// heuristics remain active, preserving rollback through feature flags.
	keywordExtractor    KeywordExtractor
	resumeFilterer      ResumeFilterer
	jobIndexer          JobIndexer
	deterministicScorer DeterministicScorer
}

// KeywordExtractor extracts resume keywords deterministically.
type KeywordExtractor func(curriculumMD string, topN int) []string

// ResumeFilterer applies the deterministic filter policy to one vacancy.
type ResumeFilterer interface {
	FilterJob(ctx context.Context, jobID int64, resumeKeywords []string, curriculumMD, title, description string, policy filtering.Policy) (filtering.Result, error)
}

// JobIndexer stores vacancies in the global search index.
type JobIndexer interface {
	IndexJob(ctx context.Context, jobID int64, title, description string) error
}

// DeterministicScorer computes the deterministic fit without the LLM.
type DeterministicScorer func(ctx context.Context, curriculumMD, title, description string) (score int, reason string, err error)

// SetKeywordExtractor wires deterministic keyword extraction.
func (o *Orchestrator) SetKeywordExtractor(extractor KeywordExtractor) {
	o.keywordExtractor = extractor
}

// SetResumeFilterer wires the deterministic resume filter.
func (o *Orchestrator) SetResumeFilterer(filterer ResumeFilterer) {
	o.resumeFilterer = filterer
}

// SetJobIndexer wires the global job search index.
func (o *Orchestrator) SetJobIndexer(indexer JobIndexer) {
	o.jobIndexer = indexer
}

// SetDeterministicScorer wires the inline deterministic scoring fallback used
// when no task queue is present.
func (o *Orchestrator) SetDeterministicScorer(scorer DeterministicScorer) {
	o.deterministicScorer = scorer
}

// ScoreEnqueuer receives the new jobs of a run for asynchronous scoring —
// implemented by the tasks queue (Pool) in production (docs/adr/0010).
type ScoreEnqueuer interface {
	EnqueueScoreJob(ctx context.Context, jobID int64, priority int) error
}

// EventPublisher receives discovery lifecycle events — implemented by the
// in-process event bus (docs/adr/0009).
type EventPublisher interface {
	Publish(e events.Event)
}

func NewOrchestrator(repo store.Repository, registry *Registry, masterKey []byte, llmClient llm.Client, logger *slog.Logger) *Orchestrator {
	if registry == nil {
		registry = NewDefaultRegistry()
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &Orchestrator{
		repo:      repo,
		registry:  registry,
		masterKey: masterKey,
		llmClient: llmClient,
		logger:    logger,
		limiter:   ratelimit.New(repo),
	}
}

// Registry returns the underlying adapter registry.
func (o *Orchestrator) Registry() *Registry {
	return o.registry
}

// Limiter exposes the persistent rate limiter (used by the sources API to
// render usage/health and to gate the manual "test source" action).
func (o *Orchestrator) Limiter() *ratelimit.Limiter {
	return o.limiter
}

// WaiterFor returns the token bucket of a source configured from its
// capabilities — also used by POST /api/sources/{id}/test.
func (o *Orchestrator) WaiterFor(sourceID string, caps Capabilities) *ratelimit.SourceWaiter {
	return o.limiter.Waiter(sourceID, ratelimit.Config{
		Requests: caps.RateLimit.Requests,
		Period:   caps.RateLimit.Period,
		Strategy: caps.RateLimit.Strategy,
	})
}

// SetScoreEnqueuer wires the async scoring queue (CP10), used for AI scoring.
// Deterministic scoring runs inline; nil keeps the legacy inline path for
// unit tests.
func (o *Orchestrator) SetScoreEnqueuer(e ScoreEnqueuer) {
	o.enqueuer = e
}

// SetPublisher wires the event bus (CP10). Best-effort: events never change
// the run outcome.
func (o *Orchestrator) SetPublisher(p EventPublisher) {
	o.publisher = p
}

// publish emits an event to the bus, ignoring failures (best-effort contract).
func (o *Orchestrator) publish(e events.Event) {
	if o.publisher != nil {
		o.publisher.Publish(e)
	}
}

// IsRunning reports whether a discovery run is actively executing.
func (o *Orchestrator) IsRunning() bool {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.running
}

// CurrentRunID returns the ID of the active run, or 0 if idle.
func (o *Orchestrator) CurrentRunID() int64 {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.currentRunID
}

// LoadConfig loads discovery configuration from settings, or returns defaults.
func (o *Orchestrator) LoadConfig(ctx context.Context) (Config, error) {
	raw, err := o.repo.GetSetting(ctx, SettingDiscoveryConfig)
	if errors.Is(err, store.ErrNotFound) {
		return DefaultConfig(), nil
	}
	if err != nil {
		return DefaultConfig(), err
	}
	var cfg Config
	if err := json.Unmarshal([]byte(raw), &cfg); err != nil {
		return DefaultConfig(), nil
	}
	return cfg, nil
}

// SaveConfig persists discovery configuration in settings.
func (o *Orchestrator) SaveConfig(ctx context.Context, cfg Config) error {
	// Hard validation: maximum 5 keywords
	cleanKws := make([]string, 0, len(cfg.Keywords))
	seen := make(map[string]bool)
	for _, kw := range cfg.Keywords {
		t := strings.TrimSpace(kw)
		if t != "" && !seen[strings.ToLower(t)] {
			seen[strings.ToLower(t)] = true
			cleanKws = append(cleanKws, t)
		}
	}
	if len(cleanKws) > 5 {
		return errors.New("máximo de 5 palavras-chave permitido")
	}
	cfg.Keywords = cleanKws

	if cfg.ResumeFilter.MinMatches <= 0 {
		cfg.ResumeFilter.MinMatches = 3
	}
	if cfg.ResumeFilter.MinMatches > 10 {
		cfg.ResumeFilter.MinMatches = 10
	}
	if cfg.ResumeFilter.MatchMode == "" {
		cfg.ResumeFilter.MatchMode = "any"
	}

	b, err := json.Marshal(cfg)
	if err != nil {
		return err
	}
	if err := o.repo.SetSetting(ctx, SettingDiscoveryConfig, string(b)); err != nil {
		return err
	}

	// Update ticker schedule based on interval
	o.ConfigureTicker(cfg.DiscoveryInterval)
	return nil
}

// GetResumeKeywords loads cached resume keywords.
func (o *Orchestrator) GetResumeKeywords(ctx context.Context) ([]string, error) {
	raw, err := o.repo.GetSetting(ctx, SettingResumeKeywords)
	if errors.Is(err, store.ErrNotFound) {
		return []string{}, nil
	}
	if err != nil {
		return nil, err
	}
	var kws []string
	if err := json.Unmarshal([]byte(raw), &kws); err != nil {
		return []string{}, nil
	}
	return kws, nil
}

// RefreshResumeKeywords extracts top keywords from latest curriculum and caches them.
func (o *Orchestrator) RefreshResumeKeywords(ctx context.Context) ([]string, error) {
	curriculum, err := o.repo.LatestCurriculum(ctx)
	if errors.Is(err, store.ErrNotFound) {
		return []string{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read curriculum: %w", err)
	}

	keywords := ExtractResumeKeywords(curriculum.ContentMD, 20)
	if o.keywordExtractor != nil {
		if extracted := o.keywordExtractor(curriculum.ContentMD, 20); len(extracted) > 0 {
			keywords = extracted
		}
	}
	b, err := json.Marshal(keywords)
	if err != nil {
		return nil, err
	}
	if err := o.repo.SetSetting(ctx, SettingResumeKeywords, string(b)); err != nil {
		return nil, fmt.Errorf("cache resume keywords: %w", err)
	}
	return keywords, nil
}

// Discover executes an end-to-end automated discovery run.
func (o *Orchestrator) Discover(ctx context.Context) (int64, RunStats, error) {
	// CP14: serialize instead of rejecting. The HTTP API enqueues one task per
	// click; the task workers may call concurrently, so the second run waits
	// here and starts as soon as the first one releases the sources.
	o.runMu.Lock()
	defer o.runMu.Unlock()

	o.mu.Lock()
	o.running = true
	o.mu.Unlock()

	defer func() {
		o.mu.Lock()
		o.running = false
		o.currentRunID = 0
		o.mu.Unlock()
	}()

	// Initialize run record in DB
	runID, err := o.repo.CreateDiscoveryRun(ctx, store.DiscoveryRun{
		Status: "running",
		Stats:  `{"fetched":0}`,
	})
	if err != nil {
		return 0, RunStats{}, fmt.Errorf("orchestrator: create run: %w", err)
	}

	o.mu.Lock()
	o.currentRunID = runID
	o.mu.Unlock()

	stats, runErr := o.runPipeline(ctx, runID)

	// Finalize run in DB
	now := time.Now().UTC()
	stats.CompletedAt = now
	status := "completed"
	errMsg := ""
	if runErr != nil {
		status = "failed"
		errMsg = runErr.Error()
	}

	statsBytes, _ := json.Marshal(stats)
	_ = o.repo.UpdateDiscoveryRun(ctx, store.DiscoveryRun{
		ID:           runID,
		CompletedAt:  &now,
		Status:       status,
		Stats:        string(statsBytes),
		ErrorMessage: errMsg,
	})

	// CP10: run lifecycle events feed the in-app bell (and future channels).
	if runErr != nil {
		o.publish(events.Event{
			Kind:     events.KindDiscoveryFailed,
			Severity: events.SeverityError,
			Title:    "Busca de vagas falhou",
			Body:     runErr.Error(),
			Link:     "/jobs",
			Metadata: map[string]any{"run_id": runID},
		})
	} else {
		o.publish(events.Event{
			Kind:     events.KindDiscoveryCompleted,
			Severity: events.SeveritySuccess,
			Title:    "Busca de vagas concluída",
			Link:     "/jobs",
			Metadata: map[string]any{
				"run_id":         runID,
				"fetched":        stats.Fetched,
				"new_jobs":       stats.NewJobs,
				"queued_scoring": stats.Scored,
				"errors":         stats.Errors,
			},
		})
	}

	return runID, stats, runErr
}

type jobToScore struct {
	ID                 int64
	Title              string
	Company            string
	Description        string
	ResumeMatchesCount int
}

func (o *Orchestrator) runPipeline(ctx context.Context, runID int64) (RunStats, error) {
	stats := RunStats{
		Errors:  []string{},
		Sources: []SourceRunStat{},
	}

	cfg, err := o.LoadConfig(ctx)
	if err != nil {
		o.logger.Warn("orchestrator: fallback to default config", "err", err)
		cfg = DefaultConfig()
	}

	// Prepare resume keywords if enabled
	resumeKeywords, _ := o.GetResumeKeywords(ctx)
	if cfg.ResumeFilter.Enabled && len(resumeKeywords) == 0 {
		// Attempt auto-extraction on the fly
		if extracted, err := o.RefreshResumeKeywords(ctx); err == nil && len(extracted) > 0 {
			resumeKeywords = extracted
		}
	}

	// The deterministic filter needs the full curriculum text; load it once.
	curriculumMD := ""
	if cfg.ResumeFilter.Enabled && o.resumeFilterer != nil {
		if cur, err := o.repo.LatestCurriculum(ctx); err == nil {
			curriculumMD = cur.ContentMD
		}
	}

	sources, err := o.repo.ListSources(ctx)
	if err != nil {
		return stats, fmt.Errorf("list sources: %w", err)
	}

	var allFetched []DiscoveredJob

	// CP14: API sources run first (fast, no quota), then scraping sources at
	// their conservative pace. Within each group the user priority wins.
	type runnableSource struct {
		src     store.Source
		adapter Adapter
		caps    Capabilities
	}
	runnable := make([]runnableSource, 0, len(sources))
	for _, src := range sources {
		if !src.Enabled {
			continue
		}
		adapter, ok := o.registry.Get(src.Adapter)
		if !ok {
			errMsg := fmt.Sprintf("fonte %s: adapter %q não encontrado", src.ID, src.Adapter)
			o.logger.Warn(errMsg)
			stats.Errors = append(stats.Errors, errMsg)
			stats.Sources = append(stats.Sources, SourceRunStat{
				SourceID: src.ID,
				Adapter:  src.Adapter,
				Fetched:  0,
				Error:    errMsg,
			})
			continue
		}
		caps := adapter.DefaultCapabilities()
		if src.Capabilities != "" {
			if err := json.Unmarshal([]byte(src.Capabilities), &caps); err != nil {
				caps = adapter.DefaultCapabilities()
			}
		}
		if caps.AccessType == "" {
			caps.AccessType = AccessAPI
		}
		runnable = append(runnable, runnableSource{src: src, adapter: adapter, caps: caps})
	}
	sort.SliceStable(runnable, func(i, j int) bool {
		iScrape := runnable[i].caps.IsScraping()
		jScrape := runnable[j].caps.IsScraping()
		if iScrape != jScrape {
			return !iScrape // API first
		}
		if runnable[i].src.Priority != runnable[j].src.Priority {
			return runnable[i].src.Priority < runnable[j].src.Priority
		}
		return runnable[i].src.ID < runnable[j].src.ID
	})

	// Fetch from each enabled source
	for _, item := range runnable {
		src, adapter, caps := item.src, item.adapter, item.caps

		stat := SourceRunStat{
			SourceID:   src.ID,
			Adapter:    src.Adapter,
			AccessType: caps.AccessType,
			DailyQuota: caps.DailyQuota,
			Status:     "ok",
		}

		// Layer 3a: circuit breaker / exponential backoff. A source in its
		// penalty window is skipped without touching the network.
		if gap, err := o.limiter.Check(ctx, src.ID); err == nil && gap.Blocked {
			stat.Skipped = true
			stat.Status = gap.Status
			stat.Note = gap.Reason
			if gap.Until != nil {
				stat.BackoffUntil = gap.Until.Format(time.RFC3339)
			}
			o.logger.Warn("orchestrator: source temporarily disabled", "source", src.ID, "status", gap.Status, "reason", gap.Reason)
			stats.Sources = append(stats.Sources, stat)
			continue
		}

		// Layer 3b: persistent daily quota (Vagas.com.br details, InfoJobs
		// requests). Exhausted today → skip; the counter resets tomorrow.
		// A details quota is special: the listing itself is free, so the
		// source still runs and only the detail enrichment is disabled.
		dailyRemaining := -1
		if caps.RequiresDailyQuota && caps.DailyQuota > 0 {
			remaining, err := o.limiter.Remaining(ctx, src.ID, caps.DailyQuota, caps.QuotaUnit)
			if err == nil {
				dailyRemaining = remaining
				stat.DailyUsed = caps.DailyQuota - remaining
				if remaining == 0 {
					if caps.QuotaUnit == "details" {
						stat.Note = "limite diário de detalhes atingido — detalhes disponíveis amanhã"
						o.logger.Info("orchestrator: detail quota exhausted, listing only", "source", src.ID, "quota", caps.DailyQuota)
					} else {
						stat.Skipped = true
						stat.Status = "quota"
						stat.Note = "limite diário atingido — tente novamente amanhã"
						o.logger.Info("orchestrator: daily quota exhausted", "source", src.ID, "quota", caps.DailyQuota)
						stats.Sources = append(stats.Sources, stat)
						continue
					}
				}
			}
		}

		// Decrypt auth key if present
		apiKey := ""
		if src.AuthType == "api_key" && src.AuthConfig != "" {
			if decrypted, err := store.DecryptString(o.masterKey, src.AuthConfig); err == nil {
				apiKey = decrypted
			}
		}

		usage := &UsageTracker{}
		srcCfg := SourceConfig{
			ID:             src.ID,
			Name:           src.Name,
			Adapter:        src.Adapter,
			Enabled:        src.Enabled,
			Priority:       src.Priority,
			AuthType:       src.AuthType,
			APIKey:         apiKey,
			Capabilities:   caps,
			RateLimiter:    o.WaiterFor(src.ID, caps),
			DailyRemaining: dailyRemaining,
			Usage:          usage,
		}

		// Fetch with fault tolerance: single source error must not break the whole run
		var srcKeywords []string
		if caps.SupportsKeywordSearch {
			srcKeywords = cfg.Keywords
		}

		jobs, fetchErr := adapter.Fetch(ctx, srcCfg, srcKeywords)
		stat.Fetched = len(jobs)

		// Layer 3c: persist the quota consumption even on partial failures.
		if u, err := o.limiter.ConsumeDaily(ctx, src.ID, usage.Requests, usage.DetailRequests); err == nil {
			stat.DetailsUsed = u.DetailsUsed
			if caps.QuotaUnit == "details" {
				stat.DailyUsed = u.DetailsUsed
			} else {
				stat.DailyUsed = u.Used
			}
		}

		if fetchErr != nil {
			errMsg := fmt.Sprintf("%s (%s): %v", src.Name, src.ID, fetchErr)
			o.logger.Error("orchestrator: source fetch failed", "source", src.ID, "err", fetchErr)
			stat.Error = fetchErr.Error()
			stats.Errors = append(stats.Errors, errMsg)

			var blockedErr *BlockedError
			blocked := errors.As(fetchErr, &blockedErr)
			if res, herr := o.limiter.RecordFailure(ctx, src.ID, fetchErr.Error(), blocked); herr == nil {
				stat.Status = res.Health.Status
				if res.Health.DegradedUntil != nil {
					stat.BackoffUntil = res.Health.DegradedUntil.Format(time.RFC3339)
				} else if res.Health.BackoffUntil != nil {
					stat.BackoffUntil = res.Health.BackoffUntil.Format(time.RFC3339)
				}
				if res.DidDegrade {
					o.publish(events.Event{
						Kind:     events.KindSourceDegraded,
						Severity: events.SeverityWarning,
						Title:    "Fonte degradada",
						Body:     fmt.Sprintf("%s foi desativada por 24h após 3 falhas consecutivas.", src.Name),
						Link:     "/settings/sources",
						DedupKey: "source_degraded:" + src.ID,
						Metadata: map[string]any{"source": src.ID, "adapter": src.Adapter, "failures": res.Health.ConsecutiveFailures},
					})
				}
			}

			// CP10: per-source failure event — deduplicated by the notifier so
			// repeated ticks do not spam the bell.
			o.publish(events.Event{
				Kind:     events.KindSourceFetchFailed,
				Severity: events.SeverityWarning,
				Title:    "Falha em fonte de vagas",
				Body:     fmt.Sprintf("%s: %v", src.Name, fetchErr),
				Link:     "/settings/sources",
				DedupKey: "source_error:" + src.ID,
				Metadata: map[string]any{"source": src.ID, "adapter": src.Adapter},
			})
		} else {
			allFetched = append(allFetched, jobs...)
			_ = o.limiter.RecordSuccess(ctx, src.ID)
		}
		stats.Sources = append(stats.Sources, stat)
	}

	stats.Fetched = len(allFetched)

	// Filter stages & Persistence
	var newJobsToScore []jobToScore

	for _, job := range allFetched {
		// 1. Keyword filter (client-side safety net and for sources without search)
		kwMatch, matchedKws := FilterKeywords(job.Title, job.Description, cfg.Keywords, "any")
		if !kwMatch {
			stats.FilteredKeywords++
			continue
		}

		// 2. Resume filter (curriculum adherence threshold)
		resMatch, matchedResume := o.filterResumeDecision(ctx, job.Title, job.Description, resumeKeywords, curriculumMD, cfg)
		if !resMatch {
			stats.FilteredResume++
			continue
		}

		// 3. Persistence with atomic deduplication on UNIQUE url
		dbJob := store.Job{
			Title:              job.Title,
			Company:            job.Company,
			URL:                job.URL,
			Description:        job.Description,
			Source:             job.Source,
			Status:             "new",
			ExternalID:         job.ExternalID,
			MatchedKeywords:    matchedKws,
			MatchedResumeTerms: matchedResume,
		}

		id, inserted, err := o.repo.InsertDiscoveredJob(ctx, dbJob)
		if err != nil {
			o.logger.Error("orchestrator: insert job failed", "url", job.URL, "err", err)
			continue
		}
		if !inserted {
			stats.Duplicates++
			continue
		}

		if o.jobIndexer != nil {
			if err := o.jobIndexer.IndexJob(ctx, id, job.Title, job.Description); err != nil {
				o.logger.Warn("orchestrator: index job failed", "job_id", id, "err", err)
			}
		}

		stats.NewJobs++
		newJobsToScore = append(newJobsToScore, jobToScore{
			ID:                 id,
			Title:              job.Title,
			Company:            job.Company,
			Description:        job.Description,
			ResumeMatchesCount: len(matchedResume),
		})
	}

	// 4. Batch Scoring with rate limiting
	// Prioritize jobs with higher resume match count
	sort.Slice(newJobsToScore, func(i, j int) bool {
		return newJobsToScore[i].ResumeMatchesCount > newJobsToScore[j].ResumeMatchesCount
	})

	deterministic := o.deterministicScoringActive(ctx)
	if len(newJobsToScore) > maxBatchScoringPerRun && !deterministic {
		newJobsToScore = newJobsToScore[:maxBatchScoringPerRun]
	}

	// 4. Scoring. Deterministic scoring is local and fast, so it runs inline
	// during discovery: the jobs are already scored when the run returns and
	// the UI never shows a freshly discovered job without a fit. AI scoring
	// still goes through the queue, where the rate limiter applies.
	if o.enqueuer != nil && !deterministic {
		enqueued := 0
		for _, j := range newJobsToScore {
			prio := scoringPriority(j.ResumeMatchesCount)
			if err := o.enqueuer.EnqueueScoreJob(ctx, j.ID, prio); err != nil {
				o.logger.Error("orchestrator: enqueue score failed", "job_id", j.ID, "err", err)
				continue
			}
			enqueued++
		}
		stats.Scored = enqueued
		return stats, nil
	}

	stats.Scored = o.batchScore(ctx, newJobsToScore)
	return stats, nil
}

// scoringPriority maps the resume match count to the queue priority: more
// matches → lower value → scored earlier (preserves the CP6 ordering).
func scoringPriority(matches int) int {
	prio := 100 - matches
	if prio < 1 {
		prio = 1
	}
	return prio
}

// filterResumeDecision applies the resume filter, preferring the deterministic
// engine and falling back to the legacy heuristic when unavailable.
func (o *Orchestrator) filterResumeDecision(ctx context.Context, title, description string, resumeKeywords []string, curriculumMD string, cfg Config) (bool, []string) {
	if !cfg.ResumeFilter.Enabled || len(resumeKeywords) == 0 {
		return true, nil
	}
	if o.resumeFilterer != nil && curriculumMD != "" {
		policy := filtering.Policy{
			MinMatches: cfg.ResumeFilter.MinMatches,
			MatchMode:  cfg.ResumeFilter.MatchMode,
		}
		if result, err := o.resumeFilterer.FilterJob(ctx, 0, resumeKeywords, curriculumMD, title, description, policy); err == nil {
			return result.Accepted, result.MatchedTerms
		} else {
			o.logger.Warn("orchestrator: deterministic filter failed, using legacy", "err", err)
		}
	}
	return FilterResume(title, description, resumeKeywords, cfg.ResumeFilter.MinMatches, cfg.ResumeFilter.MatchMode)
}

func (o *Orchestrator) deterministicScoringActive(ctx context.Context) bool {
	return o.deterministicScorer != nil && !aiusage.Load(ctx, o.repo).Score
}

func (o *Orchestrator) batchScore(ctx context.Context, jobs []jobToScore) int {
	if len(jobs) == 0 {
		return 0
	}

	curriculum, err := o.repo.LatestCurriculum(ctx)
	if err != nil || curriculum.ContentMD == "" {
		return 0
	}

	if o.deterministicScoringActive(ctx) {
		scoredCount := 0
		for _, j := range jobs {
			select {
			case <-ctx.Done():
				return scoredCount
			default:
			}
			score, reason, err := o.deterministicScorer(ctx, curriculum.ContentMD, j.Title, j.Description)
			if err != nil {
				o.logger.Warn("orchestrator: deterministic scoring failed for job", "job_id", j.ID, "err", err)
				continue
			}
			if err := o.repo.SetJobScore(ctx, j.ID, score, reason); err == nil {
				scoredCount++
			}
		}
		return scoredCount
	}

	if o.llmClient == nil {
		return 0
	}

	// Check if BYOK is configured
	apiKeySealed, err := o.repo.GetSetting(ctx, "llm.api_key")
	if err != nil || apiKeySealed == "" {
		return 0
	}
	apiKey, err := store.DecryptString(o.masterKey, apiKeySealed)
	if err != nil || apiKey == "" {
		return 0
	}
	baseURL, _ := o.repo.GetSetting(ctx, "llm.base_url")
	model, _ := o.repo.GetSetting(ctx, "llm.model")
	llmCfg := llm.Config{
		BaseURL: baseURL,
		Model:   model,
		APIKey:  apiKey,
	}.Normalize()

	scoredCount := 0
	for _, j := range jobs {
		select {
		case <-ctx.Done():
			return scoredCount
		default:
		}

		res, err := score.Score(ctx, o.llmClient, llmCfg, curriculum.ContentMD, j.Description)
		if err != nil {
			o.logger.Warn("orchestrator: scoring failed for job", "job_id", j.ID, "err", err)
			continue
		}

		if err := o.repo.SetJobScore(ctx, j.ID, res.Score, res.Reason); err == nil {
			scoredCount++
		}

		// Gentle rate limit delay between LLM calls (~300ms)
		time.Sleep(300 * time.Millisecond)
	}

	return scoredCount
}

// ConfigureTicker starts, stops or adjusts the automated discovery ticker.
func (o *Orchestrator) ConfigureTicker(intervalStr string) {
	o.tickerMu.Lock()
	defer o.tickerMu.Unlock()

	// Stop existing ticker if running
	if o.ticker != nil {
		o.ticker.Stop()
		o.ticker = nil
	}
	if o.tickerStop != nil {
		close(o.tickerStop)
		o.tickerStop = nil
	}

	intervalStr = strings.ToLower(strings.TrimSpace(intervalStr))
	if intervalStr == "" || intervalStr == "off" {
		return
	}

	var d time.Duration
	switch intervalStr {
	case "1h":
		d = 1 * time.Hour
	case "6h":
		d = 6 * time.Hour
	case "24h":
		d = 24 * time.Hour
	default:
		parsed, err := time.ParseDuration(intervalStr)
		if err != nil || parsed < 30*time.Minute {
			return
		}
		d = parsed
	}

	o.ticker = time.NewTicker(d)
	o.tickerStop = make(chan struct{})

	go func(t *time.Ticker, stop chan struct{}) {
		for {
			select {
			case <-stop:
				return
			case <-t.C:
				o.logger.Info("orchestrator: starting scheduled discovery run")
				_, _, _ = o.Discover(context.Background())
			}
		}
	}(o.ticker, o.tickerStop)
}

// Stop closes the background ticker.
func (o *Orchestrator) Stop() {
	o.ConfigureTicker("off")
}
