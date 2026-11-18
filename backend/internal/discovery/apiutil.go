package discovery

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// apiGetJSON performs one rate-limited GET and decodes the JSON body. Like the
// HTML fetchers it maps 429/403 to BlockedError for the orchestrator.
func apiGetJSON(ctx context.Context, client *http.Client, rawURL string, wait RequestWaiter, out any) error {
	body, _, err := apiFetch(ctx, client, http.MethodGet, rawURL, wait, nil, "")
	if err != nil {
		return err
	}
	if err := json.Unmarshal(body, out); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	return nil
}

// apiPostJSON performs one rate-limited POST with a JSON payload.
func apiPostJSON(ctx context.Context, client *http.Client, rawURL string, wait RequestWaiter, payload any, out any) error {
	var bodyReader *strings.Reader
	contentType := ""
	if payload != nil {
		raw, err := json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("encode payload: %w", err)
		}
		bodyReader = strings.NewReader(string(raw))
		contentType = "application/json"
	} else {
		bodyReader = strings.NewReader("")
	}
	body, _, err := apiFetch(ctx, client, http.MethodPost, rawURL, wait, bodyReader, contentType)
	if err != nil {
		return err
	}
	if out != nil {
		if err := json.Unmarshal(body, out); err != nil {
			return fmt.Errorf("decode response: %w", err)
		}
	}
	return nil
}

