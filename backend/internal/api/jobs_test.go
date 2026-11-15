package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
)

// ---- POST /api/jobs/manual -------------------------------------------------

func TestCreateManualJob(t *testing.T) {
	srv := newTestServer(t)

	resp, out := postJSON(t, srv.URL+"/api/jobs/manual", map[string]string{
		"title":       "Go Developer Sênior",
		"company":     "Acme Corp",
		"url":         "https://jobs.example.com/go-senior",
		"description": "<p>5+ anos Go</p> <li>Kubernetes</li>",
	})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("status = %d, want 201 (body %v)", resp.StatusCode, out)
	}
	if out["source"] != "manual" || out["status"] != "new" {
		t.Errorf("got source=%v status=%v, want manual/new", out["source"], out["status"])
	}
	if id, ok := out["id"].(float64); !ok || id < 1 {
		t.Errorf("expected numeric id, got %v", out["id"])
	}
}

func TestCreateManualJobValidation(t *testing.T) {
	srv := newTestServer(t)

	cases := []struct {
		name    string
		payload map[string]string
	}{
		{"missing title", map[string]string{"company": "A", "description": "JD"}},
		{"missing company", map[string]string{"title": "T", "description": "JD"}},
		{"missing description", map[string]string{"title": "T", "company": "A"}},
		{"bad url scheme", map[string]string{"title": "T", "company": "A", "description": "JD", "url": "ftp://x.com"}},
		{"url without host", map[string]string{"title": "T", "company": "A", "description": "JD", "url": "not a url"}},
	}
	for _, tc := range cases {
		resp, out := postJSON(t, srv.URL+"/api/jobs/manual", tc.payload)
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("%s: status = %d, want 400 (body %v)", tc.name, resp.StatusCode, out)
		}
	}
}

// Critical (CP5): pasting the same JD with the same URL twice → dedup error.
func TestCreateManualJobURLDedup(t *testing.T) {
	srv := newTestServer(t)

	payload := map[string]string{
		"title":       "Vaga Duplicada",
		"company":     "Acme",
		"url":         "https://jobs.example.com/dup",
		"description": "JD",
	}
	resp, _ := postJSON(t, srv.URL+"/api/jobs/manual", payload)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("first insert status = %d, want 201", resp.StatusCode)
	}

	resp, out := postJSON(t, srv.URL+"/api/jobs/manual", payload)
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("duplicate url status = %d, want 409 (body %v)", resp.StatusCode, out)
	}
	if !strings.Contains(out["error"].(string), "URL") {
		t.Errorf("expected dedup error mentioning URL, got %v", out["error"])
	}
}

// Jobs without a URL must never collide (URL is stored as NULL in SQLite).
func TestCreateManualJobMultipleWithoutURL(t *testing.T) {
	srv := newTestServer(t)

	for i := 1; i <= 2; i++ {
		resp, out := postJSON(t, srv.URL+"/api/jobs/manual", map[string]string{
			"title":       fmt.Sprintf("Vaga sem link %d", i),
			"company":     "Acme",
			"description": "JD sem URL",
		})
		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("insert %d status = %d, want 201 (body %v)", i, resp.StatusCode, out)
		}
	}
}

// ---- POST /api/jobs/:id/score ------------------------------------------------

func TestScoreJobPersistsScore(t *testing.T) {
	mock := &mockLLM{response: `{"score": 85, "reason": "Forte aderência em Go e k8s."}`}
	srv := newTestServerWithLLM(t, mock)

	// Setup: curriculum + LLM key + manual job
	uploadFile(t, srv.URL+"/api/curriculum/upload", "file", "cv.md", []byte("# João\nDev Go"))
	postJSON(t, srv.URL+"/api/settings/llm-key", map[string]string{"api_key": "sk-test"})
	_, job := postJSON(t, srv.URL+"/api/jobs/manual", map[string]string{
		"title":       "Go Dev",
		"company":     "Acme",
		"url":         "https://jobs.example.com/go",
		"description": "<p>5 anos de Go</p>",
	})
	jobID := int(job["id"].(float64))

	resp, out := postJSON(t, srv.URL+"/api/jobs/"+strconv.Itoa(jobID)+"/score", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("score status = %d, want 200 (body %v)", resp.StatusCode, out)
	}
	if out["score_fit"] != float64(85) {
		t.Errorf("score_fit = %v, want 85", out["score_fit"])
	}
	if out["score_reason"] != "Forte aderência em Go e k8s." {
		t.Errorf("score_reason = %v", out["score_reason"])
	}

	// Score must persist: GET /api/jobs shows it.
	jobs := readJobsResponse(t, srv)
	if len(jobs) != 1 || jobs[0]["score_fit"] != float64(85) {
		t.Fatalf("score did not persist: %v", jobs)
	}
}

