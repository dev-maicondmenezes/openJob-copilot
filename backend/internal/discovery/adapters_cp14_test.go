package discovery

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestCP14APIAdapters(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		adapter    Adapter
		wantTitle  string
		wantComp   string
		wantSource string
	}{
		{
			name: "getonboard",
			body: `{"data":[{"id":"gb-1","attributes":{"title":"Senior Go Engineer","company_name":"Acme LATAM","url":"https://www.getonbrd.com/jobs/gb-1","description":"<p>Go and cloud</p>"}}]}`,
			adapter: func() Adapter {
				a := NewGetOnBoardAdapter()
				return a
			}(),
			wantTitle:  "Senior Go Engineer",
			wantComp:   "Acme LATAM",
			wantSource: "getonboard",
		},
		{
			name:       "trampos",
			body:       `{"opportunities":[{"id":2,"title":"Backend Developer","company":"Trampos Co","url":"https://trampos.co/vagas/2","description":"Go backend"}]}`,
			adapter:    NewTramposAdapter(),
			wantTitle:  "Backend Developer",
			wantComp:   "Trampos Co",
			wantSource: "trampos",
		},
		{
			name:       "gupy",
			body:       `{"data":[{"id":3,"name":"Platform Engineer","careerPageName":"Gupy Corp","jobUrl":"https://portal.gupy.io/job/3","description":"Kubernetes"}]}`,
			adapter:    NewGupyAdapter(),
			wantTitle:  "Platform Engineer",
			wantComp:   "Gupy Corp",
			wantSource: "gupy",
		},
		{
			name:       "getmanfred",
			body:       `{"offers":[{"id":4,"title":"Ingeniero Backend","company":"Manfred SL","url":"https://www.getmanfred.com/offers/4","description":"Go"}]}`,
			adapter:    NewManfredAdapter(),
			wantTitle:  "Ingeniero Backend",
			wantComp:   "Manfred SL",
			wantSource: "getmanfred",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotPath string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotPath = r.URL.Path
				if ua := r.Header.Get("User-Agent"); !strings.Contains(ua, "openJob") {
					t.Errorf("User-Agent = %q, want identifiable openJob", ua)
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(tt.body))
			}))
			defer server.Close()

			switch a := tt.adapter.(type) {
			case *GetOnBoardAdapter:
				a.BaseURL = server.URL + "/api/v0"
				a.HTTPClient = server.Client()
			case *TramposAdapter:
				a.BaseURL = server.URL + "/api/v1/opportunities"
				a.HTTPClient = server.Client()
			case *GupyAdapter:
				a.BaseURL = server.URL + "/api/v1/jobs"
				a.HTTPClient = server.Client()
			case *ManfredAdapter:
				a.BaseURL = server.URL + "/api/v2/public/offers"
				a.HTTPClient = server.Client()
			}

			jobs, err := tt.adapter.Fetch(context.Background(), SourceConfig{
				Capabilities: Capabilities{MaxResultsPerRun: 10},
				Usage:        &UsageTracker{},
			}, []string{"go"})
			if err != nil {
				t.Fatalf("fetch: %v", err)
			}
			if len(jobs) != 1 {
				t.Fatalf("jobs = %d, want 1 (path %s)", len(jobs), gotPath)
			}
			if jobs[0].Title != tt.wantTitle || jobs[0].Company != tt.wantComp || jobs[0].Source != tt.wantSource {
				t.Fatalf("job = %+v, want title=%q company=%q source=%q", jobs[0], tt.wantTitle, tt.wantComp, tt.wantSource)
			}
		})
	}
}

const cp14ListingHTML = `<html><body>
<div class="job"><h2 class="job-title"><a href="/jobs/101">Go Developer</a></h2><span class="job-company">Acme</span><span class="job-location">Remoto</span></div>
<div class="job"><h2 class="job-title"><a href="/jobs/102">SRE</a></h2><span class="job-company">Beta</span><span class="job-location">São Paulo</span></div>
</body></html>`

