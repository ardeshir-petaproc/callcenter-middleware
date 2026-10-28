package store

import (
	"context"

	"callcenter-middleware/internal/domain"
)

// ListMembers returns all queue memberships of the scope.
func (s *Store) ListMembers(ctx context.Context) ([]domain.QueueMember, error) {
	sc, err := scopeOf(ctx)
	if err != nil {
		return nil, err
	}
	cond, args := sc.filter("tenant_id")
	rows, err := s.db.QueryContext(ctx, `SELECT tenant_id, queue_id, user_id, penalty FROM queue_members
		WHERE 1=1`+cond+` ORDER BY queue_id, penalty, rowid`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.QueueMember{}
	for rows.Next() {
		var m domain.QueueMember
		if err := rows.Scan(&m.TenantID, &m.QueueID, &m.UserID, &m.Penalty); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// owns verifies that every referenced queue and user belongs to tenant, so a
// membership can never link objects across tenants.
func (s *Store) owns(ctx context.Context, tenant int64, queueIDs, userIDs []int64) error {
	scoped := ForTenant(ctx, tenant)
	for _, q := range queueIDs {
		if _, err := s.Queues.Get(scoped, q); err != nil {
			return err
		}
	}
	for _, u := range userIDs {
		if _, err := s.GetUser(scoped, u); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) setMembers(ctx context.Context, column string, id int64, members []domain.QueueMember) error {
	sc, err := scopeOf(ctx)
	if err != nil || sc.system {
		return ErrNoTenant
	}
	var qs, us []int64
	for i := range members {
		if column == "queue_id" {
			members[i].QueueID = id
		} else {
			members[i].UserID = id
		}
		qs, us = append(qs, members[i].QueueID), append(us, members[i].UserID)
	}
	if err := s.owns(ctx, sc.tenant, append(qs, idIf(column == "queue_id", id)...), append(us, idIf(column == "user_id", id)...)); err != nil {
		return err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `DELETE FROM queue_members WHERE `+column+` = ? AND tenant_id = ?`, id, sc.tenant); err != nil {
		return err
	}
	for _, m := range members {
		if _, err := tx.ExecContext(ctx, `INSERT OR REPLACE INTO queue_members (tenant_id, queue_id, user_id, penalty)
			VALUES (?, ?, ?, ?)`, sc.tenant, m.QueueID, m.UserID, m.Penalty); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func idIf(cond bool, id int64) []int64 {
	if cond {
		return []int64{id}
	}
	return nil
}

// SetQueueMembers replaces the member list of a queue atomically.
func (s *Store) SetQueueMembers(ctx context.Context, queueID int64, members []domain.QueueMember) error {
	return s.setMembers(ctx, "queue_id", queueID, members)
}

// SetUserQueues replaces the queues a user belongs to atomically.
func (s *Store) SetUserQueues(ctx context.Context, userID int64, members []domain.QueueMember) error {
	return s.setMembers(ctx, "user_id", userID, members)
}

// DeleteQueueMembers removes all memberships of a queue.
func (s *Store) DeleteQueueMembers(ctx context.Context, queueID int64) error {
	sc, err := scopeOf(ctx)
	if err != nil {
		return err
	}
	cond, args := sc.filter("tenant_id")
	_, err = s.db.ExecContext(ctx, `DELETE FROM queue_members WHERE queue_id = ?`+cond, append([]any{queueID}, args...)...)
	return err
}
