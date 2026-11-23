package connector

import (
	"context"

	"github.com/maicondmenezes/openjob/internal/email"
)

// MockConnector implements Connector for unit testing.
type MockConnector struct {
	Messages   []email.RawMessage
	NewCursor  string
	FailTest   bool
	FailFetch  bool
	ErrMessage error
}

func (m *MockConnector) TestConnection(ctx context.Context) error {
	if m.FailTest {
		return m.ErrMessage
	}
	return nil
}

func (m *MockConnector) FetchNewMessages(ctx context.Context, cursor string, folder string, limit int) ([]email.RawMessage, string, error) {
	if m.FailFetch {
		return nil, cursor, m.ErrMessage
	}
	return m.Messages, m.NewCursor, nil
}

var _ email.Connector = (*MockConnector)(nil)
var _ email.Connector = (*IMAPConnector)(nil)
var _ email.Connector = (*GmailConnector)(nil)
