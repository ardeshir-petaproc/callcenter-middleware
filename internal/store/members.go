package store

import (
	"context"

	"callcenter-middleware/internal/domain"
)

// ListMembers returns all queue memberships.
func (s *Store) ListMembers(ctx context.Context) ([]domain.QueueMember, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT queue_id, user_id, penalty FROM queue_members ORDER BY queue_id, penalty, user_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.QueueMember{}
	for rows.Next() {
		var m domain.QueueMember
		if err := rows.Scan(&m.QueueID, &m.UserID, &m.Penalty); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// SetQueueMembers replaces the member list of a queue atomically.
func (s *Store) SetQueueMembers(ctx context.Context, queueID int64, members []domain.QueueMember) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `DELETE FROM queue_members WHERE queue_id = ?`, queueID); err != nil {
		return err
	}
	for _, m := range members {
		if _, err := tx.ExecContext(ctx,
			`INSERT OR REPLACE INTO queue_members (queue_id, user_id, penalty) VALUES (?, ?, ?)`,
			queueID, m.UserID, m.Penalty); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// SetUserQueues replaces the queues a user belongs to atomically.
func (s *Store) SetUserQueues(ctx context.Context, userID int64, members []domain.QueueMember) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `DELETE FROM queue_members WHERE user_id = ?`, userID); err != nil {
		return err
	}
	for _, m := range members {
		if _, err := tx.ExecContext(ctx,
			`INSERT OR REPLACE INTO queue_members (queue_id, user_id, penalty) VALUES (?, ?, ?)`,
			m.QueueID, userID, m.Penalty); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// DeleteQueueMembers removes all memberships of a queue.
func (s *Store) DeleteQueueMembers(ctx context.Context, queueID int64) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM queue_members WHERE queue_id = ?`, queueID)
	return err
}
