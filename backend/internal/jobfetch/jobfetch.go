// Package jobfetch fetches a job page from a pasted URL and extracts a
// draft (title, company, description) so the manual form can be prefilled
// (Checkpoint 5 UX: the URL is the primary source, paste-first).
//
// Extraction strategy, in order of trust:
//  1. JSON-LD JobPosting (schema.org) — the structured form used by most
//     ATS pages (Greenhouse, Lever, Workday embed it);
//  2. meta tags (og:title / og:site_name / og:description…);
//  3. <title> tag + the largest text block in the page (readability-lite).
package jobfetch

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	// maxBodyBytes bounds page downloads — job pages are text-heavy HTML.
	maxBodyBytes = 3 << 20 // 3 MiB
	// fetchTimeout is the whole-fetch budget (connect + redirects + read).
	fetchTimeout = 20 * time.Second
	// maxRedirects caps redirect chains.
	maxRedirects = 5
	// userAgent identifies the app politely.
	userAgent = "openjob/1.0 (personal job tracker)"
)

// Draft is what the extractor manages to pull from a job page.
type Draft struct {
	URL         string
	Title       string
	Company     string
	Description string
}

var (
	// ErrInvalidURL reports a malformed or non-http(s) URL.
	ErrInvalidURL = errors.New("jobfetch: URL inválida")
	// ErrNotHTML reports a response that is not an HTML page.
	ErrNotHTML = errors.New("jobfetch: resposta não é HTML")
)

// Fetch downloads and parses the page at rawURL into a Draft. The returned
// draft may have an empty Description (page without extractable body) — the
// caller decides what to do; Title/Company are best-effort.
func Fetch(ctx context.Context, rawURL string) (Draft, error) {
	u, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return Draft{}, ErrInvalidURL
	}

	client := &http.Client{
		Timeout: fetchTimeout,
		CheckRedirect: func(_ *http.Request, via []*http.Request) error {
			if len(via) > maxRedirects {
				return fmt.Errorf("jobfetch: mais de %d redirecionamentos", maxRedirects)
			}
			return nil
		},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return Draft{}, fmt.Errorf("%w: %v", ErrInvalidURL, err)
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml")

	resp, err := client.Do(req)
	if err != nil {
		return Draft{}, fmt.Errorf("jobfetch: falha ao acessar %s: %w", u.Host, err)
	}
	defer resp.Body.Close()

	ct := resp.Header.Get("Content-Type")
	if resp.StatusCode != http.StatusOK {
		return Draft{}, fmt.Errorf("jobfetch: %s respondeu HTTP %d", u.Host, resp.StatusCode)
	}
	if ct != "" && !isHTMLContentType(ct) {
		return Draft{}, ErrNotHTML
	}

	doc, err := parseHTML(io.LimitReader(resp.Body, maxBodyBytes))
	if err != nil {
		return Draft{}, fmt.Errorf("jobfetch: falha ao interpretar HTML: %w", err)
	}

	draft := Draft{URL: u.String()}
	d := &extractor{}
	d.extract(doc)
	draft.Title = d.title
	draft.Company = d.company
	draft.Description = d.description
	if draft.Title == "" && draft.Description == "" {
		return draft, fmt.Errorf("jobfetch: não foi possível extrair dados da página %s", u.Host)
	}
	return draft, nil
}

func isHTMLContentType(ct string) bool {
	media := strings.ToLower(strings.TrimSpace(strings.Split(ct, ";")[0]))
	return media == "text/html" || media == "application/xhtml+xml"
}
