package auth

import (
	"strings"
	"testing"
	"time"

	"callcenter-middleware/internal/domain"
)

func TestIssueAndVerify(t *testing.T) {
	iss := NewIssuer("secret", time.Hour)
	tok, _, err := iss.Issue(&domain.User{ID: 7, Username: "ana", Role: domain.RoleSupervisor})
	if err != nil {
		t.Fatal(err)
	}
	c, err := iss.Verify(tok)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if c.UserID != 7 || c.Role != domain.RoleSupervisor || c.Username != "ana" {
		t.Fatalf("unexpected claims %+v", c)
	}
}

func TestVerifyRejectsForgedAndExpired(t *testing.T) {
	iss := NewIssuer("secret", time.Hour)
	tok, _, _ := iss.Issue(&domain.User{ID: 1, Role: domain.RoleAgent})
	if _, err := NewIssuer("other", time.Hour).Verify(tok); err == nil {
		t.Fatal("expected forged token to fail")
	}
	parts := strings.Split(tok, ".")
	if _, err := iss.Verify(parts[0] + "." + parts[1] + "x." + parts[2]); err == nil {
		t.Fatal("expected tampered payload to fail")
	}
	expired := NewIssuer("secret", -time.Second)
	old, _, _ := expired.Issue(&domain.User{ID: 1})
	if _, err := expired.Verify(old); err == nil {
		t.Fatal("expected expired token to fail")
	}
}

func TestPasswordHash(t *testing.T) {
	h, err := HashPassword("pw123456")
	if err != nil {
		t.Fatal(err)
	}
	if !CheckPassword(h, "pw123456") || CheckPassword(h, "nope") {
		t.Fatal("password check mismatch")
	}
}
