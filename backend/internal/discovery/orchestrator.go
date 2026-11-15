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

	"github.com/maicondmenezes/openjob/internal/llm"
	"github.com/maicondmenezes/openjob/internal/score"
	"github.com/maicondmenezes/openjob/internal/store"
)

const (
	SettingDiscoveryConfig  = "discovery_config"
	SettingResumeKeywords   = "resume_keywords"
	SettingResumeHash       = "resume_hash"
	maxBatchScoringPerRun   = 15
)

// Orchestrator coordinates vacancy discovery across all enabled sources,
// executing keyword and resume filters, deduplicating, and batch scoring.
type Orchestrator struct {
	repo       store.Repository
	registry   *Registry
	masterKey  []byte
	llmClient  llm.Client
	logger     *slog.Logger

	mu           sync.Mutex
	running      bool
	currentRunID int64

	tickerMu     sync.Mutex
	ticker       *time.Ticker
	tickerStop   chan struct{}
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
	}
}

// Registry returns the underlying adapter registry.
func (o *Orchestrator) Registry() *Registry {
	return o.registry
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
	o.mu.Lock()
	if o.running {
		runID := o.currentRunID
		o.mu.Unlock()
		return runID, RunStats{}, errors.New("já existe uma busca em execução")
	}
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

	sources, err := o.repo.ListSources(ctx)
	if err != nil {
		return stats, fmt.Errorf("list sources: %w", err)
	}

	var allFetched []DiscoveredJob

	// Fetch from each enabled source
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

		// Decrypt auth key if present
		apiKey := ""
		if src.AuthType == "api_key" && src.AuthConfig != "" {
			if decrypted, err := store.DecryptString(o.masterKey, src.AuthConfig); err == nil {
				apiKey = decrypted
			}
		}

		var caps Capabilities
		if err := json.Unmarshal([]byte(src.Capabilities), &caps); err != nil {
			caps = adapter.DefaultCapabilities()
		}

		srcCfg := SourceConfig{
			ID:           src.ID,
			Name:         src.Name,
			Adapter:      src.Adapter,
			Enabled:      src.Enabled,
			Priority:     src.Priority,
			AuthType:     src.AuthType,
			APIKey:       apiKey,
			Capabilities: caps,
		}

		// Fetch with fault tolerance: single source error must not break the whole run
		var srcKeywords []string
		if caps.SupportsKeywordSearch {
			srcKeywords = cfg.Keywords
		}

		jobs, fetchErr := adapter.Fetch(ctx, srcCfg, srcKeywords)
		stat := SourceRunStat{
			SourceID: src.ID,
			Adapter:  src.Adapter,
			Fetched:  len(jobs),
		}

		if fetchErr != nil {
			errMsg := fmt.Sprintf("%s (%s): %v", src.Name, src.ID, fetchErr)
			o.logger.Error("orchestrator: source fetch failed", "source", src.ID, "err", fetchErr)
			stat.Error = fetchErr.Error()
			stats.Errors = append(stats.Errors, errMsg)
		} else {
			allFetched = append(allFetched, jobs...)
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
		resMatch := true
		var matchedResume []string
		if cfg.ResumeFilter.Enabled && len(resumeKeywords) > 0 {
			resMatch, matchedResume = FilterResume(job.Title, job.Description, resumeKeywords, cfg.ResumeFilter.MinMatches, cfg.ResumeFilter.MatchMode)
		}
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

	if len(newJobsToScore) > maxBatchScoringPerRun {
		newJobsToScore = newJobsToScore[:maxBatchScoringPerRun]
	}

	stats.Scored = o.batchScore(ctx, newJobsToScore)
	return stats, nil
}

func (o *Orchestrator) batchScore(ctx context.Context, jobs []jobToScore) int {
	if len(jobs) == 0 || o.llmClient == nil {
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

	curriculum, err := o.repo.LatestCurriculum(ctx)
	if err != nil || curriculum.ContentMD == "" {
		return 0
	}

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
