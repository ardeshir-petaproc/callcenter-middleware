// Package secure encrypts data at rest (knowledge files, email bodies,
// connector credentials) with AES-256-GCM. Every tenant gets its own key
// derived from the master key, so ciphertext of one tenant can never be
// decrypted under another tenant's key.
package secure

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// ErrDecrypt is returned when ciphertext is corrupt or for another key.
var ErrDecrypt = errors.New("secure: cannot decrypt")

// Box encrypts and decrypts tenant data.
type Box struct {
	master []byte
}

// New derives the master key from secret (any length).
func New(secret string) *Box {
	sum := sha256.Sum256([]byte("callcenter-data-key:" + secret))
	return &Box{master: sum[:]}
}

// LoadOrCreate uses secret when set; otherwise it keeps a random key in
// keyFile (created with 0600) so encrypted data survives restarts.
func LoadOrCreate(secret, keyFile string) (*Box, error) {
	if secret != "" {
		return New(secret), nil
	}
	if raw, err := os.ReadFile(keyFile); err == nil && len(strings.TrimSpace(string(raw))) > 0 {
		return New(strings.TrimSpace(string(raw))), nil
	}
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(keyFile), 0o700); err != nil {
		return nil, err
	}
	if err := os.WriteFile(keyFile, []byte(hex.EncodeToString(buf)), 0o600); err != nil {
		return nil, err
	}
	slog.Warn("DATA_ENCRYPTION_KEY not set: generated a key file; back it up with the database", "path", keyFile)
	return New(hex.EncodeToString(buf)), nil
}

func (b *Box) aead(tenant int64) (cipher.AEAD, error) {
	mac := hmac.New(sha256.New, b.master)
	mac.Write([]byte("tenant:" + strconv.FormatInt(tenant, 10)))
	block, err := aes.NewCipher(mac.Sum(nil))
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(block)
}

// Seal encrypts plaintext for a tenant (nonce prepended).
func (b *Box) Seal(tenant int64, plaintext []byte) ([]byte, error) {
	a, err := b.aead(tenant)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, a.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}
	return a.Seal(nonce, nonce, plaintext, nil), nil
}

// Open decrypts data sealed for the same tenant.
func (b *Box) Open(tenant int64, data []byte) ([]byte, error) {
	a, err := b.aead(tenant)
	if err != nil {
		return nil, err
	}
	if len(data) < a.NonceSize() {
		return nil, ErrDecrypt
	}
	out, err := a.Open(nil, data[:a.NonceSize()], data[a.NonceSize():], nil)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrDecrypt, err)
	}
	return out, nil
}

// SealString encrypts a short secret to a printable token ("enc:<base64>").
func (b *Box) SealString(tenant int64, s string) (string, error) {
	raw, err := b.Seal(tenant, []byte(s))
	if err != nil {
		return "", err
	}
	return "enc:" + base64.RawStdEncoding.EncodeToString(raw), nil
}

// OpenString reverses SealString.
func (b *Box) OpenString(tenant int64, s string) (string, error) {
	if !strings.HasPrefix(s, "enc:") {
		return "", ErrDecrypt
	}
	raw, err := base64.RawStdEncoding.DecodeString(strings.TrimPrefix(s, "enc:"))
	if err != nil {
		return "", ErrDecrypt
	}
	out, err := b.Open(tenant, raw)
	return string(out), err
}
