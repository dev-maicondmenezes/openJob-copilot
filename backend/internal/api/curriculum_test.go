package api

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"strings"
	"testing"

	"github.com/maicondmenezes/openjob/internal/llm"
	"github.com/maicondmenezes/openjob/internal/tailor"
)

type mockLLM struct {
	response string
	err      error
}

func (m *mockLLM) Ping(ctx context.Context, cfg llm.Config) error {
	return nil
}

func (m *mockLLM) Complete(ctx context.Context, cfg llm.Config, req llm.CompletionRequest) (string, error) {
	return m.response, m.err
}

func uploadFile(t *testing.T, url, fieldName, filename string, content []byte) (*http.Response, map[string]any) {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile(fieldName, filename)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write(content); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}

	req, err := http.NewRequest(http.MethodPost, url, &body)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	out := map[string]any{}
	_ = json.NewDecoder(resp.Body).Decode(&out)
	return resp, out
}

func TestCurriculumGetEmpty(t *testing.T) {
	srv := newTestServer(t)

	resp, err := srv.Client().Get(srv.URL + "/api/curriculum")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}

	var body map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}

	if body["curriculum"] != nil {
		t.Fatalf("expected curriculum to be nil, got %v", body["curriculum"])
	}
}

func TestCurriculumUploadAndGet(t *testing.T) {
	srv := newTestServer(t)

	mdContent := []byte("# João Silva\nEngenheiro de Software\n\nExperiência\n• Desenvolveu microsserviços em Go")

	resp, out := uploadFile(t, srv.URL+"/api/curriculum/upload", "file", "cv.md", mdContent)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("upload status = %d, want 201 (body %v)", resp.StatusCode, out)
	}

	if out["source_filename"] != "cv.md" {
		t.Errorf("got source_filename %v, want cv.md", out["source_filename"])
	}
	content, _ := out["content_md"].(string)
	if !strings.Contains(content, "## Experiência") {
		t.Errorf("expected structured markdown, got: %s", content)
	}

	// Now GET /api/curriculum should return this curriculum
	resp, err := srv.Client().Get(srv.URL + "/api/curriculum")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	var getBody map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&getBody); err != nil {
		t.Fatal(err)
	}
	cur, ok := getBody["curriculum"].(map[string]any)
	if !ok || cur == nil {
		t.Fatalf("expected curriculum object, got %v", getBody)
	}
	if cur["source_filename"] != "cv.md" {
		t.Errorf("got %v, want cv.md", cur["source_filename"])
	}

	// Uploading a second file replaces the active one (most recent row is active)
	mdContent2 := []byte("# João Silva V2\nPleno")
	resp, _ = uploadFile(t, srv.URL+"/api/curriculum/upload", "file", "cv_v2.md", mdContent2)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("upload 2 status = %d, want 201", resp.StatusCode)
	}

	resp, err = srv.Client().Get(srv.URL + "/api/curriculum")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	_ = json.NewDecoder(resp.Body).Decode(&getBody)
	cur = getBody["curriculum"].(map[string]any)
	if cur["source_filename"] != "cv_v2.md" {
		t.Errorf("active curriculum should be cv_v2.md, got %v", cur["source_filename"])
	}
}

func TestCurriculumUploadValidation(t *testing.T) {
	srv := newTestServer(t)

	// Unsupported extension
	resp, out := uploadFile(t, srv.URL+"/api/curriculum/upload", "file", "bad.exe", []byte("content"))
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 for bad extension (body %v)", resp.StatusCode, out)
	}

	// Missing file field
	resp, out = uploadFile(t, srv.URL+"/api/curriculum/upload", "wrong_field", "cv.md", []byte("content"))
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 for wrong field (body %v)", resp.StatusCode, out)
	}
}

func TestCurriculumTailor(t *testing.T) {
	mock := &mockLLM{
		response: "# João Silva (Adaptado)\n## Experiência\n- Especialista em Go e Cloud",
	}
	srv := newTestServerWithLLM(t, mock)

	// 1. Tailor without curriculum -> clear 400 error
	resp, out := postJSON(t, srv.URL+"/api/curriculum/tailor", map[string]string{
		"jd_text": "Vaga Go Developer",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 when curriculum is missing, got %d", resp.StatusCode)
	}
	if !strings.Contains(out["error"].(string), "Nenhum currículo-base cadastrado") {
		t.Errorf("expected clear error message, got %v", out["error"])
	}

	// 2. Upload curriculum
	uploadFile(t, srv.URL+"/api/curriculum/upload", "file", "cv.md", []byte("# João\nDev Go"))

	// 3. Tailor without LLM key -> clear 400 error (NOT 500!)
	resp, out = postJSON(t, srv.URL+"/api/curriculum/tailor", map[string]string{
		"jd_text": "Vaga Go Developer",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 when LLM key is missing, got %d", resp.StatusCode)
	}
	if !strings.Contains(out["error"].(string), "Provedor de IA não configurado") {
		t.Errorf("expected clear LLM config error, got %v", out["error"])
	}

	// 4. Save LLM key
	postJSON(t, srv.URL+"/api/settings/llm-key", map[string]string{
		"api_key": "sk-test-key",
	})

	// 5. Tailor without job_id or jd_text -> 400
	resp, out = postJSON(t, srv.URL+"/api/curriculum/tailor", map[string]string{})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 when jd is empty, got %d", resp.StatusCode)
	}

	// 6. Tailor with jd_text -> 200 OK
	resp, out = postJSON(t, srv.URL+"/api/curriculum/tailor", map[string]string{
		"jd_text": "Vaga Go Developer com experiência em microsserviços",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %v)", resp.StatusCode, out)
	}
	if out["tailored_md"] != "# João Silva (Adaptado)\n## Experiência\n- Especialista em Go e Cloud" {
		t.Errorf("unexpected tailored_md: %v", out["tailored_md"])
	}
	if out["prompt_version"] != tailor.PromptVersion {
		t.Errorf("unexpected prompt_version: %v", out["prompt_version"])
	}
}