func TestScoreJobErrors(t *testing.T) {
	mock := &mockLLM{response: "não sei, sem json"}
	srv := newTestServerWithLLM(t, mock)
	uploadFile(t, srv.URL+"/api/curriculum/upload", "file", "cv.md", []byte("# João\nDev Go"))
	postJSON(t, srv.URL+"/api/settings/llm-key", map[string]string{"api_key": "sk-test"})
	_, job := postJSON(t, srv.URL+"/api/jobs/manual", map[string]string{
		"title": "Go Dev", "company": "Acme", "description": "JD Go",
	})
	jobID := int(job["id"].(float64))

	// Unparseable model response → 502 and score stays NULL.
	resp, out := postJSON(t, srv.URL+"/api/jobs/"+strconv.Itoa(jobID)+"/score", nil)
	if resp.StatusCode != http.StatusBadGateway {
		t.Fatalf("status = %d, want 502 (body %v)", resp.StatusCode, out)
	}
	jobs := readJobsResponse(t, srv)
	if jobs[0]["score_fit"] != nil {
		t.Errorf("score_fit must stay NULL after parse failure, got %v", jobs[0]["score_fit"])
	}

	// Without LLM key → 400 with a clear message.
	srvNoKey := newTestServerWithLLM(t, &mockLLM{response: `{"score": 80, "reason": "ok"}`})
	uploadFile(t, srvNoKey.URL+"/api/curriculum/upload", "file", "cv.md", []byte("# João\nDev Go"))
	_, job2 := postJSON(t, srvNoKey.URL+"/api/jobs/manual", map[string]string{
		"title": "Go Dev", "company": "Acme", "description": "JD Go",
	})
	resp, out = postJSON(t, srvNoKey.URL+"/api/jobs/"+strconv.Itoa(int(job2["id"].(float64)))+"/score", nil)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("no-key status = %d, want 400 (body %v)", resp.StatusCode, out)
	}
	if !strings.Contains(out["error"].(string), "Provedor de IA não configurado") {
		t.Errorf("expected clear config error, got %v", out["error"])
	}
}

func TestScoreJobWithoutCurriculum(t *testing.T) {
	srv := newTestServer(t)
	_, job := postJSON(t, srv.URL+"/api/jobs/manual", map[string]string{
		"title": "Go Dev", "company": "Acme", "description": "JD Go",
	})
	resp, out := postJSON(t, srv.URL+"/api/jobs/"+strconv.Itoa(int(job["id"].(float64)))+"/score", nil)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 (body %v)", resp.StatusCode, out)
	}
	if !strings.Contains(out["error"].(string), "currículo") {
		t.Errorf("expected curriculum error, got %v", out["error"])
	}
}

// ---- POST /api/jobs/:id/approve ----------------------------------------------

