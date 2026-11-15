package tailor

import (
	"context"
	"errors"
	"strings"

	"github.com/maicondmenezes/openjob/internal/llm"
)

var (
	// ErrEmptyCurriculum is returned when the base curriculum is missing or empty.
	ErrEmptyCurriculum = errors.New("tailor: currículo-base vazio")
	// ErrEmptyJobDescription is returned when the job description is missing or empty.
	ErrEmptyJobDescription = errors.New("tailor: descrição da vaga vazia")
)

// CleanMarkdownOutput strips accidental code fences (e.g. ```markdown ... ```)
// that some LLMs wrap around their answers despite system instructions.
func CleanMarkdownOutput(raw string) string {
	s := strings.TrimSpace(raw)
	// Check if wrapped in ```markdown ... ``` or ``` ... ```
	if strings.HasPrefix(s, "```") {
		firstLineEnd := strings.Index(s, "\n")
		if firstLineEnd != -1 {
			prefix := strings.TrimSpace(s[:firstLineEnd])
			if prefix == "```" || strings.HasPrefix(prefix, "```markdown") || strings.HasPrefix(prefix, "```md") {
				lastFence := strings.LastIndex(s, "```")
				if lastFence > firstLineEnd {
					s = strings.TrimSpace(s[firstLineEnd:lastFence])
				}
			}
		}
	}
	return s
}

// Tailor generates an adapted resume tailored for a target job description.
func Tailor(ctx context.Context, client llm.Client, cfg llm.Config, curriculumMD, jdText string) (string, error) {
	if strings.TrimSpace(curriculumMD) == "" {
		return "", ErrEmptyCurriculum
	}
	if strings.TrimSpace(jdText) == "" {
		return "", ErrEmptyJobDescription
	}

	req := llm.CompletionRequest{
		Messages: []llm.Message{
			{Role: "system", Content: SystemPrompt},
			{Role: "user", Content: BuildUserPrompt(curriculumMD, jdText)},
		},
		Temperature: 0.3,
	}

	raw, err := client.Complete(ctx, cfg, req)
	if err != nil {
		return "", err
	}

	return CleanMarkdownOutput(raw), nil
}
