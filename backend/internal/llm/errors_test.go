package llm

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"
)

func TestClassifyError(t *testing.T) {
	cases := []struct {
		name       string
		err        error
		secret     string
		wantCode   string
		wantStatus int
	}{
		{
			name:       "nil error",
			err:        nil,
			wantCode:   "",
			wantStatus: 0,
		},
		{
			name:       "not configured",
			err:        ErrNotConfigured,
			wantCode:   CodeNotConfigured,
			wantStatus: http.StatusPreconditionRequired,
		},
		{
			name:       "unauthorized 401",
			err:        fmt.Errorf("llm: ping https://api.openai.com/v1: HTTP 401: Invalid API key provided: sk-12345"),
			secret:     "sk-12345",
			wantCode:   CodeUnauthorized,
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "forbidden 403",
			err:        fmt.Errorf("llm: ping https://api.openai.com/v1: HTTP 403: Forbidden access"),
			wantCode:   CodeUnauthorized,
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "model not found 404",
			err:        fmt.Errorf("llm: completion https://api.openai.com/v1: HTTP 404: The model `gpt-fake` does not exist"),
			wantCode:   CodeModelNotFound,
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "rate limit 429",
			err:        fmt.Errorf("llm: completion https://api.openai.com/v1: HTTP 429: You exceeded your current quota"),
			wantCode:   CodeRateLimited,
			wantStatus: http.StatusTooManyRequests,
		},
		{
			name:       "server overloaded 503",
			err:        fmt.Errorf("llm: completion https://api.openai.com/v1: HTTP 503: Service Unavailable: high demand"),
			wantCode:   CodeServerError,
			wantStatus: http.StatusBadGateway,
		},
		{
			name:       "context deadline exceeded",
			err:        context.DeadlineExceeded,
			wantCode:   CodeNetworkError,
			wantStatus: http.StatusGatewayTimeout,
		},
		{
			name:       "key mismatch decrypt",
			err:        errors.New("store: decrypt failed: key mismatch"),
			wantCode:   CodeDecryptError,
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fe := ClassifyError(tc.err, tc.secret)
			if tc.err == nil {
				if fe != nil {
					t.Fatalf("expected nil FriendlyError, got %+v", fe)
				}
				return
			}
			if fe == nil {
				t.Fatalf("expected FriendlyError, got nil")
			}
			if fe.Code != tc.wantCode {
				t.Errorf("Code = %q, want %q", fe.Code, tc.wantCode)
			}
			if fe.HTTPStatus != tc.wantStatus {
				t.Errorf("HTTPStatus = %d, want %d", fe.HTTPStatus, tc.wantStatus)
			}
			if tc.secret != "" {
				errStr := fe.Error()
				if strings.Contains(errStr, tc.secret) {
					t.Errorf("secret leaked in Error(): %s", errStr)
				}
			}
		})
	}
}
