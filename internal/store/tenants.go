package store

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"callcenter-middleware/internal/domain"
)

const tenantCols = `id, slug, name, status, plan, settings, created_at`

func scanTenant(sc interface{ Scan(...any) error }) (*domain.Tenant, error) {
	var t domain.Tenant
	var created int64
	if err := sc.Scan(&t.ID, &t.Slug, &t.Name, &t.Status, &t.Plan, &t.Settings, &created); err != nil {
		return nil, err
	}
	t.CreatedAt = fromMS(created)
	return &t, nil
}

// ListTenants returns every tenant (platform administration only).
func (s *Store) ListTenants(ctx context.Context) ([]domain.Tenant, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT `+tenantCols+` FROM tenants ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.Tenant{}
	for rows.Next() {
		t, err := scanTenant(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *t)
	}
	return out, rows.Err()
}

// GetTenant loads a tenant by id.
func (s *Store) GetTenant(ctx context.Context, id int64) (*domain.Tenant, error) {
	t, err := scanTenant(s.db.QueryRowContext(ctx, `SELECT `+tenantCols+` FROM tenants WHERE id = ?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return t, err
}

// GetTenantBySlug loads a tenant by its slug.
func (s *Store) GetTenantBySlug(ctx context.Context, slug string) (*domain.Tenant, error) {
	t, err := scanTenant(s.db.QueryRowContext(ctx, `SELECT `+tenantCols+` FROM tenants WHERE slug = lower(?)`, slug))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return t, err
}

// CreateTenant inserts a tenant.
func (s *Store) CreateTenant(ctx context.Context, t *domain.Tenant) error {
	if t.Status == "" {
		t.Status = domain.TenantActive
	}
	if t.Settings == "" {
		t.Settings = "{}"
	}
	t.CreatedAt = time.Now().UTC()
	res, err := s.db.ExecContext(ctx, `INSERT INTO tenants (slug, name, status, plan, settings, created_at)
		VALUES (lower(?), ?, ?, ?, ?, ?)`, t.Slug, t.Name, t.Status, t.Plan, t.Settings, ms(t.CreatedAt))
	if err != nil {
		return wrapConflict(err)
	}
	t.ID, _ = res.LastInsertId()
	return nil
}

// UpdateTenant saves name, status and plan.
func (s *Store) UpdateTenant(ctx context.Context, t *domain.Tenant) error {
	res, err := s.db.ExecContext(ctx, `UPDATE tenants SET name = ?, status = ?, plan = ? WHERE id = ?`,
		t.Name, t.Status, t.Plan, t.ID)
	if err != nil {
		return err
	}
	return requireAffected(res)
}

// AddRevision stores a historical version of a configuration object.
func (s *Store) AddRevision(ctx context.Context, r domain.Revision) error {
	owner, err := ownerFor(ctx, r.TenantID)
	if err != nil {
		return err
	}
	if r.At.IsZero() {
		r.At = time.Now().UTC()
	}
	_, err = s.db.ExecContext(ctx, `INSERT INTO revisions (tenant_id, entity, entity_id, action, data, changed_by,
		changed_by_name, at) VALUES (?,?,?,?,?,?,?,?)`, owner, r.Entity, r.EntityID, r.Action, r.Data, r.ChangedBy,
		r.ChangedByName, ms(r.At))
	return err
}

// Revisions lists the history of one object, newest first.
func (s *Store) Revisions(ctx context.Context, entity string, entityID int64) ([]domain.Revision, error) {
	cond, args, err := scoped(ctx, "tenant_id")
	if err != nil {
		return nil, err
	}
	rows, err := s.db.QueryContext(ctx, `SELECT id, tenant_id, entity, entity_id, action, data, changed_by,
		changed_by_name, at FROM revisions WHERE entity = ? AND entity_id = ?`+cond+` ORDER BY id DESC LIMIT 200`,
		append([]any{entity, entityID}, args...)...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.Revision{}
	for rows.Next() {
		var r domain.Revision
		var at int64
		if err := rows.Scan(&r.ID, &r.TenantID, &r.Entity, &r.EntityID, &r.Action, &r.Data, &r.ChangedBy,
			&r.ChangedByName, &at); err != nil {
			return nil, err
		}
		r.At = fromMS(at)
		out = append(out, r)
	}
	return out, rows.Err()
}
