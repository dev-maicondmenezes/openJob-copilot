package email

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"
)

// RawMessage represents an email fetched from a provider before classification and storage.
type RawMessage struct {
	MessageID   string    // RFC 5322 Message-ID
	ThreadID    string    // Thread / In-Reply-To identifier
	FromAddress string    // sender email
	FromName    string    // sender display name
	Subject     string    // message subject
	ReceivedAt  time.Time // timestamp of receipt
	BodyText    string    // plain text body (truncated to maxBodyLength)
}

const (
	MaxBodyLength = 20 * 1024 // ~20 KB limit to prevent inbox bloat and log leakage
)

// Connector connects to an email provider to fetch messages and test connectivity.
type Connector interface {
	TestConnection(ctx context.Context) error
	FetchNewMessages(ctx context.Context, cursor string, folder string, limit int) ([]RawMessage, string, error)
}

// GmailAuthConfig holds OAuth2 credentials for Gmail API access.
type GmailAuthConfig struct {
	ClientID     string    `json:"client_id"`
	ClientSecret string    `json:"client_secret"`
	RefreshToken string    `json:"refresh_token"`
	AccessToken  string    `json:"access_token,omitempty"`
	Expiry       time.Time `json:"expiry,omitempty"`
}

// IMAPAuthConfig holds connection credentials for generic IMAP access.
type IMAPAuthConfig struct {
	Host        string `json:"host"`
	Port        int    `json:"port"`
	Username    string `json:"username"`
	Password    string `json:"password"`
	UseTLS      bool   `json:"use_tls"`
	UseStartTLS bool   `json:"use_starttls"`
}

// EncryptAuthConfig serializes and encrypts an auth config using AES-256-GCM.
func EncryptAuthConfig(cfg any, masterKey []byte) ([]byte, error) {
	if len(masterKey) != 32 {
		return nil, errors.New("email: master key must be 32 bytes")
	}
	plaintext, err := json.Marshal(cfg)
	if err != nil {
		return nil, fmt.Errorf("email: marshal auth config: %w", err)
	}

	block, err := aes.NewCipher(masterKey)
	if err != nil {
		return nil, fmt.Errorf("email: new cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("email: new gcm: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("email: read nonce: %w", err)
	}

	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

// DecryptAuthConfig decrypts an AES-256-GCM encrypted auth config into the target struct.
func DecryptAuthConfig(ciphertext, masterKey []byte, target any) error {
	if len(masterKey) != 32 {
		return errors.New("email: master key must be 32 bytes")
	}
	block, err := aes.NewCipher(masterKey)
	if err != nil {
		return fmt.Errorf("email: new cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return fmt.Errorf("email: new gcm: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return errors.New("email: ciphertext too short")
	}

	nonce, encrypted := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, encrypted, nil)
	if err != nil {
		return fmt.Errorf("email: decrypt auth config: %w", err)
	}

	if err := json.Unmarshal(plaintext, target); err != nil {
		return fmt.Errorf("email: unmarshal auth config: %w", err)
	}
	return nil
}

// TruncateBody ensures body text stays within safe bounds.
func TruncateBody(text string) string {
	if len(text) <= MaxBodyLength {
		return text
	}
	return text[:MaxBodyLength]
}
