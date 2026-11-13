package store

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// BYOK encryption (spec §4/§6): API keys are sealed with AES-256-GCM using a
// locally generated master key stored in data/.master.key with mode 0600.
// There is no master password in the MVP (no PBKDF2) — see docs/adr/0004.
//
// If .master.key is lost or regenerated, previously stored ciphertext becomes
// undecryptable. This is a deliberate, documented trade-off: the error is
// explicit and the user simply re-enters their API key (nothing else in the
// database depends on the master key).

// MasterKeyFileName is the name of the master key file, always located next
// to the database file so a single volume backup carries both.
const MasterKeyFileName = ".master.key"

// MasterKeyPath derives the master key location from the DB path:
// /data/openjob.db → /data/.master.key.
func MasterKeyPath(dbPath string) string {
	return filepath.Join(filepath.Dir(dbPath), MasterKeyFileName)
}

// ErrKeyMismatch is returned when ciphertext cannot be opened with the
// current master key (e.g. the key file was regenerated after the value was
// stored). The recovery is always: reconfigure the affected setting.
var ErrKeyMismatch = errors.New("store: master key does not match stored ciphertext")

// LoadOrCreateMasterKey reads the 32-byte master key at path, creating it on
// first run with mode 0600. A 32-byte file is accepted as-is; any other
// existing content is treated as corruption and fails loudly instead of
// silently regenerating (which would orphan all stored ciphertext).
func LoadOrCreateMasterKey(path string) ([]byte, error) {
	if dir := filepath.Dir(path); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("store: create key dir %s: %w", dir, err)
		}
	}

	if key, err := os.ReadFile(path); err == nil {
		if len(key) != 32 {
			return nil, fmt.Errorf(
				"store: %s exists with %d bytes (want 32): refusing to regenerate — reconfigure encrypted settings or restore the file",
				path, len(key))
		}
		return key, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("store: read %s: %w", path, err)
	}

	key := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return nil, fmt.Errorf("store: generate master key: %w", err)
	}
	// O_CREAT is implicit; 0600 must be honored on create (spec: vault secret).
	if err := os.WriteFile(path, key, 0o600); err != nil {
		return nil, fmt.Errorf("store: write %s: %w", path, err)
	}
	// Best effort: enforce mode even if the file pre-existed another way.
	if err := os.Chmod(path, 0o600); err != nil {
		return nil, fmt.Errorf("store: chmod %s: %w", path, err)
	}
	return key, nil
}

// Encrypt seals plaintext with AES-256-GCM and returns base64(nonce‖ciphertext).
// The value is self-contained (nonce is random per call) and safe to store in
// the settings table as TEXT.
func Encrypt(key, plaintext []byte) (string, error) {
	if len(key) != 32 {
		return "", fmt.Errorf("store: master key must be 32 bytes, got %d", len(key))
	}
	gcm, err := newGCM(key)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("store: nonce: %w", err)
	}
	sealed := gcm.Seal(nonce, nonce, plaintext, nil)
	return base64.StdEncoding.EncodeToString(sealed), nil
}

// Decrypt opens a value produced by Encrypt.
func Decrypt(key []byte, encoded string) ([]byte, error) {
	if len(key) != 32 {
		return nil, fmt.Errorf("store: master key must be 32 bytes, got %d", len(key))
	}
	raw, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrKeyMismatch, err)
	}
	gcm, err := newGCM(key)
	if err != nil {
		return nil, err
	}
	nonceSize := gcm.NonceSize()
	if len(raw) < nonceSize {
		return nil, ErrKeyMismatch
	}
	plaintext, err := gcm.Open(nil, raw[:nonceSize], raw[nonceSize:], nil)
	if err != nil {
		return nil, ErrKeyMismatch
	}
	return plaintext, nil
}

// EncryptString / DecryptString are convenience wrappers for settings values.
func EncryptString(key []byte, plaintext string) (string, error) {
	return Encrypt(key, []byte(plaintext))
}

func DecryptString(key []byte, encoded string) (string, error) {
	plaintext, err := Decrypt(key, encoded)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}

// newGCM builds the AES-256-GCM AEAD from the 32-byte master key.
func newGCM(key []byte) (cipher.AEAD, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("store: aes cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("store: gcm: %w", err)
	}
	return gcm, nil
}
