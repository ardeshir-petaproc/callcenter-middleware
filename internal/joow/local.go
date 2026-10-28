package joow

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"log/slog"
	"strings"
	"time"
)

// Local is a self-contained Joow stand-in backed by the joow_accounts table.
// Tokens look like "jdev.<account id>.<hmac>" and are minted by MintToken.
type Local struct {
	db     *sql.DB
	secret []byte
}

// NewLocal creates the development provider.
func NewLocal(db *sql.DB, secret string) *Local { return &Local{db: db, secret: []byte(secret)} }

// Name implements Provider.
func (l *Local) Name() string { return "local" }

func (l *Local) sign(id string) string {
	mac := hmac.New(sha256.New, l.secret)
	mac.Write([]byte("joow-dev:" + id))
	return hex.EncodeToString(mac.Sum(nil))[:32]
}

// MintToken issues a development session token for an account.
func (l *Local) MintToken(id string) string { return "jdev." + id + "." + l.sign(id) }

// VerifyToken implements Provider.
func (l *Local) VerifyToken(ctx context.Context, token string) (*Account, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 || parts[0] != "jdev" || !hmac.Equal([]byte(parts[2]), []byte(l.sign(parts[1]))) {
		return nil, ErrInvalidToken
	}
	return l.Get(ctx, parts[1])
}

func (l *Local) scan(rows *sql.Rows) ([]Account, error) {
	defer rows.Close()
	out := []Account{}
	for rows.Next() {
		var a Account
		if err := rows.Scan(&a.ID, &a.Name, &a.Email, &a.Phone, &a.AvatarURL); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// Search implements Provider.
func (l *Local) Search(ctx context.Context, query string, limit int) ([]Account, error) {
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	like := "%" + strings.ToLower(query) + "%"
	rows, err := l.db.QueryContext(ctx, `SELECT id, name, email, phone, avatar_url FROM joow_accounts
		WHERE lower(name) LIKE ? OR lower(email) LIKE ? OR phone LIKE ? ORDER BY name LIMIT ?`, like, like, like, limit)
	if err != nil {
		return nil, err
	}
	return l.scan(rows)
}

// Get implements Provider.
func (l *Local) Get(ctx context.Context, id string) (*Account, error) {
	rows, err := l.db.QueryContext(ctx, `SELECT id, name, email, phone, avatar_url FROM joow_accounts WHERE id = ?`, id)
	if err != nil {
		return nil, err
	}
	list, err := l.scan(rows)
	if err != nil {
		return nil, err
	}
	if len(list) == 0 {
		return nil, ErrNotFound
	}
	return &list[0], nil
}

// Upsert creates or updates a local account (seeding and tests).
func (l *Local) Upsert(ctx context.Context, a Account) error {
	if a.ID == "" {
		return errors.New("joow: account id required")
	}
	_, err := l.db.ExecContext(ctx, `INSERT INTO joow_accounts (id, name, email, phone, avatar_url, created_at)
		VALUES (?,?,?,?,?,?) ON CONFLICT(id) DO UPDATE SET name = excluded.name, email = excluded.email,
		phone = excluded.phone, avatar_url = excluded.avatar_url`, a.ID, a.Name, a.Email, a.Phone, a.AvatarURL,
		time.Now().UnixMilli())
	return err
}

// Notify implements Provider by logging (there is no device to reach).
func (l *Local) Notify(_ context.Context, accountID string, n Notification) error {
	slog.Info("Joow notification (local provider)", "account", accountID, "kind", n.Kind, "title", n.Title)
	return nil
}
