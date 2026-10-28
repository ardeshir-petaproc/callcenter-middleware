package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

// ErrNotFound is returned when a record does not exist (or belongs to
// another tenant: the two are deliberately indistinguishable).
var ErrNotFound = errors.New("not found")

// Collection stores configuration documents of one kind as JSON rows.
// Configuration objects are small and read far more often than written, so
// a document layout keeps the schema flexible without per-entity tables.
// Every operation is scoped to the tenant carried by the context.
type Collection[T any] struct {
	db   *sql.DB
	kind string
	keys func(*T) (id *int64, tenant *int64)
}

func newCollection[T any](db *sql.DB, kind string, keys func(*T) (*int64, *int64)) *Collection[T] {
	return &Collection[T]{db: db, kind: kind, keys: keys}
}

func (c *Collection[T]) decode(id, tenant int64, data string) (T, error) {
	var item T
	if err := json.Unmarshal([]byte(data), &item); err != nil {
		return item, fmt.Errorf("decode %s %d: %w", c.kind, id, err)
	}
	idp, tp := c.keys(&item)
	*idp, *tp = id, tenant
	return item, nil
}

// List returns every document of the scope ordered by id.
func (c *Collection[T]) List(ctx context.Context) ([]T, error) {
	sc, err := scopeOf(ctx)
	if err != nil {
		return nil, err
	}
	cond, args := sc.filter("tenant_id")
	rows, err := c.db.QueryContext(ctx, `SELECT id, tenant_id, data FROM entities WHERE kind = ?`+cond+` ORDER BY id`,
		append([]any{c.kind}, args...)...)
	if err != nil {
		return nil, fmt.Errorf("list %s: %w", c.kind, err)
	}
	defer rows.Close()
	out := []T{}
	for rows.Next() {
		var id, tenant int64
		var data string
		if err := rows.Scan(&id, &tenant, &data); err != nil {
			return nil, err
		}
		item, err := c.decode(id, tenant, data)
		if err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

// Get loads one document by id.
func (c *Collection[T]) Get(ctx context.Context, id int64) (*T, error) {
	sc, err := scopeOf(ctx)
	if err != nil {
		return nil, err
	}
	cond, args := sc.filter("tenant_id")
	var tenant int64
	var data string
	err = c.db.QueryRowContext(ctx, `SELECT tenant_id, data FROM entities WHERE kind = ? AND id = ?`+cond,
		append([]any{c.kind, id}, args...)...).Scan(&tenant, &data)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get %s: %w", c.kind, err)
	}
	item, err := c.decode(id, tenant, data)
	if err != nil {
		return nil, err
	}
	return &item, nil
}

// Create inserts a document and assigns its id and tenant.
func (c *Collection[T]) Create(ctx context.Context, item *T) error {
	sc, err := scopeOf(ctx)
	if err != nil {
		return err
	}
	idp, tp := c.keys(item)
	owner, err := sc.owner(*tp)
	if err != nil {
		return err
	}
	*idp, *tp = 0, owner
	data, err := json.Marshal(item)
	if err != nil {
		return err
	}
	now := time.Now().UnixMilli()
	res, err := c.db.ExecContext(ctx,
		`INSERT INTO entities (kind, tenant_id, data, created_at, updated_at) VALUES (?, ?, ?, ?, ?)`,
		c.kind, owner, string(data), now, now)
	if err != nil {
		return fmt.Errorf("create %s: %w", c.kind, err)
	}
	*idp, err = res.LastInsertId()
	return err
}

// Update replaces a document by id. The tenant of a document never changes.
func (c *Collection[T]) Update(ctx context.Context, item *T) error {
	sc, err := scopeOf(ctx)
	if err != nil {
		return err
	}
	idp, tp := c.keys(item)
	cond, args := sc.filter("tenant_id")
	var owner int64
	err = c.db.QueryRowContext(ctx, `SELECT tenant_id FROM entities WHERE kind = ? AND id = ?`+cond,
		append([]any{c.kind, *idp}, args...)...).Scan(&owner)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	*tp = owner
	data, err := json.Marshal(item)
	if err != nil {
		return err
	}
	res, err := c.db.ExecContext(ctx, `UPDATE entities SET data = ?, updated_at = ? WHERE kind = ? AND id = ? AND tenant_id = ?`,
		string(data), time.Now().UnixMilli(), c.kind, *idp, owner)
	if err != nil {
		return fmt.Errorf("update %s: %w", c.kind, err)
	}
	return requireAffected(res)
}

// Delete removes a document by id.
func (c *Collection[T]) Delete(ctx context.Context, id int64) error {
	sc, err := scopeOf(ctx)
	if err != nil {
		return err
	}
	cond, args := sc.filter("tenant_id")
	res, err := c.db.ExecContext(ctx, `DELETE FROM entities WHERE kind = ? AND id = ?`+cond,
		append([]any{c.kind, id}, args...)...)
	if err != nil {
		return fmt.Errorf("delete %s: %w", c.kind, err)
	}
	return requireAffected(res)
}
