// Package store owns persistence for openjob. SQL (including AUTOINCREMENT
// and any SQLite-specific syntax) is confined to this package so that handlers
// never see database dialect — a future swap to PostgreSQL (spec §4/§10.1)
// touches only the implementation, not the Repository interface.
package store

import (
	"context"
	"errors"
	"time"
)

// ErrNotFound is returned when a requested row does not exist.
var ErrNotFound = errors.New("store: not found")

// Curriculum is a base resume (the most recent row is the active one).
type Curriculum struct {
	ID             int64
	ContentMD      string
	SourceFilename string // empty when pasted directly
	UploadedAt     time.Time
}

// Job is a vacancy discovered by an automated source or pasted manually (spec §7).
type Job struct {
	ID                 int64
	Title              string
	Company            string
	URL                string // unique; empty when no link is available
	Description        string
	Source             string // 'remotive' | 'himalayas' | 'jobicy' | 'remoteok' | 'manual'
	ScoreFit           *int   // nil until scoring runs (CP5)
	ScoreReason        string
	Status             string // 'new' | 'approved' | 'discarded'
	ExternalID         string
	MatchedKeywords    []string
	MatchedResumeTerms []string
	CreatedAt          time.Time
}

// Source represents an external vacancy source (CP6).
type Source struct {
	ID           string // slug PK (e.g. "himalayas", "remotive")
	Name         string // display name
	Adapter      string // adapter slug registered in code
	Enabled      bool
	Priority     int
	AuthType     string // "none" | "api_key"
	AuthConfig   string // encrypted JSON (empty when auth_type == "none")
	Capabilities string // JSON string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// DiscoveryRun records an automated or manual discovery execution (CP6).
type DiscoveryRun struct {
	ID           int64
	StartedAt    time.Time
	CompletedAt  *time.Time
	Status       string // 'running' | 'completed' | 'failed'
	Stats        string // JSON with counters and per-source status
	ErrorMessage string
}

// Application tracks a candidacy after a job is approved.
type Application struct {
	ID                   int64
	JobID                int64
	TailoredCurriculumMD string
	Status               string // 'pending' | 'sent' | 'interview' | 'rejected' | 'offer'
	SentAt               *time.Time
	UpdatedAt            time.Time
}

// NotifyLog records a notification that was actually sent (spec §7).
type NotifyLog struct {
	ID      int64
	Type    string // 'daily_report' | 'status_change' | 'test'
	Payload string
	SentAt  time.Time
}

// Notification is an in-app notification (CP10): a persisted feed entry shown
// in the bell dropdown and on the /notifications page.
type Notification struct {
	ID        int64
	Type      string // 'application_status' | 'discovery_result' | 'task_completed' | 'task_failed' | 'source_error' | 'test'
	Severity  string // 'info' | 'success' | 'warning' | 'error'
	Title     string
	Body      string
	Link      string // internal SPA deep link (e.g. "/applications/42")
	Metadata  string // JSON, empty when absent
	ReadAt    *time.Time
	CreatedAt time.Time
}

// NotificationFilter selects which notifications a listing returns.
type NotificationFilter struct {
	UnreadOnly bool
	Type       string // empty = all types
	Severity   string // empty = all severities
	Limit      int
	Cursor     string // opaque cursor from the previous page
}

// Task is a queued asynchronous AI job (CP10). ID is a ULID so IDs sort by
// creation time; claims are atomic via UPDATE ... RETURNING (see tasks.go).
type Task struct {
	ID          string
	Type        string // 'score_job' | 'extract_resume_keywords' | ...
	Status      string // 'queued' | 'running' | 'done' | 'failed' | 'cancelled'
	Payload     string // JSON input
	Result      string // JSON output, empty until done
	Error       string // last legible error
	Progress    int    // 0-100
	Attempts    int
	MaxAttempts int
	Priority    int // lower runs first
	NextRunAt   *time.Time
	CreatedAt   time.Time
	StartedAt   *time.Time
	FinishedAt  *time.Time
}

// Repository is the single persistence boundary of openjob. It is defined up
// front with the methods foreseen for every table in spec §10.1 so that later
// checkpoints only add implementations, never new abstractions.
type Repository interface {
	Close() error

	// settings — simple key/value pairs (BYOK keys arrive already encrypted
	// by the caller; the store is ciphertext-agnostic).
	GetSetting(ctx context.Context, key string) (string, error)
	SetSetting(ctx context.Context, key, value string) error
	DeleteSetting(ctx context.Context, key string) error

	// curriculum
	SaveCurriculum(ctx context.Context, c Curriculum) (int64, error)
	LatestCurriculum(ctx context.Context) (Curriculum, error)

	// jobs
	CreateJob(ctx context.Context, j Job) (int64, error)
	InsertDiscoveredJob(ctx context.Context, j Job) (int64, bool, error) // dedup on url; returns (id, inserted, error)
	GetJob(ctx context.Context, id int64) (Job, error)
	ListJobs(ctx context.Context, status string) ([]Job, error) // empty status = all
	UpdateJobStatus(ctx context.Context, id int64, status string) error
	SetJobScore(ctx context.Context, id int64, score int, reason string) error
	JobURLExists(ctx context.Context, url string) (bool, error) // dedup (CP5/CP6)

	// sources (CP6)
	ListSources(ctx context.Context) ([]Source, error)
	GetSource(ctx context.Context, id string) (Source, error)
	CreateSource(ctx context.Context, s Source) error
	UpdateSource(ctx context.Context, s Source) error
	DeleteSource(ctx context.Context, id string) error

	// discovery_runs (CP6)
	CreateDiscoveryRun(ctx context.Context, r DiscoveryRun) (int64, error)
	UpdateDiscoveryRun(ctx context.Context, r DiscoveryRun) error
	GetDiscoveryRun(ctx context.Context, id int64) (DiscoveryRun, error)
	LatestDiscoveryRun(ctx context.Context) (DiscoveryRun, error)

	// applications
	CreateApplication(ctx context.Context, a Application) (int64, error)
	GetApplication(ctx context.Context, id int64) (Application, error)
	ListApplications(ctx context.Context) ([]Application, error)
	UpdateApplicationStatus(ctx context.Context, id int64, status string) error
	// ActiveApplicationForJob guards against duplicated approvals (CP5):
	// one open candidacy per job, ErrNotFound when the coast is clear.
	ActiveApplicationForJob(ctx context.Context, jobID int64) (Application, error)

	// notify_log
	InsertNotifyLog(ctx context.Context, n NotifyLog) (int64, error)
	ListNotifyLog(ctx context.Context, limit int) ([]NotifyLog, error)

	// notifications (CP10) — persisted in-app feed
	InsertNotification(ctx context.Context, n Notification) (int64, error)
	GetNotification(ctx context.Context, id int64) (Notification, error)
	ListNotifications(ctx context.Context, f NotificationFilter) ([]Notification, string, error) // returns next cursor ("" = end)
	MarkNotificationRead(ctx context.Context, id int64) error
	MarkAllNotificationsRead(ctx context.Context) (int64, error)
	DeleteNotification(ctx context.Context, id int64) error
	DeleteReadNotifications(ctx context.Context) (int64, error)
	DeleteReadNotificationsBefore(ctx context.Context, cutoff time.Time) (int64, error)
	CountUnreadNotifications(ctx context.Context) (int, error)

	// tasks (CP10) — durable queue for asynchronous AI jobs
	InsertTask(ctx context.Context, t Task) error
	ClaimNextTask(ctx context.Context) (Task, error) // atomic claim; ErrNotFound when the queue is empty
	GetTask(ctx context.Context, id string) (Task, error)
	ListTasks(ctx context.Context, statuses []string, typ, cursor string, limit int) ([]Task, string, error)
	UpdateTaskProgress(ctx context.Context, id string, progress int) error
	FinishTask(ctx context.Context, id, status, result, errMsg string) error
	RequeueTaskWithBackoff(ctx context.Context, id string, nextRunAt time.Time, errMsg string) error
	RetryTask(ctx context.Context, id string) error         // manual retry: resets attempts, clears error/result
	CancelQueuedTask(ctx context.Context, id, errMsg string) (bool, error)
	RequeueRunningTasks(ctx context.Context) (int64, error) // boot recovery for orphaned tasks
	CountTasksByStatus(ctx context.Context) (map[string]int, error)
	CountTasks(ctx context.Context, statuses []string, typ string) (int, error)
	CountTasksFailedSince(ctx context.Context, since time.Time) (int, error)
	CancelAllQueuedTasks(ctx context.Context, errMsg string) (int64, error) // bulk action
	RetryAllFailedTasks(ctx context.Context) (int64, error)                // bulk action
}
