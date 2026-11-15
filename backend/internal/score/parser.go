package score

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// Result carries the LLM's fit assessment after parsing and clamping.
type Result struct {
	Score  int    // 0–100, always clamped
	Reason string // human-readable justification (may be empty on fallback)
}

// ErrParse reports that the model response could not be interpreted as a
// score. The caller maps it to a legible 502 and leaves score_fit NULL.
type ErrParse struct{ detail string }

func (e *ErrParse) Error() string {
	return fmt.Sprintf("score: resposta da IA sem score interpretável: %s", e.detail)
}

// jsonShape mirrors the strict format demanded by the system prompt.
type jsonShape struct {
	Score  json.Number `json:"score"`
	Reason string      `json:"reason"`
}

// scoreRe is the fallback extractor: a score number next to the word "score"
// in whatever shape the model wrote it ("score": 78, score: 78, Score - 78...).
var scoreRe = regexp.MustCompile(`(?i)"?score"?\s*[:\-–=]?\s*([0-9]+(?:[.,][0-9]+)?)`)

// reasonRe is the fallback extractor for the reason string field.
var reasonRe = regexp.MustCompile(`(?is)"?reason"?\s*[:\-–=]\s*"([^"]+)"`)

// fenceRe strips a leading/trailing markdown code fence wrapper.
var fenceRe = regexp.MustCompile("(?s)^\\s*```[a-zA-Z]*\\s*(.*?)\\s*```\\s*$")

// ParseScore extracts {"score": int, "reason": str} from the model output.
// Strategy (spec CP5): strict JSON first, then substring between the outer
// braces, then regex fallback. The score is always clamped to 0–100.
func ParseScore(raw string) (Result, error) {
	s := strings.TrimSpace(raw)
	if m := fenceRe.FindStringSubmatch(s); m != nil {
		s = strings.TrimSpace(m[1])
	}

	if res, ok := decodeJSONScore(s); ok {
		return res, nil
	}

	// Substring between the first '{' and the last '}' — the model may add
	// prose around the JSON despite instructions.
	if open := strings.Index(s, "{"); open >= 0 {
		if close := strings.LastIndex(s, "}"); close > open {
			if res, ok := decodeJSONScore(s[open : close+1]); ok {
				return res, nil
			}
		}
	}

	// Regex fallback: score alone is enough; reason is best-effort.
	if m := scoreRe.FindStringSubmatch(s); m != nil {
		score, clampErr := clampNumber(m[1])
		if clampErr == nil {
			res := Result{Score: score}
			if rm := reasonRe.FindStringSubmatch(s); rm != nil {
				res.Reason = strings.TrimSpace(rm[1])
			}
			return res, nil
		}
	}

	return Result{}, &ErrParse{detail: truncateDetail(s)}
}

func decodeJSONScore(s string) (Result, bool) {
	var shape jsonShape
	if err := json.Unmarshal([]byte(s), &shape); err != nil {
		return Result{}, false
	}
	score, err := clampNumber(shape.Score.String())
	if err != nil {
		return Result{}, false
	}
	return Result{Score: score, Reason: strings.TrimSpace(shape.Reason)}, true
}

// clampNumber parses a numeric string (int, float or "78,5") and clamps it
// to 0–100 (spec CP5: "clamp 0–100").
func clampNumber(v string) (int, error) {
	v = strings.ReplaceAll(strings.TrimSpace(v), ",", ".")
	if v == "" {
		return 0, fmt.Errorf("empty number")
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return 0, err
	}
	switch {
	case f < 0:
		return 0, nil
	case f > 100:
		return 100, nil
	}
	return int(f), nil
}

// truncateDetail keeps error messages short — the raw output can be huge.
func truncateDetail(s string) string {
	s = strings.TrimSpace(s)
	if len(s) > 200 {
		s = s[:200] + "…"
	}
	if s == "" {
		return "resposta vazia"
	}
	return s
}