func TestApproveJobCreatesPendingApplication(t *testing.T) {
	srv := newTestServer(t)
	_, job := postJSON(t, srv.URL+"/api/jobs/manual", map[string]string{
		"title": "Go Dev", "company": "Acme", "description": "JD Go",
	})
	jobID := int(job["id"].(float64))

	resp, out := postJSON(t, srv.URL+"/api/jobs/"+strconv.Itoa(jobID)+"/approve", nil)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("approve status = %d, want 201 (body %v)", resp.StatusCode, out)
	}
	app := out["application"].(map[string]any)
	if app["status"] != "pending" || app["job_id"] != float64(jobID) {
		t.Errorf("application = %v, want pending/job %d", app, jobID)
	}

	// Second approval is blocked (one active application per job).
	resp, out = postJSON(t, srv.URL+"/api/jobs/"+strconv.Itoa(jobID)+"/approve", nil)
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("duplicate approve status = %d, want 409 (body %v)", resp.StatusCode, out)
	}

	// The application appears in GET /api/applications as pending.
	apps := readApplicationsResponse(t, srv)
	if len(apps) != 1 || apps[0]["status"] != "pending" {
		t.Fatalf("expected 1 pending application, got %v", apps)
	}
	jobData := apps[0]["job"].(map[string]any)
	if jobData["title"] != "Go Dev" || jobData["company"] != "Acme" {
		t.Errorf("application job data = %v", jobData)
	}
}

func TestApproveUnknownJob(t *testing.T) {
	srv := newTestServer(t)
	resp, _ := postJSON(t, srv.URL+"/api/jobs/999/approve", nil)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", resp.StatusCode)
	}
}

// ---- POST /api/jobs/:id/discard ----------------------------------------------

func TestDiscardJob(t *testing.T) {
	srv := newTestServer(t)
	_, job := postJSON(t, srv.URL+"/api/jobs/manual", map[string]string{
		"title": "Go Dev", "company": "Acme", "description": "JD Go",
	})
	jobID := int(job["id"].(float64))

	resp, out := postJSON(t, srv.URL+"/api/jobs/"+strconv.Itoa(jobID)+"/discard", nil)
	if resp.StatusCode != http.StatusOK || out["status"] != "discarded" {
		t.Fatalf("discard = %d %v, want 200/discarded", resp.StatusCode, out)
	}

	jobs := readJobsResponse(t, srv)
	if jobs[0]["status"] != "discarded" {
		t.Errorf("status = %v, want discarded", jobs[0]["status"])
	}
}

// ---- GET /api/jobs ordering ----------------------------------------------------

func TestListJobsSortedByScoreNullsLast(t *testing.T) {
	srv := newTestServerWithLLM(t, &mockLLM{response: `{"score": 60, "reason": "ok"}`})
	uploadFile(t, srv.URL+"/api/curriculum/upload", "file", "cv.md", []byte("# João\nDev Go"))
	postJSON(t, srv.URL+"/api/settings/llm-key", map[string]string{"api_key": "sk-test"})

	postJSON(t, srv.URL+"/api/jobs/manual", map[string]string{
		"title": "Sem score", "company": "A", "description": "JD",
	})
	_, job2 := postJSON(t, srv.URL+"/api/jobs/manual", map[string]string{
		"title": "Com score", "company": "B", "description": "JD",
	})
	resp, _ := postJSON(t, srv.URL+"/api/jobs/"+strconv.Itoa(int(job2["id"].(float64)))+"/score", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("score setup failed: %d", resp.StatusCode)
	}

	jobs := readJobsResponse(t, srv)
	if len(jobs) != 2 {
		t.Fatalf("len(jobs) = %d, want 2", len(jobs))
	}
	if jobs[0]["title"] != "Com score" {
		t.Errorf("scored job should come first, got %v", jobs[0]["title"])
	}
	if jobs[1]["score_fit"] != nil {
		t.Errorf("unscored job last, got %v", jobs[1])
	}
}

// ---- GET /api/jobs ordering ----------------------------------------------------

