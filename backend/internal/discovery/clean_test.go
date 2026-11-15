package discovery

import (
	"strings"
	"testing"
)

func TestCleanHTML(t *testing.T) {
	raw := `<div><h1>Job Title</h1><p>We are looking for a <strong>Golang</strong> engineer.</p><ul><li>Remote</li><li>Full-time</li></ul><script>alert(1);</script></div>`
	clean := CleanHTML(raw)
	if strings.Contains(clean, "alert(1)") {
		t.Errorf("script tag not stripped: %s", clean)
	}
	if !strings.Contains(clean, "Job Title") || !strings.Contains(clean, "Golang engineer") {
		t.Errorf("text content missing in cleaned output: %s", clean)
	}
	if !strings.Contains(clean, "- Remote") {
		t.Errorf("list item formatting missing: %s", clean)
	}
}
