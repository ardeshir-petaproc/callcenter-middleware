package store

import (
	"context"
	"database/sql"
	"encoding/json"
)

// Count returns the number of documents of this kind in the scope.
func (c *Collection[T]) Count(ctx context.Context) (int, error) {
	sc, err := scopeOf(ctx)
	if err != nil {
		return 0, err
	}
	cond, args := sc.filter("tenant_id")
	var n int
	err = c.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM entities WHERE kind = ?`+cond,
		append([]any{c.kind}, args...)...).Scan(&n)
	return n, err
}

// SetID assigns id to item (used when updating from a request path).
func (c *Collection[T]) SetID(item *T, id int64) { idp, _ := c.keys(item); *idp = id }

// IDOf returns the id of item.
func (c *Collection[T]) IDOf(item *T) int64 { idp, _ := c.keys(item); return *idp }

// Kind returns the entity kind name.
func (c *Collection[T]) Kind() string { return c.kind }

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

// GetJSON returns the stored JSON of a document (for revision history).
func (c *Collection[T]) GetJSON(ctx context.Context, id int64) (string, error) {
	item, err := c.Get(ctx, id)
	if err != nil {
		return "", err
	}
	raw, err := json.Marshal(item)
	return string(raw), err
}
