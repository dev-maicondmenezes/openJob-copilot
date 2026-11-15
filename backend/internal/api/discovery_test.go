package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"
)

func TestDiscoverySettingsEndpoints(t *testing.T) {
	srv := newTestServer(t)

	// 1. Get default discovery settings
	resp, err := srv.Client().Get(srv.URL + "/api/settings/discovery")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("get discovery settings status = %d, want 200", resp.StatusCode)
	}
	var settings map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&settings); err != nil {
		t.Fatal(err)
	}
	if settings["discovery_interval"] != "off" {
		t.Errorf("default discovery_interval = %v, want 'off'", settings["discovery_interval"])
	}

	// 2. Reject > 5 keywords with 400
	badBody, _ := json.Marshal(map[string]any{
		"keywords": []string{"one", "two", "three", "four", "five", "six"},
	})
	req, _ := http.NewRequest(http.MethodPut, srv.URL+"/api/settings/discovery", bytes.NewReader(badBody))
	req.Header.Set("Content-Type", "application/json")
	resp, err = srv.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want 400 for >5 keywords", resp.StatusCode)
	}

	// 3. Save valid <= 5 keywords
	goodBody, _ := json.Marshal(map[string]any{
		"keywords":           []string{"go", "backend", "remote"},
		"discovery_interval": "6h",
		"resume_filter": map[string]any{
			"enabled":     true,
			"min_matches": 3,
			"match_mode":  "any",
		},
	})
	req, _ = http.NewRequest(http.MethodPut, srv.URL+"/api/settings/discovery", bytes.NewReader(goodBody))
	req.Header.Set("Content-Type", "application/json")
	resp, err = srv.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("put settings status = %d, want 200", resp.StatusCode)
	}

	// 4. Check discovery status
	resp, err = srv.Client().Get(srv.URL + "/api/jobs/discover/status")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("get discover status = %d, want 200", resp.StatusCode)
	}
	var status map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&status)
	if status["running"] != false {
		t.Errorf("running = %v, want false", status["running"])
	}
}