func TestFetchJobPrefillsDraft(t *testing.T) {
	srv := newTestServer(t)
	page := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(jobPageHTML))
	}))
	defer page.Close()

	resp, out := postJSON(t, srv.URL+"/api/jobs/fetch", map[string]string{
		"url": page.URL + "/vagas/go-dev",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("fetch status = %d, want 200 (body %v)", resp.StatusCode, out)
	}
	if out["title"] != "Senior Go Developer" {
		t.Errorf("title = %v, want Senior Go Developer", out["title"])
	}
	if out["company"] != "Acme Corp" {
		t.Errorf("company = %v, want Acme Corp", out["company"])
	}
	desc, _ := out["description"].(string)
	if !strings.Contains(desc, "5+ anos de Go") || strings.Contains(desc, "<") {
		t.Errorf("description should be cleaned text, got %q", desc)
	}
	if out["exists"] != false {
		t.Errorf("exists = %v, want false", out["exists"])
	}

	// After registering the URL, fetch reports exists=true.
	_, _ = postJSON(t, srv.URL+"/api/jobs/manual", map[string]string{
		"title": "Go Dev", "company": "Acme",
		"url": page.URL + "/vagas/go-dev", "description": "JD",
	})
	resp, out = postJSON(t, srv.URL+"/api/jobs/fetch", map[string]string{
		"url": page.URL + "/vagas/go-dev",
	})
	if resp.StatusCode != http.StatusOK || out["exists"] != true {
		t.Errorf("exists flag after registration = %v (status %d)", out["exists"], resp.StatusCode)
	}
}

func TestFetchJobValidationAndErrors(t *testing.T) {
	srv := newTestServer(t)

	// Invalid URL → 400
	resp, _ := postJSON(t, srv.URL+"/api/jobs/fetch", map[string]string{"url": "notaurl"})
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("bad url status = %d, want 400", resp.StatusCode)
	}

	// Empty URL → 400
	resp, _ = postJSON(t, srv.URL+"/api/jobs/fetch", map[string]string{"url": "  "})
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("empty url status = %d, want 400", resp.StatusCode)
	}

	// Unreachable host → 502 with legible message
	resp, out := postJSON(t, srv.URL+"/api/jobs/fetch", map[string]string{
		"url": "http://127.0.0.1:1/nope",
	})
	if resp.StatusCode != http.StatusBadGateway {
		t.Errorf("unreachable status = %d, want 502 (body %v)", resp.StatusCode, out)
	}
	if !strings.Contains(out["error"].(string), "Falha ao buscar") {
		t.Errorf("expected legible fetch error, got %v", out["error"])
	}

	// Non-HTML content → 502
	page := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/pdf")
		_, _ = w.Write([]byte("%PDF-1.4"))
	}))
	defer page.Close()
	resp, out = postJSON(t, srv.URL+"/api/jobs/fetch", map[string]string{"url": page.URL})
	if resp.StatusCode != http.StatusBadGateway || !strings.Contains(out["error"].(string), "HTML") {
		t.Errorf("non-html = %d %v, want 502 mentioning HTML", resp.StatusCode, out)
	}
}

// jobPageHTML mimics an ATS (Greenhouse-style) posting with JSON-LD.
const jobPageHTML = `<!DOCTYPE html>
<html><head>
<title>Senior Go Developer — Acme Corp</title>
<script type="application/ld+json">
{"@context":"https://schema.org","@type":"JobPosting",
 "title":"Senior Go Developer",
 "description":"<p>Vaga remota.</p><ul><li>5+ anos de Go</li><li>Kubernetes</li></ul>",
 "hiringOrganization":{"name":"Acme Corp"}}
</script>
</head>
<body><nav>Menu Início Vagas</nav>
<div class="content"><p>Requisitos: Go, Docker.</p></div>
<footer>Rodapé</footer></body></html>`

// ---- helpers --------------------------------------------------------------------

func getJSON(t *testing.T, url string) []map[string]any {
	t.Helper()
	resp, err := http.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET %s = %d, want 200", url, resp.StatusCode)
	}
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	var wrapper map[string]json.RawMessage
	if err := json.Unmarshal(raw, &wrapper); err != nil {
		t.Fatal(err)
	}
	var key string
	switch {
	case strings.Contains(string(raw), `"jobs":`):
		key = "jobs"
	default:
		key = "applications"
	}
	var items []map[string]any
	if err := json.Unmarshal(wrapper[key], &items); err != nil {
		t.Fatal(err)
	}
	return items
}

func readJobsResponse(t *testing.T, srv *httptest.Server) []map[string]any {
	return getJSON(t, srv.URL+"/api/jobs")
}

func readApplicationsResponse(t *testing.T, srv *httptest.Server) []map[string]any {
	return getJSON(t, srv.URL+"/api/applications")
}
