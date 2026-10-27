package store

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"callcenter-middleware/internal/domain"
)

const cbCols = `id, call_id, number, name, queue_id, status, assigned_to, due_at, attempts, notes, created_at, updated_at`

func scanCallback(sc interface{ Scan(...any) error }) (*domain.Callback, error) {
	var c domain.Callback
	var due sql.NullInt64
	var created, updated int64
	if err := sc.Scan(&c.ID, &c.CallID, &c.Number, &c.Name, &c.QueueID, &c.Status, &c.AssignedTo,
		&due, &c.Attempts, &c.Notes, &created, &updated); err != nil {
		return nil, err
	}
	c.DueAt = fromNullMS(due)
	c.CreatedAt, c.UpdatedAt = fromMS(created), fromMS(updated)
	return &c, nil
}

// CreateCallback inserts a callback request.
func (s *Store) CreateCallback(ctx context.Context, c *domain.Callback) error {
	now := time.Now().UTC()
	if c.Status == "" {
		c.Status = domain.CallbackPending
	}
	res, err := s.db.ExecContext(ctx, `INSERT INTO callbacks
		(call_id, number, name, queue_id, status, assigned_to, due_at, attempts, notes, created_at, updated_at)
		VALUES (?,?,?,?,?,?,?,?,?,?,?)`, c.CallID, c.Number, c.Name, c.QueueID, c.Status, c.AssignedTo,
		msPtr(c.DueAt), c.Attempts, c.Notes, ms(now), ms(now))
	if err != nil {
		return err
	}
	c.ID, _ = res.LastInsertId()
	c.CreatedAt, c.UpdatedAt = now, now
	return nil
}

// UpdateCallback saves mutable callback fields.
func (s *Store) UpdateCallback(ctx context.Context, c *domain.Callback) error {
	c.UpdatedAt = time.Now().UTC()
	res, err := s.db.ExecContext(ctx, `UPDATE callbacks SET number = ?, name = ?, queue_id = ?, status = ?,
		assigned_to = ?, due_at = ?, attempts = ?, notes = ?, updated_at = ? WHERE id = ?`,
		c.Number, c.Name, c.QueueID, c.Status, c.AssignedTo, msPtr(c.DueAt), c.Attempts, c.Notes,
		ms(c.UpdatedAt), c.ID)
	if err != nil {
		return err
	}
	return requireAffected(res)
}

// GetCallback loads one callback.
func (s *Store) GetCallback(ctx context.Context, id int64) (*domain.Callback, error) {
	c, err := scanCallback(s.db.QueryRowContext(ctx, `SELECT `+cbCols+` FROM callbacks WHERE id = ?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return c, err
}

// ListCallbacks returns callbacks, optionally filtered by status.
func (s *Store) ListCallbacks(ctx context.Context, status string, limit int) ([]domain.Callback, error) {
	if limit <= 0 {
		limit = 200
	}
	q := `SELECT ` + cbCols + ` FROM callbacks`
	args := []any{}
	if status != "" {
		q += ` WHERE status = ?`
		args = append(args, status)
	}
	q += ` ORDER BY CASE status WHEN 'pending' THEN 0 WHEN 'in_progress' THEN 1 ELSE 2 END, created_at DESC LIMIT ?`
	rows, err := s.db.QueryContext(ctx, q, append(args, limit)...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.Callback{}
	for rows.Next() {
		c, err := scanCallback(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *c)
	}
	return out, rows.Err()
}

// DeleteCallback removes a callback.
func (s *Store) DeleteCallback(ctx context.Context, id int64) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM callbacks WHERE id = ?`, id)
	if err != nil {
		return err
	}
	return requireAffected(res)
}
