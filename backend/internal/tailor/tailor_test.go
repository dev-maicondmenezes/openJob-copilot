package tailor

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

func TestTailorSuccess(t *testing.T) {
	mock := &mockLLMClient{
		resp: "```markdown\n# Fulano\n## Experiência\n- Desenvolveu sistemas Go\n```",
	}
	cfg := llm.Config{APIKey: "test", Model: "gpt-4o-mini"}
	out, err := Tailor(context.Background(), mock, cfg, "# Fulano\nGo Dev", "Vaga Go")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "# Fulano\n## Experiência\n- Desenvolveu sistemas Go"
	if out != expected {
		t.Fatalf("got %q, want %q", out, expected)
	}

	if len(mock.lastReq.Messages) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(mock.lastReq.Messages))
	}
	if mock.lastReq.Messages[0].Role != "system" {
		t.Errorf("system message expected first")
	}
	if !strings.Contains(mock.lastReq.Messages[1].Content, "Vaga Go") {
		t.Errorf("user message should contain JD text")
	}
}

func TestTailorValidation(t *testing.T) {
	mock := &mockLLMClient{}
	cfg := llm.Config{APIKey: "test"}

	if _, err := Tailor(context.Background(), mock, cfg, "", "Vaga"); err != ErrEmptyCurriculum {
		t.Errorf("expected ErrEmptyCurriculum, got %v", err)
	}

	if _, err := Tailor(context.Background(), mock, cfg, "CV", "   "); err != ErrEmptyJobDescription {
		t.Errorf("expected ErrEmptyJobDescription, got %v", err)
	}
}

func TestCleanMarkdownOutput(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{
			input:    "```markdown\n# Title\n- Item\n```",
			expected: "# Title\n- Item",
		},
		{
			input:    "```md\n# Title\n```",
			expected: "# Title",
		},
		{
			input:    "```\n# Title\n```",
			expected: "# Title",
		},
		{
			input:    "# Normal Markdown without fences",
			expected: "# Normal Markdown without fences",
		},
	}

	for _, tc := range tests {
		got := CleanMarkdownOutput(tc.input)
		if got != tc.expected {
			t.Errorf("CleanMarkdownOutput(%q) = %q, want %q", tc.input, got, tc.expected)
		}
	}
}

func TestTruncation(t *testing.T) {
	huge := strings.Repeat("A", MaxCurriculumChars+500)
	prompt := BuildUserPrompt(huge, "JD")
	if !strings.Contains(prompt, "[... truncado pelo limite de tamanho ...]") {
		t.Errorf("expected prompt to contain truncation notice")
	}
}