func TestCP14ScrapingAdaptersParseCards(t *testing.T) {
	type scraper struct {
		name    string
		adapter Adapter
		html    string
		path    string
	}
	tests := []scraper{
		{
			name: "programathor", adapter: NewProgramathorAdapter(),
			html: cp14ListingHTML, path: "/jobs",
		},
		{
			name: "computrabajo", adapter: NewComputrabajoAdapter(),
			html: `<div class="box_offer"><a class="js-o-link" href="/offer/1">Vendedor</a><span class="company">ACME</span><span class="location">Madrid</span></div>`,
			path: "/trabajo-de-go",
		},
		{
			name: "catho", adapter: NewCathoAdapter(),
			html: `<div class="job-card"><a class="job-title" href="/vagas/1">Dev Go</a><span class="company">Catho Co</span><span class="location">Remoto</span></div>`,
			path: "/vagas/go/",
		},
		{
			name: "infojobs", adapter: NewInfoJobsAdapter(),
			html: `<div class="js_card"><a class="js_vacancyTitle" href="/vaga/1">Dev Go</a><span class="js_vacancyCompany">Info Co</span><span class="js_vacancyLocation">Brasil</span></div>`,
			path: "/empregos.aspx",
		},
		{
			name: "vagas", adapter: NewVagasAdapter(),
			html: `<div class="vaga"><a class="link-detalhes-vaga" href="/vagas/123">Dev Go</a><span class="emprVaga">Vagas Co</span><span class="local">Remoto</span></div>`,
			path: "/vagas-de-go",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var requested []string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requested = append(requested, r.URL.RequestURI())
				w.Header().Set("Content-Type", "text/html")
				_, _ = w.Write([]byte(tt.html))
			}))
			defer server.Close()

			switch a := tt.adapter.(type) {
			case *ProgramathorAdapter:
				a.BaseURL = server.URL
			case *ComputrabajoAdapter:
				a.BaseURL = server.URL
			case *CathoAdapter:
				a.BaseURL = server.URL
			case *InfoJobsAdapter:
				a.BaseURL = server.URL
			case *VagasAdapter:
				a.BaseURL = server.URL
			}

			jobs, err := tt.adapter.Fetch(context.Background(), SourceConfig{
				Capabilities:   Capabilities{MaxResultsPerRun: 10},
				DailyRemaining: 0,
				Usage:          &UsageTracker{},
			}, []string{"go"})
			if err != nil {
				t.Fatalf("fetch: %v", err)
			}
			if len(jobs) == 0 {
				t.Fatalf("no jobs parsed; requested=%v", requested)
			}
			if !strings.Contains(requested[0], tt.path) {
				t.Fatalf("path = %q, want prefix %q", requested[0], tt.path)
			}
			if jobs[0].Source != tt.adapter.Slug() {
				t.Fatalf("source = %q, want %q", jobs[0].Source, tt.adapter.Slug())
			}
		})
	}
}

func TestProgramathorCachesListingForSixHours(t *testing.T) {
	var requests int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		_, _ = w.Write([]byte(cp14ListingHTML))
	}))
	defer server.Close()

	a := NewProgramathorAdapter()
	a.BaseURL = server.URL

	usage := &UsageTracker{}
	for i := 0; i < 2; i++ {
		if _, err := a.Fetch(context.Background(), SourceConfig{
			Capabilities: Capabilities{MaxResultsPerRun: 2},
			Usage:        usage,
		}, []string{"go"}); err != nil {
			t.Fatalf("fetch %d: %v", i+1, err)
		}
	}
	if requests != 1 {
		t.Fatalf("network requests = %d, want 1 (second run must hit the 6h cache)", requests)
	}
	if usage.Requests != 1 {
		t.Fatalf("tracked requests = %d, want 1 (cache hits are not requests)", usage.Requests)
	}
}

func TestComputrabajoNeverUsesDisallowedFilters(t *testing.T) {
	var requested []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requested = append(requested, r.URL.RequestURI())
		_, _ = w.Write([]byte(cp14ListingHTML))
	}))
	defer server.Close()

	a := NewComputrabajoAdapter()
	a.BaseURL = server.URL
	_, err := a.Fetch(context.Background(), SourceConfig{
		Capabilities: Capabilities{MaxResultsPerRun: 1},
		Usage:        &UsageTracker{},
	}, []string{"go", "backend"})
	if err != nil {
		t.Fatal(err)
	}
	if len(requested) == 0 {
		t.Fatal("no request made")
	}
	for _, uri := range requested {
		for _, forbidden := range []string{"by=", "pubdate=", "sal="} {
			if strings.Contains(uri, forbidden) {
				t.Fatalf("request %q used robots-disallowed parameter %q", uri, forbidden)
			}
		}
	}
}

