package store

import (
	"context"
	"time"

	"callcenter-middleware/internal/domain"
)

// Audit appends to the audit log; failures are not fatal to the caller.
func (s *Store) Audit(ctx context.Context, e domain.AuditEntry) error {
	owner, err := ownerFor(ctx, e.TenantID)
	if err != nil {
		return err
	}
	if e.At.IsZero() {
		e.At = time.Now().UTC()
	}
	_, err = s.db.ExecContext(ctx, `INSERT INTO audit_log (tenant_id, user_id, username, action, entity, entity_id, detail, at)
		VALUES (?,?,?,?,?,?,?,?)`, owner, e.UserID, e.Username, e.Action, e.Entity, e.EntityID, e.Detail, ms(e.At))
	return err
}

// AuditFilter narrows audit queries.
type AuditFilter struct {
	Entity   string
	EntityID string
	Action   string
	UserID   int64
	Limit    int
	Offset   int
}

// ListAudit returns the newest audit entries matching f.
func (s *Store) ListAudit(ctx context.Context, f AuditFilter) ([]domain.AuditEntry, error) {
	cond, args, err := scoped(ctx, "tenant_id")
	if err != nil {
		return nil, err
	}
	if f.Limit <= 0 || f.Limit > 500 {
		f.Limit = 100
	}
	for col, v := range map[string]string{"entity": f.Entity, "entity_id": f.EntityID, "action": f.Action} {
		if v != "" {
			cond, args = cond+" AND "+col+" = ?", append(args, v)
		}
	}
	if f.UserID > 0 {
		cond, args = cond+" AND user_id = ?", append(args, f.UserID)
	}
	rows, err := s.db.QueryContext(ctx, `SELECT id, tenant_id, user_id, username, action, entity, entity_id, detail, at
		FROM audit_log WHERE 1=1`+cond+` ORDER BY id DESC LIMIT ? OFFSET ?`, append(args, f.Limit, f.Offset)...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.AuditEntry{}
	for rows.Next() {
		var e domain.AuditEntry
		var at int64
		if err := rows.Scan(&e.ID, &e.TenantID, &e.UserID, &e.Username, &e.Action, &e.Entity, &e.EntityID,
			&e.Detail, &at); err != nil {
			return nil, err
		}
		e.At = fromMS(at)
		out = append(out, e)
	}
	return out, rows.Err()
}

// AddAgentEvent records an agent status transition.
func (s *Store) AddAgentEvent(ctx context.Context, e domain.AgentEvent) error {
	owner, err := ownerFor(ctx, e.TenantID)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, `INSERT INTO agent_events (tenant_id, user_id, status, reason, at) VALUES (?,?,?,?,?)`,
		owner, e.UserID, e.Status, e.Reason, ms(e.At))
	return err
}
