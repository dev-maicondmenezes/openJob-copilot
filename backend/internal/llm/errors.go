package llm

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"regexp"
	"strconv"
	"strings"
)

// Standard error codes following ADR 0021 and Backlog B6.
const (
	CodeUnauthorized  = "llm.unauthorized"
	CodeModelNotFound = "llm.model_not_found"
	CodeRateLimited   = "llm.rate_limited"
	CodeServerError   = "llm.server_error"
	CodeNetworkError  = "llm.network_error"
	CodeDecryptError  = "llm.decrypt_error"
	CodeNotConfigured = "llm.not_configured"
	CodeUnknownError  = "llm.unknown"
)

// FriendlyError provides user-facing error details with code, readable message,
// suggested action and HTTP status, stripping raw payloads and stacks (B6).
type FriendlyError struct {
	Code       string `json:"code"`
	HTTPStatus int    `json:"http_status"`
	Message    string `json:"message"`
	Suggestion string `json:"suggestion"`
	Cause      error  `json:"-"`
}

func (e *FriendlyError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s (%v)", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func (e *FriendlyError) Unwrap() error {
	return e.Cause
}

var httpStatusRegex = regexp.MustCompile(`\bHTTP\s+(\d{3})\b`)

// ClassifyError inspects an error from LLM operations and returns a FriendlyError.
// Any secret passed is scrubbed via Redact before logging or exposing.
func ClassifyError(err error, secret string) *FriendlyError {
	if err == nil {
		return nil
	}

	// Redact secret from Cause and error string
	safeErr := redact(err, secret)
	errStr := safeErr.Error()
	lower := strings.ToLower(errStr)

	// Sentinel: Not configured
	if errors.Is(err, ErrNotConfigured) || strings.Contains(lower, "not configured") {
		return &FriendlyError{
			Code:       CodeNotConfigured,
			HTTPStatus: http.StatusPreconditionRequired,
			Message:    "AI provider not configured. Add your API key in Settings.",
			Suggestion: "Configure your provider API key in Settings > AI.",
			Cause:      safeErr,
		}
	}

	// Master key mismatch or decryption failure
	if strings.Contains(lower, "key mismatch") || strings.Contains(lower, "decrypt") || strings.Contains(lower, "cipher") {
		return &FriendlyError{
			Code:       CodeDecryptError,
			HTTPStatus: http.StatusBadRequest,
			Message:    "Master key changed or stored key could not be decrypted.",
			Suggestion: "Reconfigure your API key in Settings > AI.",
			Cause:      safeErr,
		}
	}

	// Context cancellation or timeout
	if errors.Is(err, context.Canceled) {
		return &FriendlyError{
			Code:       CodeNetworkError,
			HTTPStatus: http.StatusRequestTimeout,
			Message:    "Request was cancelled.",
			Suggestion: "Try again when ready.",
			Cause:      safeErr,
		}
	}
	if errors.Is(err, context.DeadlineExceeded) || strings.Contains(lower, "timeout") || strings.Contains(lower, "deadline exceeded") {
		return &FriendlyError{
			Code:       CodeNetworkError,
			HTTPStatus: http.StatusGatewayTimeout,
			Message:    "Could not reach the AI provider (timeout).",
			Suggestion: "Check your internet connection and provider base URL.",
			Cause:      safeErr,
		}
	}

	// Network-level errors
	var netErr net.Error
	if errors.As(err, &netErr) || strings.Contains(lower, "dial tcp") || strings.Contains(lower, "no such host") || strings.Contains(lower, "connection refused") {
		return &FriendlyError{
			Code:       CodeNetworkError,
			HTTPStatus: http.StatusBadGateway,
			Message:    "Network error reaching AI provider.",
			Suggestion: "Check your internet connection and provider base URL in Settings.",
			Cause:      safeErr,
		}
	}

	// Extract HTTP status code if present
	var statusCode int
	if m := httpStatusRegex.FindStringSubmatch(errStr); len(m) > 1 {
		statusCode, _ = strconv.Atoi(m[1])
	}

	switch {
	case statusCode == 401 || statusCode == 403 || strings.Contains(lower, "unauthorized") || strings.Contains(lower, "invalid_api_key") || strings.Contains(lower, "forbidden"):
		return &FriendlyError{
			Code:       CodeUnauthorized,
			HTTPStatus: http.StatusUnauthorized,
			Message:    "Invalid or expired API key, or missing model permissions.",
			Suggestion: "Generate a new key in your provider console and update Settings.",
			Cause:      safeErr,
		}

	case statusCode == 404 || strings.Contains(lower, "model_not_found") || strings.Contains(lower, "does not exist") || strings.Contains(lower, "model not found"):
		return &FriendlyError{
			Code:       CodeModelNotFound,
			HTTPStatus: http.StatusNotFound,
			Message:    "Model not available for your account or key.",
			Suggestion: "Verify the model name or select a supported model in Settings.",
			Cause:      safeErr,
		}

	case statusCode == 429 || strings.Contains(lower, "rate limit") || strings.Contains(lower, "quota") || strings.Contains(lower, "too many requests"):
		return &FriendlyError{
			Code:       CodeRateLimited,
			HTTPStatus: http.StatusTooManyRequests,
			Message:    "Quota or rate limit reached on the AI provider.",
			Suggestion: "Wait a few moments or switch to a different model/provider.",
			Cause:      safeErr,
		}

	case (statusCode >= 500 && statusCode < 600) || strings.Contains(lower, "server error") || strings.Contains(lower, "overloaded") || strings.Contains(lower, "high demand"):
		return &FriendlyError{
			Code:       CodeServerError,
			HTTPStatus: http.StatusBadGateway,
			Message:    "AI provider is temporarily overloaded or experiencing an outage.",
			Suggestion: "Try again in a few moments or choose another model.",
			Cause:      safeErr,
		}

	default:
		return &FriendlyError{
			Code:       CodeUnknownError,
			HTTPStatus: http.StatusInternalServerError,
			Message:    "An unexpected error occurred while communicating with the AI provider.",
			Suggestion: "Check the server logs or test your connection in Settings.",
			Cause:      safeErr,
		}
	}
}
