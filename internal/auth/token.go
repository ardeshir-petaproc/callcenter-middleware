// Package auth issues and verifies API tokens and hashes passwords.
package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"callcenter-middleware/internal/domain"

	"golang.org/x/crypto/bcrypt"
)

// ErrInvalidToken is returned for malformed, forged or expired tokens.
var ErrInvalidToken = errors.New("invalid token")

// Claims identify the authenticated user inside a token.
type Claims struct {
	UserID   int64       `json:"sub"`
	TenantID int64       `json:"tid"`
	Platform bool        `json:"pla,omitempty"`
	Username string      `json:"usr"`
	Role     domain.Role `json:"role"`
	Expires  int64       `json:"exp"`
	IssuedAt int64       `json:"iat"`
}

// Issuer signs and verifies HS256 JWTs.
type Issuer struct {
	secret []byte
	ttl    time.Duration
}

// NewIssuer creates a token issuer with the given HMAC secret and lifetime.
func NewIssuer(secret string, ttl time.Duration) *Issuer {
	return &Issuer{secret: []byte(secret), ttl: ttl}
}

var jwtHeader = base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))

// Issue creates a signed token for u.
func (i *Issuer) Issue(u *domain.User) (string, time.Time, error) {
	now := time.Now()
	exp := now.Add(i.ttl)
	payload, err := json.Marshal(Claims{
		UserID: u.ID, TenantID: u.TenantID, Platform: u.PlatformAdmin, Username: u.Username, Role: u.Role,
		Expires: exp.Unix(), IssuedAt: now.Unix(),
	})
	if err != nil {
		return "", time.Time{}, err
	}
	unsigned := jwtHeader + "." + base64.RawURLEncoding.EncodeToString(payload)
	return unsigned + "." + i.sign(unsigned), exp, nil
}

// Verify validates signature and expiry and returns the claims.
func (i *Issuer) Verify(token string) (*Claims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 || parts[0] != jwtHeader {
		return nil, ErrInvalidToken
	}
	expected := i.sign(parts[0] + "." + parts[1])
	if !hmac.Equal([]byte(expected), []byte(parts[2])) {
		return nil, ErrInvalidToken
	}
	raw, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, ErrInvalidToken
	}
	var c Claims
	if err := json.Unmarshal(raw, &c); err != nil {
		return nil, ErrInvalidToken
	}
	if time.Now().Unix() >= c.Expires {
		return nil, ErrInvalidToken
	}
	return &c, nil
}

func (i *Issuer) sign(s string) string {
	mac := hmac.New(sha256.New, i.secret)
	mac.Write([]byte(s))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

// HashPassword returns a bcrypt hash of pw.
func HashPassword(pw string) (string, error) {
	h, err := bcrypt.GenerateFromPassword([]byte(pw), bcrypt.DefaultCost)
	return string(h), err
}

// CheckPassword compares a bcrypt hash with a candidate password.
func CheckPassword(hash, pw string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(pw)) == nil
}
