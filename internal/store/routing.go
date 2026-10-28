package store

import (
	"context"
	"encoding/json"
	"time"

	"callcenter-middleware/internal/domain"
)

// SaveDecision appends a routing decision to the log.
func (s *Store) SaveDecision(ctx context.Context, d *domain.RoutingDecision) error {
	tenant, err := ownerFor(ctx, d.TenantID)
	if err != nil {
		return err
	}
	d.TenantID = tenant
	if d.At.IsZero() {
		d.At = time.Now().UTC()
	}
	raw, _ := json.Marshal(d)
	res, err := s.db.ExecContext(ctx, `INSERT INTO routing_decisions (tenant_id, call_id, at, rule_id, data) VALUES (?,?,?,?,?)`,
		tenant, d.CallID, d.At.UnixMilli(), d.RuleID, string(raw))
	if err != nil {
		return err
	}
	d.ID, _ = res.LastInsertId()
	return nil
}

// Decisions lists logged decisions, newest first, optionally for one call
// or one rule.
func (s *Store) Decisions(ctx context.Context, callID string, ruleID int64, limit int) ([]domain.RoutingDecision, error) {
	cond, args, err := scoped(ctx, "tenant_id")
	if err != nil {
		return nil, err
	}
	if callID != "" {
		cond, args = cond+" AND call_id = ?", append(args, callID)
	}
	if ruleID > 0 {
		cond, args = cond+" AND rule_id = ?", append(args, ruleID)
	}
	rows, err := s.db.QueryContext(ctx, `SELECT id, data FROM routing_decisions WHERE 1=1`+cond+
		` ORDER BY at DESC, id DESC LIMIT ?`, append(args, limit)...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.RoutingDecision{}
	for rows.Next() {
		var id int64
		var raw string
		if err := rows.Scan(&id, &raw); err != nil {
			return nil, err
		}
		var d domain.RoutingDecision
		_ = json.Unmarshal([]byte(raw), &d)
		d.ID = id
		out = append(out, d)
	}
	return out, rows.Err()
}
