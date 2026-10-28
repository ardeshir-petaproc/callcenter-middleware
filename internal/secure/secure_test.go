package secure

import (
	"bytes"
	"errors"
	"path/filepath"
	"testing"
)

func TestSealOpenPerTenant(t *testing.T) {
	b := New("master")
	ct, err := b.Seal(1, []byte("secret policy"))
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(ct, []byte("secret")) {
		t.Fatal("ciphertext leaks plaintext")
	}
	if pt, err := b.Open(1, ct); err != nil || string(pt) != "secret policy" {
		t.Fatalf("open: %v %q", err, pt)
	}
	if _, err := b.Open(2, ct); !errors.Is(err, ErrDecrypt) {
		t.Fatal("another tenant's key must not decrypt")
	}
	if _, err := New("other").Open(1, ct); !errors.Is(err, ErrDecrypt) {
		t.Fatal("another master key must not decrypt")
	}
	s, _ := b.SealString(3, "imap-password")
	if got, err := b.OpenString(3, s); err != nil || got != "imap-password" {
		t.Fatalf("string round trip: %v %q", err, got)
	}
}

func TestLoadOrCreatePersistsKey(t *testing.T) {
	path := filepath.Join(t.TempDir(), "k")
	a, err := LoadOrCreate("", path)
	if err != nil {
		t.Fatal(err)
	}
	ct, _ := a.Seal(1, []byte("x"))
	b, _ := LoadOrCreate("", path)
	if _, err := b.Open(1, ct); err != nil {
		t.Fatal("key file must be reused across restarts")
	}
}
