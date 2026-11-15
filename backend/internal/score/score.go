// Package score computes the fit score (0–100) of the base curriculum
// against a job description via the LLM (Checkpoint 5). It mirrors the
// internal/tailor package: a versioned prompt, input normalization and a
// strict output format with a tolerant parser.
package score

import (
	"context"
	"errors"
	"strings"

	"github.com/maicondmenezes/openjob/internal/llm"
)

var (
	// ErrEmptyCurriculum is returned when the base curriculum is missing or empty.
	ErrEmptyCurriculum = errors.New("score: currículo-base vazio")
	// ErrEmptyJobDescription is returned when the job description is missing or empty.
	ErrEmptyJobDescription = errors.New("score: descrição da vaga vazia")
)

// Score sends curriculum + JD to the LLM and returns the parsed, clamped
// fit assessment. The JD is normalized (CleanJD) before leaving the process.
func Score(ctx context.Context, client llm.Client, cfg llm.Config, curriculumMD, jdText string) (Result, error) {
	if strings.TrimSpace(curriculumMD) == "" {
		return Result{}, ErrEmptyCurriculum
	}
	if strings.TrimSpace(jdText) == "" {
		return Result{}, ErrEmptyJobDescription
	}

	req := llm.CompletionRequest{
		Messages: []llm.Message{
			{Role: "system", Content: SystemPrompt},
			{Role: "user", Content: BuildUserPrompt(curriculumMD, CleanJD(jdText))},
		},
		Temperature: 0.2,
	}

	raw, err := client.Complete(ctx, cfg, req)
	if err != nil {
		return Result{}, err
	}

	res, err := ParseScore(raw)
	if err != nil {
		return Result{}, err
	}
	return res, nil
}

// AsErrParse reports whether err is a score-parsing failure (as opposed to a
// provider/network failure), so handlers can produce a legible message.
func AsErrParse(err error) bool {
	var ep *ErrParse
	return errors.As(err, &ep)
}

// ParseErrorDetail returns the ErrParse detail when err is a parse failure.
func ParseErrorDetail(err error) string {
	var ep *ErrParse
	if errors.As(err, &ep) {
		return ep.detail
	}
	return ""
}
