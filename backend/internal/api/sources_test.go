package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"
)

func TestSourcesEndpoints(t *testing.T) {
	srv := newTestServer(t)

	// 1. List default seeded sources
	resp, err := srv.Client().Get(srv.URL + "/api/sources")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("list sources status = %d, want 200", resp.StatusCode)
	}
	var sources []sourceResponse
	if err := json.NewDecoder(resp.Body).Decode(&sources); err != nil {
		t.Fatal(err)
	}
	if len(sources) < 4 {
		t.Fatalf("expected at least 4 default sources, got %d", len(sources))
	}

	// 2. List adapters
	resp, err = srv.Client().Get(srv.URL + "/api/sources/adapters")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("list adapters status = %d, want 200", resp.StatusCode)
	}

	// 3. Create source with nonexistent adapter -> 400
	badBody, _ := json.Marshal(map[string]any{
		"id":      "ghost",
		"name":    "Ghost Source",
		"adapter": "nonexistent_adapter",
	})
	resp, err = srv.Client().Post(srv.URL+"/api/sources", "application/json", bytes.NewReader(badBody))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want 400 for unknown adapter", resp.StatusCode)
	}

	// 4. Update source (toggle enabled)
	patchBody, _ := json.Marshal(map[string]any{
		"enabled":  false,
		"priority": 999,
	})
	req, _ := http.NewRequest(http.MethodPatch, srv.URL+"/api/sources/remotive", bytes.NewReader(patchBody))
	req.Header.Set("Content-Type", "application/json")
	resp, err = srv.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("patch status = %d, want 200", resp.StatusCode)
	}
	var updated sourceResponse
	_ = json.NewDecoder(resp.Body).Decode(&updated)
	if updated.Enabled || updated.Priority != 999 {
		t.Errorf("expected updated fields, got %+v", updated)
	}

	// 5. Delete source
	req, _ = http.NewRequest(http.MethodDelete, srv.URL+"/api/sources/jobicy", nil)
	resp, err = srv.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Errorf("delete status = %d, want 204", resp.StatusCode)
	}
}
