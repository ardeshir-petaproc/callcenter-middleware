package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"callcenter-middleware/internal/domain"
)

const ticketCols = `id, tenant_id, seq, subject, description, category, priority, status, source, department_id,
	assignee_id, contact_id, requester_name, requester_phone, requester_email, call_ids, tags, created_by,
	created_at, updated_at, first_response_at, resolved_at, closed_at, response_due_at, resolution_due_at`

func scanTicket(sc interface{ Scan(...any) error }) (*domain.Ticket, error) {
	var t domain.Ticket
	var seq, created, updated int64
	var calls, tags string
	var first, resolved, closed, respDue, resDue sql.NullInt64
	if err := sc.Scan(&t.ID, &t.TenantID, &seq, &t.Subject, &t.Description, &t.Category, &t.Priority, &t.Status,
		&t.Source, &t.DepartmentID, &t.AssigneeID, &t.ContactID, &t.RequesterName, &t.RequesterPhone,
		&t.RequesterEmail, &calls, &tags, &t.CreatedBy, &created, &updated, &first, &resolved, &closed,
		&respDue, &resDue); err != nil {
		return nil, err
	}
	t.Number = fmt.Sprintf("T-%d", seq)
	_ = json.Unmarshal([]byte(calls), &t.CallIDs)
	_ = json.Unmarshal([]byte(tags), &t.Tags)
	t.CallIDs, t.Tags = nonNil(t.CallIDs), nonNil(t.Tags)
	t.CreatedAt, t.UpdatedAt = fromMS(created), fromMS(updated)
	t.FirstResponse, t.ResolvedAt, t.ClosedAt = fromNullMS(first), fromNullMS(resolved), fromNullMS(closed)
	t.ResponseDue, t.ResolutionDue = fromNullMS(respDue), fromNullMS(resDue)
	return &t, nil
}

// CreateTicket inserts a ticket with the next per-tenant number.
func (s *Store) CreateTicket(ctx context.Context, t *domain.Ticket) error {
	owner, err := ownerFor(ctx, t.TenantID)
	if err != nil {
		return err
	}
	t.TenantID = owner
	now := time.Now().UTC()
	t.CreatedAt, t.UpdatedAt = now, now
	calls, _ := json.Marshal(nonNil(t.CallIDs))
	tags, _ := json.Marshal(nonNil(t.Tags))
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var seq int64
	if err := tx.QueryRowContext(ctx, `SELECT COALESCE(MAX(seq), 1000) + 1 FROM tickets WHERE tenant_id = ?`, owner).Scan(&seq); err != nil {
		return err
	}
	res, err := tx.ExecContext(ctx, `INSERT INTO tickets (tenant_id, seq, subject, description, category, priority,
		status, source, department_id, assignee_id, contact_id, requester_name, requester_phone, requester_email,
		call_ids, tags, created_by, created_at, updated_at, response_due_at, resolution_due_at)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`, owner, seq, t.Subject, t.Description, t.Category,
		t.Priority, t.Status, t.Source, t.DepartmentID, t.AssigneeID, t.ContactID, t.RequesterName, t.RequesterPhone,
		t.RequesterEmail, string(calls), string(tags), t.CreatedBy, ms(now), ms(now), msPtr(t.ResponseDue),
		msPtr(t.ResolutionDue))
	if err != nil {
		return err
	}
	t.ID, _ = res.LastInsertId()
	t.Number = fmt.Sprintf("T-%d", seq)
	return tx.Commit()
}

// UpdateTicket saves every mutable field.
func (s *Store) UpdateTicket(ctx context.Context, t *domain.Ticket) error {
	cond, args, err := scoped(ctx, "tenant_id")
	if err != nil {
		return err
	}
	t.UpdatedAt = time.Now().UTC()
	calls, _ := json.Marshal(nonNil(t.CallIDs))
	tags, _ := json.Marshal(nonNil(t.Tags))
	res, err := s.db.ExecContext(ctx, `UPDATE tickets SET subject = ?, description = ?, category = ?, priority = ?,
		status = ?, department_id = ?, assignee_id = ?, contact_id = ?, requester_name = ?, requester_phone = ?,
		requester_email = ?, call_ids = ?, tags = ?, updated_at = ?, first_response_at = ?, resolved_at = ?,
		closed_at = ?, response_due_at = ?, resolution_due_at = ? WHERE id = ?`+cond,
		append([]any{t.Subject, t.Description, t.Category, t.Priority, t.Status, t.DepartmentID, t.AssigneeID,
			t.ContactID, t.RequesterName, t.RequesterPhone, t.RequesterEmail, string(calls), string(tags), ms(t.UpdatedAt),
			msPtr(t.FirstResponse), msPtr(t.ResolvedAt), msPtr(t.ClosedAt), msPtr(t.ResponseDue), msPtr(t.ResolutionDue),
			t.ID}, args...)...)
	if err != nil {
		return err
	}
	return requireAffected(res)
}

// GetTicket loads one ticket.
func (s *Store) GetTicket(ctx context.Context, id int64) (*domain.Ticket, error) {
	cond, args, err := scoped(ctx, "tenant_id")
	if err != nil {
		return nil, err
	}
	t, err := scanTicket(s.db.QueryRowContext(ctx, `SELECT `+ticketCols+` FROM tickets WHERE id = ?`+cond,
		append([]any{id}, args...)...))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return t, err
}
