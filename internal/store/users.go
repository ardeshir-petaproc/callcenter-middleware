package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"callcenter-middleware/internal/domain"
)

// ErrConflict is returned when a unique constraint would be violated.
var ErrConflict = errors.New("conflict")

const userCols = `id, username, display_name, email, role, extension, endpoint, skills, enabled, password_hash, created_at, updated_at`

func scanUser(sc interface{ Scan(...any) error }) (*domain.User, error) {
	var u domain.User
	var skills string
	var enabled int
	var created, updated int64
	if err := sc.Scan(&u.ID, &u.Username, &u.DisplayName, &u.Email, &u.Role, &u.Extension, &u.Endpoint,
		&skills, &enabled, &u.PasswordHash, &created, &updated); err != nil {
		return nil, err
	}
	_ = json.Unmarshal([]byte(skills), &u.Skills)
	if u.Skills == nil {
		u.Skills = []string{}
	}
	u.Enabled = enabled == 1
	u.CreatedAt, u.UpdatedAt = fromMS(created), fromMS(updated)
	return &u, nil
}

// ListUsers returns every user ordered by id.
func (s *Store) ListUsers(ctx context.Context) ([]domain.User, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT `+userCols+` FROM users ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.User{}
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *u)
	}
	return out, rows.Err()
}

// GetUser loads one user by id.
func (s *Store) GetUser(ctx context.Context, id int64) (*domain.User, error) {
	u, err := scanUser(s.db.QueryRowContext(ctx, `SELECT `+userCols+` FROM users WHERE id = ?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return u, err
}

// GetUserByUsername loads one user by username (case-insensitive).
func (s *Store) GetUserByUsername(ctx context.Context, username string) (*domain.User, error) {
	u, err := scanUser(s.db.QueryRowContext(ctx,
		`SELECT `+userCols+` FROM users WHERE lower(username) = lower(?)`, username))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return u, err
}

// CreateUser inserts a user; PasswordHash must already be set.
func (s *Store) CreateUser(ctx context.Context, u *domain.User) error {
	now := time.Now()
	skills, _ := json.Marshal(nonNil(u.Skills))
	res, err := s.db.ExecContext(ctx, `INSERT INTO users
		(username, display_name, email, role, extension, endpoint, skills, enabled, password_hash, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		u.Username, u.DisplayName, u.Email, u.Role, u.Extension, u.Endpoint, string(skills),
		boolInt(u.Enabled), u.PasswordHash, ms(now), ms(now))
	if err != nil {
		return wrapConflict(err)
	}
	u.ID, _ = res.LastInsertId()
	u.CreatedAt, u.UpdatedAt = now, now
	return nil
}

// UpdateUser saves profile fields; an empty PasswordHash keeps the old one.
func (s *Store) UpdateUser(ctx context.Context, u *domain.User) error {
	skills, _ := json.Marshal(nonNil(u.Skills))
	res, err := s.db.ExecContext(ctx, `UPDATE users SET username = ?, display_name = ?, email = ?, role = ?,
		extension = ?, endpoint = ?, skills = ?, enabled = ?,
		password_hash = CASE WHEN ? = '' THEN password_hash ELSE ? END, updated_at = ? WHERE id = ?`,
		u.Username, u.DisplayName, u.Email, u.Role, u.Extension, u.Endpoint, string(skills),
		boolInt(u.Enabled), u.PasswordHash, u.PasswordHash, ms(time.Now()), u.ID)
	if err != nil {
		return wrapConflict(err)
	}
	return requireAffected(res)
}

// DeleteUser removes a user and their queue memberships.
func (s *Store) DeleteUser(ctx context.Context, id int64) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM users WHERE id = ?`, id)
	if err != nil {
		return err
	}
	return requireAffected(res)
}

// CountUsers returns the number of users.
func (s *Store) CountUsers(ctx context.Context) (int, error) {
	var n int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM users`).Scan(&n)
	return n, err
}

func wrapConflict(err error) error {
	if strings.Contains(err.Error(), "UNIQUE") {
		return fmt.Errorf("%w: %v", ErrConflict, err)
	}
	return err
}

func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func nonNil(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}
