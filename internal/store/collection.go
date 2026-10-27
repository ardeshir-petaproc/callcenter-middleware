package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

// ErrNotFound is returned when a record does not exist.
var ErrNotFound = errors.New("not found")

// Collection stores configuration documents of one kind as JSON rows.
// Configuration objects are small and read far more often than written, so
// a document layout keeps the schema flexible without per-entity tables.
type Collection[T any] struct {
	db    *sql.DB
	kind  string
	idPtr func(*T) *int64
}

func newCollection[T any](db *sql.DB, kind string, idPtr func(*T) *int64) *Collection[T] {
	return &Collection[T]{db: db, kind: kind, idPtr: idPtr}
}

// List returns every document ordered by id.
func (c *Collection[T]) List(ctx context.Context) ([]T, error) {
	rows, err := c.db.QueryContext(ctx, `SELECT id, data FROM entities WHERE kind = ? ORDER BY id`, c.kind)
	if err != nil {
		return nil, fmt.Errorf("list %s: %w", c.kind, err)
	}
	defer rows.Close()
	out := []T{}
	for rows.Next() {
		var id int64
		var data string
		if err := rows.Scan(&id, &data); err != nil {
			return nil, err
		}
		var item T
		if err := json.Unmarshal([]byte(data), &item); err != nil {
			return nil, fmt.Errorf("decode %s %d: %w", c.kind, id, err)
		}
		*c.idPtr(&item) = id
		out = append(out, item)
	}
	return out, rows.Err()
}

// Get loads one document by id.
func (c *Collection[T]) Get(ctx context.Context, id int64) (*T, error) {
	var data string
	err := c.db.QueryRowContext(ctx, `SELECT data FROM entities WHERE kind = ? AND id = ?`, c.kind, id).Scan(&data)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get %s: %w", c.kind, err)
	}
	var item T
	if err := json.Unmarshal([]byte(data), &item); err != nil {
		return nil, err
	}
	*c.idPtr(&item) = id
	return &item, nil
}

// Create inserts a document and assigns its id.
func (c *Collection[T]) Create(ctx context.Context, item *T) error {
	*c.idPtr(item) = 0
	data, err := json.Marshal(item)
	if err != nil {
		return err
	}
	now := time.Now().UnixMilli()
	res, err := c.db.ExecContext(ctx,
		`INSERT INTO entities (kind, data, created_at, updated_at) VALUES (?, ?, ?, ?)`,
		c.kind, string(data), now, now)
	if err != nil {
		return fmt.Errorf("create %s: %w", c.kind, err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return err
	}
	*c.idPtr(item) = id
	return nil
}

// Update replaces a document by id.
func (c *Collection[T]) Update(ctx context.Context, item *T) error {
	data, err := json.Marshal(item)
	if err != nil {
		return err
	}
	res, err := c.db.ExecContext(ctx,
		`UPDATE entities SET data = ?, updated_at = ? WHERE kind = ? AND id = ?`,
		string(data), time.Now().UnixMilli(), c.kind, *c.idPtr(item))
	if err != nil {
		return fmt.Errorf("update %s: %w", c.kind, err)
	}
	return requireAffected(res)
}

// Delete removes a document by id.
func (c *Collection[T]) Delete(ctx context.Context, id int64) error {
	res, err := c.db.ExecContext(ctx, `DELETE FROM entities WHERE kind = ? AND id = ?`, c.kind, id)
	if err != nil {
		return fmt.Errorf("delete %s: %w", c.kind, err)
	}
	return requireAffected(res)
}

// Count returns the number of documents of this kind.
func (c *Collection[T]) Count(ctx context.Context) (int, error) {
	var n int
	err := c.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM entities WHERE kind = ?`, c.kind).Scan(&n)
	return n, err
}

func requireAffected(res sql.Result) error {
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// SetID assigns id to item (used when updating from a request path).
func (c *Collection[T]) SetID(item *T, id int64) { *c.idPtr(item) = id }

// IDOf returns the id of item.
func (c *Collection[T]) IDOf(item *T) int64 { return *c.idPtr(item) }