func apiFetch(ctx context.Context, client *http.Client, method, rawURL string, wait RequestWaiter, body io.Reader, contentType string) ([]byte, string, error) {
	if wait != nil {
		if err := wait.Wait(ctx); err != nil {
			return nil, rawURL, err
		}
	}
	if client == nil {
		client = http.DefaultClient
	}
	req, err := http.NewRequestWithContext(ctx, method, rawURL, body)
	if err != nil {
		return nil, rawURL, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("User-Agent", scrapeUserAgent)
	req.Header.Set("Accept", "application/json")
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, rawURL, fmt.Errorf("http request failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode == http.StatusForbidden {
		return nil, rawURL, &BlockedError{StatusCode: resp.StatusCode}
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, rawURL, fmt.Errorf("unexpected status HTTP %d", resp.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxScrapeBody+1))
	if err != nil {
		return nil, rawURL, fmt.Errorf("read body: %w", err)
	}
	if len(data) > maxScrapeBody {
		return nil, rawURL, fmt.Errorf("response body too large")
	}
	return data, resp.Request.URL.String(), nil
}

// jsonItems extracts the first plausible list of objects from an API payload.
// It accepts direct arrays and common envelopes (data/jobs/offers/results/
// opportunities/vagas) recursively, because LATAM APIs vary widely.
func jsonItems(root any) []map[string]any {
	switch v := root.(type) {
	case []any:
		return mapsOnly(v)
	case map[string]any:
		for _, key := range []string{"data", "jobs", "offers", "results", "opportunities", "vagas", "items", "content"} {
			if child, ok := v[key]; ok {
				if items := jsonItems(child); len(items) > 0 {
					return items
				}
			}
		}
		// Some APIs wrap the array one level deeper (data.items, etc.).
		for _, child := range v {
			if items := jsonItems(child); len(items) > 0 {
				return items
			}
		}
	}
	return nil
}

func mapsOnly(in []any) []map[string]any {
	out := make([]map[string]any, 0, len(in))
	for _, item := range in {
		if m, ok := item.(map[string]any); ok {
			out = append(out, m)
		}
	}
	return out
}

// apiJobFromMap normalizes the many field names used by LATAM job APIs.
// siteBase is used to synthesize an absolute URL when the payload only has a
// slug/id.
func apiJobFromMap(m map[string]any, siteBase string) DiscoveredJob {
	// Flatten the common JSON:API "attributes" envelope.
	if attrs, ok := m["attributes"].(map[string]any); ok {
		merged := make(map[string]any, len(m)+len(attrs))
		for k, v := range m {
			merged[k] = v
		}
		for k, v := range attrs {
			merged[k] = v
		}
		m = merged
	}

	title := jStr(m, "title", "name", "jobTitle", "position", "cargo", "titulo")
	company := jStr(m, "company_name", "companyName", "company", "careerPageName", "empresa", "organization", "hiringOrganization")
	if company == "" {
		company = nestedStr(m, []string{"company", "data", "attributes", "name"})
	}
	if company == "" {
		company = nestedStr(m, []string{"hiringOrganization", "name"})
	}
	rawURL := jStr(m, "url", "jobUrl", "jobURL", "link", "applicationLink", "apply_url", "applyUrl", "permalink", "external_url")
	if rawURL == "" {
		slug := jStr(m, "slug", "path", "permalink_path")
		if slug != "" && siteBase != "" {
			rawURL = strings.TrimRight(siteBase, "/") + "/" + strings.TrimLeft(slug, "/")
		}
	}
	description := jStr(m, "description", "jobDescription", "descricao", "descripcion", "excerpt", "summary")
	location := jStr(m, "location", "city", "local", "localizacao", "ubicacion")
	if state := jStr(m, "state", "province"); state != "" && !strings.Contains(location, state) {
		if location == "" {
			location = state
		} else {
			location += ", " + state
		}
	}
	if country := jStr(m, "country", "pais"); country != "" && !strings.Contains(location, country) {
		if location == "" {
			location = country
		} else {
			location += ", " + country
		}
	}
	modality := jStr(m, "modality", "workplaceType", "workplace_type", "workModel", "work_model", "regime")
	if modality == "" {
		if b, ok := m["remote"].(bool); ok {
			if b {
				modality = "remoto"
			} else {
				modality = "presencial"
			}
		}
	}
	salary := jStr(m, "salary", "salary_range", "salario", "faixa_salarial")

	extID := jStr(m, "id", "guid", "code", "slug")
	if extID == "" {
		extID = extractExternalID(rawURL)
	}
	if id, ok := m["id"]; ok {
		extID = fmt.Sprint(id)
	}

	return DiscoveredJob{
		Title:       strings.TrimSpace(title),
		Company:     strings.TrimSpace(company),
		URL:         strings.TrimSpace(rawURL),
		Description: composeDescription(description, location, modality, salary),
		ExternalID:  extID,
		Location:    location,
		Modality:    modality,
		Salary:      salary,
	}
}

func jStr(m map[string]any, keys ...string) string {
	for _, key := range keys {
		if v, ok := m[key]; ok {
			switch t := v.(type) {
			case string:
				if s := strings.TrimSpace(t); s != "" {
					return s
				}
			case float64:
				return strconv.FormatFloat(t, 'f', -1, 64)
			case json.Number:
				return t.String()
			case map[string]any:
				if s := jStr(t, "name", "title", "label"); s != "" {
					return s
				}
			}
		}
	}
	return ""
}

func nestedStr(m map[string]any, path []string) string {
	var cur any = m
	for _, key := range path {
		obj, ok := cur.(map[string]any)
		if !ok {
			return ""
		}
		cur = obj[key]
	}
	if s, ok := cur.(string); ok {
		return strings.TrimSpace(s)
	}
	return ""
}

// composeDescription keeps listing metadata visible in the persisted job
// description without schema changes.
func composeDescription(description, location, modality, salary string) string {
	desc := CleanHTML(description)
	var meta []string
	if location != "" {
		meta = append(meta, "Local: "+location)
	}
	if modality != "" {
		meta = append(meta, "Modalidade: "+modality)
	}
	if salary != "" {
		meta = append(meta, "Salário: "+salary)
	}
	if len(meta) == 0 {
		return desc
	}
	prefix := strings.Join(meta, " · ")
	if desc == "" {
		return prefix
	}
	return prefix + "\n\n" + desc
}

// buildQuery joins a base URL with query parameters in a stable order.
func buildQuery(base string, params map[string]string) string {
	u, err := url.Parse(base)
	if err != nil {
		return base
	}
	q := u.Query()
	for k, v := range params {
		q.Set(k, v)
	}
	u.RawQuery = q.Encode()
	return u.String()
}
