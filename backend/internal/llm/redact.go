package llm

import (
	"strings"
)

// redacted masks a secret (the API key) in an error message while preserving
// the wrapped error chain — errors.Is/As keep working for sentinel errors
// like ErrNotConfigured.
//
// Why: provider error bodies are echoed verbatim into returned errors, which
// handlers both log and forward to the UI. Some providers include the
// credentials in their error messages; redaction at the client boundary
// guarantees the key can never reach logs or HTTP responses, even from code
// written later that forgets to scrub.
type redacted struct {
	err    error
	secret string
}

func (r *redacted) Error() string {
	return strings.ReplaceAll(r.err.Error(), r.secret, "***")
}

func (r *redacted) Unwrap() error { return r.err }

// redact returns err with every occurrence of secret masked. An empty secret
// (not configured) is returned untouched — ReplaceAll with an empty pattern
// would corrupt the message.
func redact(err error, secret string) error {
	if err == nil || secret == "" {
		return err
	}
	return &redacted{err: err, secret: secret}
}

// Redact is exported for callers that interpolate secrets into their own
// error strings (e.g. handlers showing provider messages in the UI).
func Redact(msg, secret string) string {
	if secret == "" {
		return msg
	}
	return strings.ReplaceAll(msg, secret, "***")
}

var _ error = (*redacted)(nil)
