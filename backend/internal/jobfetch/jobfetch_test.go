package jobfetch

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const fixtureJSONLD = `<!DOCTYPE html>
<html><head>
<title>Senior Go Developer - Greenhouse</title>
<script type="application/ld+json">
{"@context":"https://schema.org","@graph":[
  {"@type":"WebSite","name":"Greenhouse"},
  {"@type":"JobPosting","title":"Senior Go Developer",
   "description":"<p>Remote-first role.</p><ul><li>5+ years of Go</li><li>Kubernetes</li></ul>",
   "hiringOrganization":{"name":"Acme Corp"}}]}
</script>
</head><body><div>Irrelevant marketing copy</div></body></html>`

const fixtureMeta = `<!DOCTYPE html>
<html><head>
<meta property="og:title" content="Backend Engineer — Beta Inc">
<meta property="og:site_name" content="Beta Careers">
<title>Ignored page title</title>
</head>
<body><header>Nav Home Jobs About</header>
<article><p>Requisitos:</p><ul><li>Go</li><li>SQL</li></ul>
<p>Vaga remota para engenheiros com forte experiência em bancos de dados relacionais e desenho de APIs REST de alto tráfego.</p></article>
<footer>Copyright</footer></body></html>`

const fixtureLargestDiv = `<!DOCTYPE html>
<html><head><title>Pleno DevOps — Gama</title></head>
<body>
<div>menu início vagas contato</div>
<div>
<h1>Pleno DevOps</h1>
<p>A Gama busca pessoa DevOps plena para atuar remotamente.</p>
<p>Requisitos essenciais: Linux, Docker, CI/CD, observabilidade e automação de infraestrutura com Terraform em ambientes cloud de grande escala.</p>
<p>Diferenciais: Kubernetes, service mesh e cultura de engenharia madura com deploy contínuo diário e revisões de arquitetura frequentes.</p>
</div>
<div>footer</div>
</body></html>`

func TestFetchJSONLDPosting(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(fixtureJSONLD))
	}))
	defer srv.Close()

	draft, err := Fetch(context.Background(), srv.URL+"/jobs/123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if draft.Title != "Senior Go Developer" {
		t.Errorf("title = %q, want Senior Go Developer", draft.Title)
	}
	if draft.Company != "Acme Corp" {
		t.Errorf("company = %q, want Acme Corp", draft.Company)
	}
	want := "Remote-first role.\n\n- 5+ years of Go\n- Kubernetes"
	if draft.Description != want {
		t.Errorf("description =\n%q\nwant\n%q", draft.Description, want)
	}
}

func TestFetchMetaTags(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(fixtureMeta))
	}))
	defer srv.Close()

	draft, err := Fetch(context.Background(), srv.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if draft.Title != "Backend Engineer — Beta Inc" {
		t.Errorf("title = %q", draft.Title)
	}
	if draft.Company != "Beta Careers" {
		t.Errorf("company = %q", draft.Company)
	}
	if !strings.Contains(draft.Description, "Requisitos:") ||
		strings.Contains(draft.Description, "Nav Home") ||
		strings.Contains(draft.Description, "Copyright") {
		t.Errorf("description should be the article block only, got:\n%s", draft.Description)
	}
}

func TestFetchLargestBlockFallback(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(fixtureLargestDiv))
	}))
	defer srv.Close()

	draft, err := Fetch(context.Background(), srv.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// <title> beats h1 here? No: h1 has priority over <title>.
	if draft.Title != "Pleno DevOps" {
		t.Errorf("title = %q, want Pleno DevOps", draft.Title)
	}
	if draft.Company != "Gama" {
		t.Errorf("company = %q, want Gama (from title suffix)", draft.Company)
	}
	if !strings.Contains(draft.Description, "Requisitos essenciais") ||
		strings.Contains(draft.Description, "menu início") ||
		strings.Contains(draft.Description, "footer") {
		t.Errorf("description picked wrong block:\n%s", draft.Description)
	}
}

func TestFetchErrors(t *testing.T) {
	// Invalid URLs
	for _, raw := range []string{"", "notaurl", "ftp://x.com/a"} {
		if _, err := Fetch(context.Background(), raw); err != ErrInvalidURL {
			t.Errorf("Fetch(%q) error = %v, want ErrInvalidURL", raw, err)
		}
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/pdf":
			w.Header().Set("Content-Type", "application/pdf")
			_, _ = w.Write([]byte("%PDF-1.4"))
		case "/404":
			http.NotFound(w, r)
		case "/broke":
			panic("boom")
		default:
			_, _ = w.Write([]byte(fixtureJSONLD))
		}
	}))
	defer srv.Close()

	if _, err := Fetch(context.Background(), srv.URL+"/pdf"); err != ErrNotHTML {
		t.Errorf("PDF fetch error = %v, want ErrNotHTML", err)
	}
	if _, err := Fetch(context.Background(), srv.URL+"/404"); err == nil || strings.Contains(err.Error(), "HTTP 404") == false {
		t.Errorf("404 error = %v, want HTTP 404 detail", err)
	}
}

func TestFetchFollowsRedirect(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/final" {
			http.Redirect(w, r, "/final", http.StatusMovedPermanently)
			return
		}
		_, _ = w.Write([]byte(fixtureJSONLD))
	}))
	defer srv.Close()

	draft, err := Fetch(context.Background(), srv.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if draft.Title != "Senior Go Developer" {
		t.Errorf("title = %q", draft.Title)
	}
}

// JS-rendered ATS pages (e.g. new Greenhouse) have no meaningful text in the
// raw HTML — script/style contents must be ignored and the description ends
// up empty so the UI asks for manual paste.
const fixtureJSRendered = `<!DOCTYPE html>
<html><head><title>Careers at Faire | Join us!</title>
<script>var huge = "bootstrap json ".repeat(500);</script>
<style>body{margin:0}</style></head>
<body><div id="__next"></div><script>mount();</script></body></html>`

func TestFetchJSRenderedPageGivesNoDescription(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(fixtureJSRendered))
	}))
	defer srv.Close()

	draft, err := Fetch(context.Background(), srv.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if draft.Description != "" {
		t.Errorf("description should be empty for JS-rendered page, got %d chars", len(draft.Description))
	}
	if strings.Contains(draft.Company, "Join us") {
		t.Errorf("page chrome must not become the company: %q", draft.Company)
	}
}
