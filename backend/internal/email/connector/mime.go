package connector

import (
	"io"
	"mime"
	"mime/multipart"
	"net/mail"
	"regexp"
	"strings"
	"time"

	"github.com/maicondmenezes/openjob/internal/email"
)

var htmlTagRegex = regexp.MustCompile(`<[^>]*>`)

// ParseMIME parses an RFC 5322 MIME message into a RawMessage.
func ParseMIME(r io.Reader) (*email.RawMessage, error) {
	msg, err := mail.ReadMessage(r)
	if err != nil {
		return nil, err
	}

	header := msg.Header
	msgID := strings.TrimSpace(header.Get("Message-ID"))
	threadID := strings.TrimSpace(header.Get("In-Reply-To"))
	if threadID == "" {
		refs := strings.Fields(header.Get("References"))
		if len(refs) > 0 {
			threadID = refs[0]
		}
	}
	if threadID == "" {
		threadID = msgID
	}

	fromAddr := ""
	fromName := ""
	if fromList, err := header.AddressList("From"); err == nil && len(fromList) > 0 {
		fromAddr = strings.ToLower(fromList[0].Address)
		fromName = fromList[0].Name
	} else {
		fromAddr = strings.ToLower(header.Get("From"))
	}

	dec := new(mime.WordDecoder)
	subject, _ := dec.DecodeHeader(header.Get("Subject"))
	if subject == "" {
		subject = header.Get("Subject")
	}

	var receivedAt time.Time
	if dateStr := header.Get("Date"); dateStr != "" {
		if t, err := mail.ParseDate(dateStr); err == nil {
			receivedAt = t.UTC()
		}
	}
	if receivedAt.IsZero() {
		receivedAt = time.Now().UTC()
	}

	mediaType, params, err := mime.ParseMediaType(header.Get("Content-Type"))
	body := ""
	if err == nil && strings.HasPrefix(mediaType, "multipart/") {
		body = extractMultipartBody(msg.Body, params["boundary"])
	} else {
		rawBytes, _ := io.ReadAll(io.LimitReader(msg.Body, int64(email.MaxBodyLength*2)))
		if strings.Contains(mediaType, "html") {
			body = stripHTML(string(rawBytes))
		} else {
			body = string(rawBytes)
		}
	}

	return &email.RawMessage{
		MessageID:   msgID,
		ThreadID:    threadID,
		FromAddress: fromAddr,
		FromName:    fromName,
		Subject:     subject,
		ReceivedAt:  receivedAt,
		BodyText:    email.TruncateBody(strings.TrimSpace(body)),
	}, nil
}

func extractMultipartBody(r io.Reader, boundary string) string {
	if boundary == "" {
		data, _ := io.ReadAll(io.LimitReader(r, int64(email.MaxBodyLength)))
		return string(data)
	}

	mr := multipart.NewReader(r, boundary)
	var plainText string
	var htmlText string

	for {
		part, err := mr.NextPart()
		if err != nil {
			break
		}
		partMediaType, _, _ := mime.ParseMediaType(part.Header.Get("Content-Type"))
		partData, err := io.ReadAll(io.LimitReader(part, int64(email.MaxBodyLength)))
		if err != nil {
			continue
		}

		if strings.HasPrefix(partMediaType, "text/plain") && plainText == "" {
			plainText = string(partData)
		} else if strings.HasPrefix(partMediaType, "text/html") && htmlText == "" {
			htmlText = stripHTML(string(partData))
		}
	}

	if plainText != "" {
		return plainText
	}
	return htmlText
}

func stripHTML(html string) string {
	clean := htmlTagRegex.ReplaceAllString(html, " ")
	clean = strings.ReplaceAll(clean, "&nbsp;", " ")
	clean = strings.ReplaceAll(clean, "&amp;", "&")
	clean = strings.ReplaceAll(clean, "&lt;", "<")
	clean = strings.ReplaceAll(clean, "&gt;", ">")
	return strings.Join(strings.Fields(clean), " ")
}
