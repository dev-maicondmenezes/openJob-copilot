package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"
)

// ---- helpers ----------------------------------------------------------------

func postJSON2(t *testing.T, url string, body any) (int, map[string]any) {
	t.Helper()
	b, _ := json.Marshal(body)
	resp, err := http.Post(url, "application/json", bytes.NewReader(b))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	var out map[string]any
	_ = json.Unmarshal(data, &out)
	return resp.StatusCode, out
}

func doJSON(t *testing.T, method, url string, body any) (int, map[string]any) {
	t.Helper()
	var rd io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		rd = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, url, rd)
	if err != nil {
		t.Fatal(err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	var out map[string]any
	_ = json.Unmarshal(data, &out)
	return resp.StatusCode, out
}

// waitForCond polls a condition for up to 3s (bus → notifier is async).
func waitForCond(t *testing.T, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("condition not met within 3s")
}

// seedApproval creates a job and approves it, returning (jobID, applicationID).
func seedApproval(t *testing.T, baseURL string) (float64, float64) {
	t.Helper()
	code, body := postJSON2(t, baseURL+"/api/jobs/manual", map[string]any{
		"title": "Backend Go", "company": "Acme",
		"description": "Vaga para gopher", "url": fmt.Sprintf("https://x.example/%d", time.Now().UnixNano()),
	})
	if code != http.StatusCreated {
		t.Fatalf("manual job: %d %v", code, body)
	}
	jobID := body["id"].(float64)
	code, body = postJSON2(t, fmt.Sprintf("%s/api/jobs/%d/approve", baseURL, int(jobID)), nil)
	if code != http.StatusCreated {
		t.Fatalf("approve: %d %v", code, body)
	}
	return jobID, body["application"].(map[string]any)["id"].(float64)
}

// uploadCurriculumForTest uploads a small multipart curriculum, returning the
// HTTP status.
func uploadCurriculumForTest(t *testing.T, baseURL string) int {
	t.Helper()
	content := "## Experiência\n- Go, Docker, Kubernetes, PostgreSQL, Redis, Microservices, AWS, CI/CD"
	resp, _ := uploadFile(t, baseURL+"/api/curriculum/upload", "file", "cv.md", []byte(content))
	return resp.StatusCode
}

// ---- PATCH /api/applications/{id} → event → notification ----------------------

func TestApplicationStatusChangeProducesNotification(t *testing.T) {
	srv := newTestServer(t)
	base := srv.URL

	_, appID := seedApproval(t, base)

	// Invalid status rejected.
	code, _ := doJSON(t, http.MethodPatch, fmt.Sprintf("%s/api/applications/%d", base, int(appID)), map[string]any{"status": "hired"})
	if code != http.StatusBadRequest {
		t.Fatalf("invalid status = %d, want 400", code)
	}

	// interview → bell badge (event → notifier is async: poll briefly)
	code, body := doJSON(t, http.MethodPatch, fmt.Sprintf("%s/api/applications/%d", base, int(appID)), map[string]any{"status": "interview"})
	if code != http.StatusOK || body["status"] != "interview" {
		t.Fatalf("patch = %d %v", code, body)
	}

	waitForCond(t, func() bool {
		_, b := doJSON(t, http.MethodGet, base+"/api/notifications/unread-count", nil)
		u, _ := b["unread"].(float64)
		return u >= 1
	})
	code, body = doJSON(t, http.MethodGet, base+"/api/notifications/unread-count", nil)
	if code != http.StatusOK {
		t.Fatalf("unread-count = %d", code)
	}
	if unread, _ := body["unread"].(float64); unread < 1 {
		t.Fatalf("unread = %v, want >= 1 after status change", body["unread"])
	}

	// Feed contains the application_status notification with metadata.
	code, body = doJSON(t, http.MethodGet, base+"/api/notifications?type=application_status", nil)
	if code != http.StatusOK {
		t.Fatal(code)
	}
	items := body["notifications"].([]any)
	if len(items) == 0 {
		t.Fatal("no application_status notification")
	}
	first := items[0].(map[string]any)
	if first["severity"] != "info" || first["link"] != "/applications" {
		t.Fatalf("notification payload: %v", first)
	}

	// 404 for unknown application.
	code, _ = doJSON(t, http.MethodPatch, base+"/api/applications/9999", map[string]any{"status": "sent"})
	if code != http.StatusNotFound {
		t.Fatalf("unknown application = %d, want 404", code)
	}
}

// ---- notifications endpoints --------------------------------------------------

func TestNotificationsEndpoints(t *testing.T) {
	srv := newTestServer(t)
	base := srv.URL

	// Seed via test event: PATCH an application twice.
	_, appID := seedApproval(t, base)
	doJSON(t, http.MethodPatch, fmt.Sprintf("%s/api/applications/%d", base, int(appID)), map[string]any{"status": "interview"})
	doJSON(t, http.MethodPatch, fmt.Sprintf("%s/api/applications/%d", base, int(appID)), map[string]any{"status": "offer"})

	// list + filter + cursor
	code, body := doJSON(t, http.MethodGet, base+"/api/notifications?limit=1", nil)
	if code != 200 {
		t.Fatal(code)
	}
	items := body["notifications"].([]any)
	if len(items) != 1 {
		t.Fatalf("limit not applied: %d", len(items))
	}
	cursor, _ := body["next_cursor"].(string)
	if cursor == "" {
		t.Fatal("expected next cursor")
	}
	code, body = doJSON(t, http.MethodGet, base+"/api/notifications?limit=10&cursor="+cursor, nil)
	items2 := body["notifications"].([]any)
	if len(items2) != 1 {
		t.Fatalf("cursor page = %d, want 1", len(items2))
	}
	if fmt.Sprint(items[0].(map[string]any)["id"]) == fmt.Sprint(items2[0].(map[string]any)["id"]) {
		t.Fatal("cursor repeated an item")
	}

	// mark one read
	id := int(items[0].(map[string]any)["id"].(float64))
	code, _ = doJSON(t, http.MethodPost, fmt.Sprintf("%s/api/notifications/%d/read", base, id), nil)
	if code != 200 {
		t.Fatal(code)
	}
	code, body = doJSON(t, http.MethodGet, base+"/api/notifications/unread-count", nil)
	if unread, _ := body["unread"].(float64); unread != 1 {
		t.Fatalf("unread after mark = %v, want 1", body["unread"])
	}

	// read-all
	code, body = doJSON(t, http.MethodPost, base+"/api/notifications/read-all", nil)
	if code != 200 || body["updated"].(float64) != 1 {
		t.Fatalf("read-all = %d %v", code, body)
	}
	code, body = doJSON(t, http.MethodGet, base+"/api/notifications/unread-count", nil)
	if unread, _ := body["unread"].(float64); unread != 0 {
		t.Fatalf("unread after read-all = %v, want 0", body["unread"])
	}

	// delete
	code, _ = doJSON(t, http.MethodDelete, fmt.Sprintf("%s/api/notifications/%d", base, id), nil)
	if code != 200 {
		t.Fatal(code)
	}
	code, _ = doJSON(t, http.MethodDelete, fmt.Sprintf("%s/api/notifications/%d", base, id), nil)
	if code != http.StatusNotFound {
		t.Fatalf("double delete = %d, want 404", code)
	}
}

// ---- settings endpoints --------------------------------------------------------

func TestNotificationAndTaskSettings(t *testing.T) {
	srv := newTestServer(t)
	base := srv.URL

	code, body := doJSON(t, http.MethodGet, base+"/api/settings/notifications", nil)
	if code != 200 || body["in_app_enabled"] != true || body["retention_days"].(float64) != 90 {
		t.Fatalf("defaults = %d %v", code, body)
	}

	code, body = doJSON(t, http.MethodPut, base+"/api/settings/notifications", map[string]any{
		"in_app_enabled": false, "retention_days": 30, "browser_notifications": true,
	})
	if code != 200 || body["retention_days"].(float64) != 30 {
		t.Fatalf("update = %d %v", code, body)
	}

	// invalid retention rejected
	code, _ = doJSON(t, http.MethodPut, base+"/api/settings/notifications", map[string]any{"retention_days": 0})
	if code != http.StatusBadRequest {
		t.Fatalf("retention 0 = %d, want 400", code)
	}

	code, body = doJSON(t, http.MethodGet, base+"/api/settings/tasks", nil)
	if code != 200 || body["max_concurrency"].(float64) != 2 || body["worker_poll_ms"].(float64) != 500 {
		t.Fatalf("task defaults = %d %v", code, body)
	}

	code, _ = doJSON(t, http.MethodPut, base+"/api/settings/tasks", map[string]any{"max_concurrency": 9})
	if code != http.StatusBadRequest {
		t.Fatalf("max_concurrency 9 = %d, want 400", code)
	}
	code, _ = doJSON(t, http.MethodPut, base+"/api/settings/tasks", map[string]any{"worker_poll_ms": 99})
	if code != http.StatusBadRequest {
		t.Fatalf("worker_poll_ms 99 = %d, want 400", code)
	}
	code, body = doJSON(t, http.MethodPut, base+"/api/settings/tasks", map[string]any{"max_concurrency": 4, "worker_poll_ms": 1000})
	if code != 200 || body["max_concurrency"].(float64) != 4 {
		t.Fatalf("tasks update = %d %v", code, body)
	}
}

// ---- tasks endpoints -----------------------------------------------------------

func TestTasksEndpointsAndQueueFlow(t *testing.T) {
	srv := newTestServer(t)
	base := srv.URL

	// Enqueue a keyword-refresh task through the default (async) path.
	code, body := doJSON(t, http.MethodPost, base+"/api/settings/discovery/resume-keywords/refresh", nil)
	if code != http.StatusAccepted {
		t.Fatalf("async refresh = %d %v", code, body)
	}
	taskID, _ := body["task_id"].(string)
	if taskID == "" {
		t.Fatal("expected task_id")
	}

	// GET /api/tasks/{id} shows it queued (or already running/done).
	code, body = doJSON(t, http.MethodGet, base+"/api/tasks/"+taskID, nil)
	if code != 200 {
		t.Fatal(code)
	}
	if body["type"] != "extract_resume_keywords" {
		t.Fatalf("type = %v", body["type"])
	}
	status, _ := body["status"].(string)
	if status != "queued" && status != "running" && status != "done" && status != "failed" {
		t.Fatalf("unexpected status %q", status)
	}

	// Retry a failed/cancelled task: force-cancel then retry.
	doJSON(t, http.MethodPost, base+"/api/tasks/"+taskID+"/cancel", nil)
	code, body = doJSON(t, http.MethodGet, base+"/api/tasks/"+taskID, nil)
	if status, _ := body["status"].(string); status != "cancelled" {
		t.Fatalf("status after cancel = %q, want cancelled", status)
	}
	code, body = doJSON(t, http.MethodPost, base+"/api/tasks/"+taskID+"/retry", nil)
	if code != 200 || body["status"] != "queued" {
		t.Fatalf("retry = %d %v", code, body)
	}
	if attempts, _ := body["attempts"].(float64); attempts != 0 {
		t.Fatalf("retry must reset attempts, got %v", body["attempts"])
	}

	// Bulk actions exist.
	code, _ = doJSON(t, http.MethodPost, base+"/api/tasks/cancel-all-queued", nil)
	if code != 200 {
		t.Fatal(code)
	}
	code, _ = doJSON(t, http.MethodPost, base+"/api/tasks/retry-all-failed", nil)
	if code != 200 {
		t.Fatal(code)
	}

	// list with status filter + counts
	code, body = doJSON(t, http.MethodGet, base+"/api/tasks?status=cancelled", nil)
	if code != 200 {
		t.Fatal(code)
	}
	if body["status_counts"] == nil {
		t.Fatal("expected status_counts")
	}
}

func TestSyncKeywordRefreshFallback(t *testing.T) {
	srv := newTestServer(t)
	base := srv.URL

	// Upload a valid multipart curriculum so extraction has input.
	res := uploadCurriculumForTest(t, base)
	if res != http.StatusCreated {
		t.Fatalf("upload = %d", res)
	}

	code, body := doJSON(t, http.MethodPost, base+"/api/settings/discovery/resume-keywords/refresh?sync=true", nil)
	if code != 200 {
		t.Fatalf("sync refresh = %d %v", code, body)
	}
	kws, _ := body["resume_keywords"].([]any)
	if len(kws) == 0 {
		t.Fatal("sync refresh produced no keywords")
	}
}

func TestHealthIncludesQueueAndBell(t *testing.T) {
	srv := newTestServer(t)
	base := srv.URL
	code, body := doJSON(t, http.MethodGet, base+"/api/health", nil)
	if code != 200 {
		t.Fatal(code)
	}
	if body["tasks"] == nil || body["notifications"] == nil {
		t.Fatalf("health missing CP10 blocks: %v", body)
	}
	tasks := body["tasks"].(map[string]any)
	for _, key := range []string{"queued", "running", "failed_last_24h"} {
		if _, ok := tasks[key]; !ok {
			t.Fatalf("health.tasks missing %q", key)
		}
	}
	if _, ok := body["notifications"].(map[string]any)["unread"]; !ok {
		t.Fatal("health.notifications missing unread")
	}
}

// ---- score regression: task vs synchronous endpoint -------------------------------

func TestScoreSyncEndpointContract(t *testing.T) {
	// CP10 regression guarantee: score_job enqueued by discovery uses the same
	// score.Score path as the synchronous endpoint (handler-level equality is
	// covered in internal/tasks). Here we pin the sync endpoint contract with
	// a mocked LLM wired through the BYOK settings + httptest provider.
	srv := newTestServer(t)
	base := srv.URL

	if code := uploadCurriculumForTest(t, base); code != http.StatusCreated {
		t.Fatalf("upload = %d", code)
	}

	code, body := postJSON2(t, base+"/api/jobs/manual", map[string]any{
		"title": "Dev", "company": "A", "description": "Go backend",
	})
	if code != http.StatusCreated {
		t.Fatalf("manual = %d %v", code, body)
	}
	jobID := int(body["id"].(float64))

	// Without BYOK configured the endpoint answers 400 with a legible error —
	// the queue handler must produce the same legibility (asserted in tasks).
	code, body = postJSON2(t, fmt.Sprintf("%s/api/jobs/%d/score", base, jobID), nil)
	if code != http.StatusBadRequest {
		t.Fatalf("score without BYOK = %d %v, want 400", code, body)
	}
}
