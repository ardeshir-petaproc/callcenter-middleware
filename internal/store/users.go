package store

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"callcenter-middleware/internal/domain"
)

// ErrConflict is returned when a unique constraint would be violated.
var ErrConflict = errors.New("conflict")

const userCols = `id, tenant_id, username, display_name, email, role, extension, endpoint, skills, enabled,
	password_hash, joow_account_id, custom_role_id, platform_admin, status, profile, created_at, updated_at`

func scanUser(sc interface{ Scan(...any) error }) (*domain.User, error) {
	var u domain.User
	var skills, profile string
	var enabled, platform int
	var created, updated int64
	if err := sc.Scan(&u.ID, &u.TenantID, &u.Username, &u.DisplayName, &u.Email, &u.Role, &u.Extension,
		&u.Endpoint, &skills, &enabled, &u.PasswordHash, &u.JoowAccountID, &u.CustomRoleID, &platform,
		&u.Status, &profile, &created, &updated); err != nil {
		return nil, err
	}
	_ = json.Unmarshal([]byte(skills), &u.Skills)
	_ = json.Unmarshal([]byte(profile), &u.Profile)
	normalizeUser(&u)
	u.Enabled, u.PlatformAdmin = enabled == 1, platform == 1
	u.CreatedAt, u.UpdatedAt = fromMS(created), fromMS(updated)
	return &u, nil
}

func normalizeUser(u *domain.User) {
	if u.Skills == nil {
		u.Skills = []string{}
	}
	if u.DepartmentIDs == nil {
		u.DepartmentIDs = []int64{}
	}
	if u.Languages == nil {
		u.Languages = []string{}
	}
	if u.Status == "" {
		u.Status = domain.UserActive
	}
}

func (s *Store) queryUsers(ctx context.Context, where string, args ...any) ([]domain.User, error) {
	sc, err := scopeOf(ctx)
	if err != nil {
		return nil, err
	}
	cond, targs := sc.filter("tenant_id")
	rows, err := s.db.QueryContext(ctx, `SELECT `+userCols+` FROM users WHERE 1=1`+where+cond+` ORDER BY id`,
		append(args, targs...)...)
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

// ListUsers returns the users of the scope ordered by id.
func (s *Store) ListUsers(ctx context.Context) ([]domain.User, error) { return s.queryUsers(ctx, "") }

// UsersByJoowAccount returns every employee record linked to a Joow account
// (one per tenant the person works for).
func (s *Store) UsersByJoowAccount(ctx context.Context, accountID string) ([]domain.User, error) {
	if accountID == "" {
		return []domain.User{}, nil
	}
	return s.queryUsers(ctx, ` AND joow_account_id = ?`, accountID)
}

// GetUser loads one user by id.
func (s *Store) GetUser(ctx context.Context, id int64) (*domain.User, error) {
	return first(s.queryUsers(ctx, ` AND id = ?`, id))
}

// GetUserByUsername loads one user by username (case-insensitive).
func (s *Store) GetUserByUsername(ctx context.Context, username string) (*domain.User, error) {
	return first(s.queryUsers(ctx, ` AND lower(username) = lower(?)`, username))
}

func userJSON(u *domain.User) (skills, profile string) {
	normalizeUser(u)
	sk, _ := json.Marshal(u.Skills)
	pr, _ := json.Marshal(u.Profile)
	return string(sk), string(pr)
}

// CreateUser inserts a user; PasswordHash must already be set.
func (s *Store) CreateUser(ctx context.Context, u *domain.User) error {
	sc, err := scopeOf(ctx)
	if err != nil {
		return err
	}
	if u.TenantID, err = sc.owner(u.TenantID); err != nil {
		return err
	}
	now := time.Now()
	skills, profile := userJSON(u)
	res, err := s.db.ExecContext(ctx, `INSERT INTO users (tenant_id, username, display_name, email, role, extension,
		endpoint, skills, enabled, password_hash, joow_account_id, custom_role_id, platform_admin, status, profile,
		created_at, updated_at) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		u.TenantID, u.Username, u.DisplayName, u.Email, u.Role, u.Extension, u.Endpoint, skills, boolInt(u.Enabled),
		u.PasswordHash, u.JoowAccountID, u.CustomRoleID, boolInt(u.PlatformAdmin), u.Status, profile, ms(now), ms(now))
	if err != nil {
		return wrapConflict(err)
	}
	u.ID, _ = res.LastInsertId()
	u.CreatedAt, u.UpdatedAt = now, now
	return nil
}

// UpdateUser saves profile fields; an empty PasswordHash keeps the old one.
// Tenant and platform-admin flag are never changed here.
func (s *Store) UpdateUser(ctx context.Context, u *domain.User) error {
	sc, err := scopeOf(ctx)
	if err != nil {
		return err
	}
	cond, args := sc.filter("tenant_id")
	skills, profile := userJSON(u)
	res, err := s.db.ExecContext(ctx, `UPDATE users SET username = ?, display_name = ?, email = ?, role = ?,
		extension = ?, endpoint = ?, skills = ?, enabled = ?, joow_account_id = ?, custom_role_id = ?, status = ?,
		profile = ?, password_hash = CASE WHEN ? = '' THEN password_hash ELSE ? END, updated_at = ? WHERE id = ?`+cond,
		append([]any{u.Username, u.DisplayName, u.Email, u.Role, u.Extension, u.Endpoint, skills, boolInt(u.Enabled),
			u.JoowAccountID, u.CustomRoleID, u.Status, profile, u.PasswordHash, u.PasswordHash, ms(time.Now()), u.ID},
			args...)...)
	if err != nil {
		return wrapConflict(err)
	}
	return requireAffected(res)
}

// DeleteUser removes a user and their queue memberships.
func (s *Store) DeleteUser(ctx context.Context, id int64) error {
	sc, err := scopeOf(ctx)
	if err != nil {
		return err
	}
	cond, args := sc.filter("tenant_id")
	res, err := s.db.ExecContext(ctx, `DELETE FROM users WHERE id = ?`+cond, append([]any{id}, args...)...)
	if err != nil {
		return err
	}
	return requireAffected(res)
}
