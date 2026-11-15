package score

import (
	"context"
	"strings"
	"testing"

	"github.com/maicondmenezes/openjob/internal/llm"
)

type mockLLMClient struct {
	lastReq llm.CompletionRequest
	resp    string
	err     error
}

func (m *mockLLMClient) Ping(ctx context.Context, cfg llm.Config) error {
	return nil
}

func (m *mockLLMClient) Complete(ctx context.Context, cfg llm.Config, req llm.CompletionRequest) (string, error) {
	m.lastReq = req
	return m.resp, m.err
}

// ---- CleanJD (parsing de JD — teste crítico, spec §10.5) ------------------

func TestCleanJDStripsHTML(t *testing.T) {
	raw := `<p>Senior <b>Go</b> Developer<br>Remote&nbsp;only</p>
<div>Requirements:<ul><li>5+ years Go</li><li>Kubernetes</li></ul>
<script>trackPageView();</script><style>.p{color:red}</style></div>`

	got := CleanJD(raw)
	want := "Senior Go Developer\nRemote only\n\nRequirements:\n- 5+ years Go\n- Kubernetes"

	if got != want {
		t.Errorf("CleanJD =\n%q\nwant\n%q", got, want)
	}
}

func TestCleanJDUnescapesEntities(t *testing.T) {
	got := CleanJD("Salary: R$&nbsp;10.000 &amp; benefits &lt;awesome&gt;")
	want := "Salary: R$ 10.000 & benefits <awesome>"
	if got != want {
		t.Errorf("CleanJD = %q, want %q", got, want)
	}
}

func TestCleanJDCollapsesWhitespace(t *testing.T) {
	got := CleanJD("Line one\n\n\n\n\nLine two\n   \n\nLine  three    four")
	want := "Line one\n\nLine two\n\nLine three four"
	if got != want {
		t.Errorf("CleanJD = %q, want %q", got, want)
	}
}

func TestCleanJDPlainTextUnchanged(t *testing.T) {
	in := "Vaga Go: 5 anos de experiência.\nRemoto."
	if got := CleanJD(in); got != in {
		t.Errorf("CleanJD = %q, want %q", got, in)
	}
}

// ---- ParseScore (validação/clamp — teste crítico) --------------------------

func TestParseScoreStrictJSON(t *testing.T) {
	res, err := ParseScore(`{"score": 82, "reason": "Forte aderência em Go."}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Score != 82 || res.Reason != "Forte aderência em Go." {
		t.Errorf("got %+v, want score=82 reason=Forte aderência em Go.", res)
	}
}

func TestParseScoreFencedJSON(t *testing.T) {
	res, err := ParseScore("```json\n{\"score\": 64, \"reason\": \"Match parcial.\"}\n```")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Score != 64 || res.Reason != "Match parcial." {
		t.Errorf("got %+v", res)
	}
}

func TestParseScoreJSONWithProse(t *testing.T) {
	res, err := ParseScore(`Avaliação: {"score": 55, "reason": "Lacunas em k8s."} Fim.`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Score != 55 || res.Reason != "Lacunas em k8s." {
		t.Errorf("got %+v", res)
	}
}

func TestParseScoreRegexFallback(t *testing.T) {
	// Model ignored the JSON contract entirely.
	res, err := ParseScore("O candidato tem fit razoável. score: 48\nMotivo: faltou experiência.")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Score != 48 {
		t.Errorf("score = %d, want 48", res.Score)
	}
}

func TestParseScoreClamp(t *testing.T) {
	tests := []struct {
		input string
		want  int
	}{
		{`{"score": 150, "reason": "x"}`, 100},
		{`{"score": -7, "reason": "x"}`, 0},
		{`{"score": 82.9, "reason": "x"}`, 82},
		{`{"score": "92", "reason": "x"}`, 92},
		{`{"score": 73,5, "reason": "x"}`, 73},
	}
	for _, tc := range tests {
		res, err := ParseScore(tc.input)
		if err != nil {
			t.Errorf("ParseScore(%q) error: %v", tc.input, err)
			continue
		}
		if res.Score != tc.want {
			t.Errorf("ParseScore(%q) score = %d, want %d", tc.input, res.Score, tc.want)
		}
	}
}

func TestParseScoreUnparseable(t *testing.T) {
	for _, input := range []string{"", "não sei avaliar", "score: none", `{"reason": "sem score"}`} {
		if _, err := ParseScore(input); err == nil {
			t.Errorf("ParseScore(%q) = nil error, want ErrParse", input)
		}
	}
}

// ---- Score end-to-end with mock LLM ---------------------------------------

func TestScoreSuccess(t *testing.T) {
	mock := &mockLLMClient{resp: `{"score": 88, "reason": "Excelente match."}`}
	cfg := llm.Config{APIKey: "test", Model: "gpt-4o-mini"}

	res, err := Score(context.Background(), mock, cfg, "# Fulano\nDev Go", "<p>Vaga Go<br>Remoto</p>")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Score != 88 || res.Reason != "Excelente match." {
		t.Errorf("got %+v", res)
	}

	// The JD must reach the prompt cleaned of HTML.
	if !strings.Contains(mock.lastReq.Messages[1].Content, "Vaga Go\nRemoto") {
		t.Errorf("JD was not cleaned before the prompt: %q", mock.lastReq.Messages[1].Content)
	}
	if len(mock.lastReq.Messages) != 2 || mock.lastReq.Messages[0].Role != "system" {
		t.Errorf("expected system+user messages, got %+v", mock.lastReq.Messages)
	}
}

func TestScoreValidation(t *testing.T) {
	mock := &mockLLMClient{}
	cfg := llm.Config{APIKey: "test"}

	if _, err := Score(context.Background(), mock, cfg, "", "Vaga"); err != ErrEmptyCurriculum {
		t.Errorf("expected ErrEmptyCurriculum, got %v", err)
	}
	if _, err := Score(context.Background(), mock, cfg, "CV", "   "); err != ErrEmptyJobDescription {
		t.Errorf("expected ErrEmptyJobDescription, got %v", err)
	}
}

func TestScorePropagatesProviderError(t *testing.T) {
	mock := &mockLLMClient{err: context.DeadlineExceeded}
	_, err := Score(context.Background(), mock, llm.Config{APIKey: "test"}, "CV", "JD")
	if err == nil {
		t.Fatal("expected provider error to propagate")
	}
	if AsErrParse(err) {
		t.Errorf("provider error must not be classified as parse error: %v", err)
	}
}
