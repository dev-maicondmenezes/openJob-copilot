package discovery

import (
	"testing"
)

func TestMatchKeyword(t *testing.T) {
	tests := []struct {
		text    string
		keyword string
		want    bool
	}{
		{"Looking for a Go developer", "go", true},
		{"Senior Golang Engineer", "go", false}, // "go" != "golang"
		{"Expert in Google Cloud", "go", false}, // "go" != "google"
		{"Backend (Go) with microservices", "go", true},
		{"Experience with C# and .NET", "c#", true},
		{"Experience with C++ systems", "c++", true},
		{"JavaScript and TypeScript", "type", false},
	}

	for _, tt := range tests {
		got := MatchKeyword(tt.text, tt.keyword)
		if got != tt.want {
			t.Errorf("MatchKeyword(%q, %q) = %v, want %v", tt.text, tt.keyword, got, tt.want)
		}
	}
}

func TestFilterKeywords(t *testing.T) {
	title := "Senior Go Engineer"
	desc := "We are looking for a remote Go developer with Docker and Kubernetes knowledge."

	// Empty keywords passes all
	ok, matches := FilterKeywords(title, desc, nil, "any")
	if !ok || len(matches) != 0 {
		t.Errorf("empty keywords: got ok=%v, matches=%v", ok, matches)
	}

	// Match "any"
	ok, matches = FilterKeywords(title, desc, []string{"go", "rust"}, "any")
	if !ok || len(matches) != 1 || matches[0] != "go" {
		t.Errorf("match any: got ok=%v, matches=%v", ok, matches)
	}

	// Match "all" - passes
	ok, matches = FilterKeywords(title, desc, []string{"go", "docker"}, "all")
	if !ok || len(matches) != 2 {
		t.Errorf("match all (should pass): got ok=%v, matches=%v", ok, matches)
	}

	// Match "all" - fails when one is missing
	ok, matches = FilterKeywords(title, desc, []string{"go", "python"}, "all")
	if ok || len(matches) != 1 {
		t.Errorf("match all (should fail): got ok=%v, matches=%v", ok, matches)
	}
}