func TestBumeranUsesSearchV2WithoutDisallowedQuery(t *testing.T) {
	var gotPath, gotQuery, gotBody string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotQuery = r.URL.Path, r.URL.RawQuery
		body, _ := io.ReadAll(r.Body)
		gotBody = string(body)
		_, _ = w.Write([]byte(`{"avisos":[{"id":"av-1","titulo":"Dev Go","empresa":"ACME","url":"https://www.bumeran.com.ar/empleos/av-1","descripcion":"Go"}]}`))
	}))
	defer server.Close()

	a := NewBumeranAdapter()
	a.APIURL = server.URL + "/api/avisos/searchV2"
	a.SiteURL = "https://www.bumeran.com.ar"
	jobs, err := a.Fetch(context.Background(), SourceConfig{
		Capabilities: Capabilities{MaxResultsPerRun: 10},
		Usage:        &UsageTracker{},
	}, []string{"go"})
	if err != nil {
		t.Fatal(err)
	}
	if len(jobs) != 1 || jobs[0].Title != "Dev Go" {
		t.Fatalf("jobs = %+v", jobs)
	}
	if gotPath != "/api/avisos/searchV2" {
		t.Fatalf("path = %q", gotPath)
	}
	if strings.Contains(gotQuery, "recientes=") || strings.Contains(gotQuery, "relevantes=") || strings.Contains(gotBody, "recientes=") || strings.Contains(gotBody, "relevantes=") {
		t.Fatalf("used robots-disallowed mode: query=%q body=%q", gotQuery, gotBody)
	}
}

func TestVagasNeverAccessesPesquisasAndRespectsDetailQuota(t *testing.T) {
	var listingCalls, detailCalls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/vagas/pesquisas") {
			t.Errorf("adapter accessed robots-disallowed path %q", r.URL.Path)
		}
		if strings.HasPrefix(r.URL.Path, "/vagas/") {
			detailCalls++
			_, _ = w.Write([]byte(`<div id="descricao"><p>Detalhe completo da vaga.</p></div>`))
			return
		}
		listingCalls++
		// Two cards to exercise the per-run detail budget.
		_, _ = w.Write([]byte(`<div class="vaga"><a class="link-detalhes-vaga" href="/vagas/1">Dev Go 1</a><span class="emprVaga">A</span></div>
<div class="vaga"><a class="link-detalhes-vaga" href="/vagas/2">Dev Go 2</a><span class="emprVaga">B</span></div>`))
	}))
	defer server.Close()

	a := NewVagasAdapter()
	a.BaseURL = server.URL
	usage := &UsageTracker{}
	jobs, err := a.Fetch(context.Background(), SourceConfig{
		Capabilities:   a.DefaultCapabilities(),
		DailyRemaining: 20,
		Usage:          usage,
	}, []string{"go"})
	if err != nil {
		t.Fatal(err)
	}
	if len(jobs) != 2 || listingCalls < 1 {
		t.Fatalf("jobs=%d listingCalls=%d", len(jobs), listingCalls)
	}
	if detailCalls != 2 || usage.DetailRequests != 2 {
		t.Fatalf("detailCalls=%d usage=%+v, want 2 (max_details_per_run)", detailCalls, usage)
	}

	// Exhausted quota: listing still works, no detail request at all.
	detailCalls = 0
	usage = &UsageTracker{}
	if _, err := a.Fetch(context.Background(), SourceConfig{
		Capabilities:   a.DefaultCapabilities(),
		DailyRemaining: 0,
		Usage:          usage,
	}, []string{"go"}); err != nil {
		t.Fatal(err)
	}
	if detailCalls != 0 || usage.DetailRequests != 0 {
		t.Fatalf("detailCalls=%d usage=%+v, want 0", detailCalls, usage)
	}
}

func TestScrapingAdapterMapsBlockToBlockedError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer server.Close()

	a := NewProgramathorAdapter()
	a.BaseURL = server.URL
	_, err := a.Fetch(context.Background(), SourceConfig{
		Capabilities: Capabilities{MaxResultsPerRun: 1},
		Usage:        &UsageTracker{},
	}, nil)
	var blocked *BlockedError
	if !errors.As(err, &blocked) {
		t.Fatalf("err = %v, want BlockedError", err)
	}
	if blocked.StatusCode != http.StatusTooManyRequests {
		t.Fatalf("status = %d, want 429", blocked.StatusCode)
	}
}

func TestAbsoluteURLResolution(t *testing.T) {
	base := "https://example.com/jobs"
	u, _ := url.Parse(absoluteURL(base, "/job/1"))
	if u.String() != "https://example.com/job/1" {
		t.Fatalf("resolved = %s", u)
	}
}
