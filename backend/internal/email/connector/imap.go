package connector

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
	"github.com/maicondmenezes/openjob/internal/email"
)

// IMAPConnector connects to standard IMAP mailboxes.
type IMAPConnector struct {
	cfg email.IMAPAuthConfig
}

// NewIMAPConnector creates a new IMAP connector.
func NewIMAPConnector(cfg email.IMAPAuthConfig) *IMAPConnector {
	if cfg.Port <= 0 {
		if cfg.UseTLS {
			cfg.Port = 993
		} else {
			cfg.Port = 143
		}
	}
	return &IMAPConnector{cfg: cfg}
}

func (c *IMAPConnector) dial() (*imapclient.Client, error) {
	addr := net.JoinHostPort(c.cfg.Host, strconv.Itoa(c.cfg.Port))
	opts := &imapclient.Options{}

	if c.cfg.UseTLS {
		tlsConfig := &tls.Config{
			ServerName: c.cfg.Host,
			MinVersion: tls.VersionTLS12,
		}
		return imapclient.DialTLS(addr, &imapclient.Options{TLSConfig: tlsConfig})
	}

	if c.cfg.UseStartTLS {
		tlsConfig := &tls.Config{
			ServerName: c.cfg.Host,
			MinVersion: tls.VersionTLS12,
		}
		return imapclient.DialStartTLS(addr, &imapclient.Options{TLSConfig: tlsConfig})
	}

	return imapclient.DialInsecure(addr, opts)
}

func (c *IMAPConnector) TestConnection(ctx context.Context) error {
	client, err := c.dial()
	if err != nil {
		return fmt.Errorf("imap: connect to %s:%d: %w", c.cfg.Host, c.cfg.Port, err)
	}
	defer client.Close()

	if err := client.Login(c.cfg.Username, c.cfg.Password).Wait(); err != nil {
		return fmt.Errorf("imap: login failed: %w", err)
	}

	folder := "INBOX"
	if _, err := client.Select(folder, nil).Wait(); err != nil {
		return fmt.Errorf("imap: select %s failed: %w", folder, err)
	}

	return nil
}

func (c *IMAPConnector) FetchNewMessages(ctx context.Context, cursor string, folder string, limit int) ([]email.RawMessage, string, error) {
	if folder == "" {
		folder = "INBOX"
	}
	if limit <= 0 {
		limit = 50
	}

	client, err := c.dial()
	if err != nil {
		return nil, cursor, fmt.Errorf("imap: connect failed: %w", err)
	}
	defer client.Close()

	if err := client.Login(c.cfg.Username, c.cfg.Password).Wait(); err != nil {
		return nil, cursor, fmt.Errorf("imap: login failed: %w", err)
	}

	selectData, err := client.Select(folder, nil).Wait()
	if err != nil {
		return nil, cursor, fmt.Errorf("imap: select %s failed: %w", folder, err)
	}

	if selectData.NumMessages == 0 {
		return nil, cursor, nil
	}

	var lastUID uint32
	if cursor != "" {
		if u, err := strconv.ParseUint(cursor, 10, 32); err == nil {
			lastUID = uint32(u)
		}
	}

	// Search for messages with UID > lastUID
	criteria := &imap.SearchCriteria{}
	if lastUID > 0 {
		var uidSet imap.UIDSet
		uidSet.AddRange(imap.UID(lastUID+1), imap.UID(4294967295))
		criteria.UID = []imap.UIDSet{uidSet}
	} else {
		// First sync: only fetch recent messages (last 30 days) to avoid downloading entire inboxes
		thirtyDaysAgo := time.Now().UTC().AddDate(0, 0, -30)
		criteria.Since = thirtyDaysAgo
	}

	searchCmd := client.UIDSearch(criteria, nil)
	searchData, err := searchCmd.Wait()
	if err != nil {
		return nil, cursor, fmt.Errorf("imap: search failed: %w", err)
	}

	allUIDs := searchData.AllUIDs()
	if len(allUIDs) == 0 {
		return nil, cursor, nil
	}

	if len(allUIDs) > limit {
		allUIDs = allUIDs[len(allUIDs)-limit:]
	}

	var fetchUIDSet imap.UIDSet
	for _, u := range allUIDs {
		fetchUIDSet.AddNum(u)
	}

	bodySection := &imap.FetchItemBodySection{}
	fetchOptions := &imap.FetchOptions{
		UID:         true,
		BodySection: []*imap.FetchItemBodySection{bodySection},
	}

	fetchCmd := client.Fetch(fetchUIDSet, fetchOptions)
	defer fetchCmd.Close()

	var messages []email.RawMessage
	maxSeenUID := lastUID

	for {
		msgData := fetchCmd.Next()
		if msgData == nil {
			break
		}

		buf, err := msgData.Collect()
		if err != nil {
			continue
		}

		if uint32(buf.UID) > maxSeenUID {
			maxSeenUID = uint32(buf.UID)
		}

		rawBytes := buf.FindBodySection(bodySection)
		if len(rawBytes) == 0 {
			continue
		}

		parsed, err := ParseMIME(bytes.NewReader(rawBytes))
		if err != nil {
			continue
		}

		messages = append(messages, *parsed)
	}

	newCursor := strconv.FormatUint(uint64(maxSeenUID), 10)
	return messages, newCursor, nil
}
