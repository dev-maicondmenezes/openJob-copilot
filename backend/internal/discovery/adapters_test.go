package discovery

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHimalayasAdapter(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Himalayas-Key") != "my-key" {
			t.Errorf("expected X-Himalayas-Key header, got %q", r.Header.Get("X-Himalayas-Key"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"jobs": [
				{
					"title": "Senior Go Engineer",
					"companyName": "Acme Inc",
					"applicationLink": "https://himalayas.app/jobs/1",
					"guid": "guid-1",
					"description": "<p>Build cloud services in Go.</p>"
				}
			]
		}`))
	}))
	defer server.Close()

	adapter := NewHimalayasAdapter()
	adapter.BaseURL = server.URL
	adapter.HTTPClient = server.Client()

	cfg := SourceConfig{
		APIKey: "my-key",
		Capabilities: Capabilities{
			MaxResultsPerRun: 10,
		},
	}

	jobs, err := adapter.Fetch(context.Background(), cfg, []string{"go"})
	if err != nil {
		t.Fatalf("fetch himalayas: %v", err)
	}
	if len(jobs) != 1 {
		t.Fatalf("expected 1 job, got %d", len(jobs))
	}
	if jobs[0].Title != "Senior Go Engineer" || jobs[0].Company != "Acme Inc" || jobs[0].URL != "https://himalayas.app/jobs/1" {
		t.Errorf("unexpected job: %+v", jobs[0])
	}
	if jobs[0].Source != "himalayas" || jobs[0].ExternalID != "guid-1" {
		t.Errorf("unexpected source/externalID: %+v", jobs[0])
	}
}

func TestRemotiveAdapter(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("search") != "go backend" {
			t.Errorf("expected search query 'go backend', got %q", r.URL.Query().Get("search"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"jobs": [
				{
					"id": 9988,
					"url": "https://remotive.com/jobs/9988",
					"title": "Backend Go Developer",
					"company_name": "Remotive Co",
					"description": "<div>Great Go position</div>"
				}
			]
		}`))
	}))
	defer server.Close()

	adapter := NewRemotiveAdapter()
	adapter.BaseURL = server.URL
	adapter.HTTPClient = server.Client()

	cfg := SourceConfig{
		Capabilities: Capabilities{
			MaxResultsPerRun: 25,
		},
	}

	jobs, err := adapter.Fetch(context.Background(), cfg, []string{"go", "backend"})
	if err != nil {
		t.Fatalf("fetch remotive: %v", err)
	}
	if len(jobs) != 1 {
		t.Fatalf("expected 1 job, got %d", len(jobs))
	}
	if jobs[0].Title != "Backend Go Developer" || jobs[0].ExternalID != "9988" {
		t.Errorf("unexpected job: %+v", jobs[0])
	}
}

func TestJobicyAdapter(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"jobs": [
				{
					"id": 1234,
					"url": "https://jobicy.com/jobs/1234",
					"jobTitle": "DevOps Specialist",
					"companyName": "CloudCorp",
					"jobDescription": "<p>Kubernetes and Terraform</p>"
				}
			]
		}`))
	}))
	defer server.Close()

	adapter := NewJobicyAdapter()
	adapter.BaseURL = server.URL
	adapter.HTTPClient = server.Client()

	jobs, err := adapter.Fetch(context.Background(), SourceConfig{}, nil)
	if err != nil {
		t.Fatalf("fetch jobicy: %v", err)
	}
	if len(jobs) != 1 {
		t.Fatalf("expected 1 job, got %d", len(jobs))
	}
	if jobs[0].Title != "DevOps Specialist" || jobs[0].Company != "CloudCorp" || jobs[0].ExternalID != "1234" {
		t.Errorf("unexpected job: %+v", jobs[0])
	}
}

func TestRemoteOKAdapter(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// RemoteOK returns array, first item is disclaimer without position
		_, _ = w.Write([]byte(`[
			{"legal": "Notice"},
			{
				"id": "rok-1",
				"url": "https://remoteok.com/j/1",
				"position": "Staff Engineer",
				"company": "Nomad Inc",
				"description": "<div>Join our distributed team</div>"
			}
		]`))
	}))
	defer server.Close()

	adapter := NewRemoteOKAdapter()
	adapter.BaseURL = server.URL
	adapter.HTTPClient = server.Client()

	jobs, err := adapter.Fetch(context.Background(), SourceConfig{}, nil)
	if err != nil {
		t.Fatalf("fetch remoteok: %v", err)
	}
	if len(jobs) != 1 {
		t.Fatalf("expected 1 job, got %d", len(jobs))
	}
	if jobs[0].Title != "Staff Engineer" || jobs[0].Company != "Nomad Inc" || jobs[0].ExternalID != "rok-1" {
		t.Errorf("unexpected job: %+v", jobs[0])
	}
}
