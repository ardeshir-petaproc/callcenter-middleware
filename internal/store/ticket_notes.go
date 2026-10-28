package store

import (
	"context"
	"time"

	"callcenter-middleware/internal/domain"
)

// AddTicketComment stores a reply or internal note on a ticket.
func (s *Store) AddTicketComment(ctx context.Context, tenant int64, c *domain.TicketComment) error {
	if c.CreatedAt.IsZero() {
		c.CreatedAt = time.Now().UTC()
	}
	res, err := s.db.ExecContext(ctx, `INSERT INTO ticket_comments (tenant_id, ticket_id, author_id, author_name, body,
		internal, created_at) VALUES (?,?,?,?,?,?,?)`, tenant, c.TicketID, c.AuthorID, c.AuthorName, c.Body,
		boolInt(c.Internal), ms(c.CreatedAt))
	if err != nil {
		return err
	}
	c.ID, _ = res.LastInsertId()
	return nil
}

// TicketComments lists a ticket's comments in order.
func (s *Store) TicketComments(ctx context.Context, ticketID int64) ([]domain.TicketComment, error) {
	cond, args, err := scoped(ctx, "tenant_id")
	if err != nil {
		return nil, err
	}
	rows, err := s.db.QueryContext(ctx, `SELECT id, ticket_id, author_id, author_name, body, internal, created_at
		FROM ticket_comments WHERE ticket_id = ?`+cond+` ORDER BY id`, append([]any{ticketID}, args...)...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.TicketComment{}
	for rows.Next() {
		var c domain.TicketComment
		var internal int
		var at int64
		if err := rows.Scan(&c.ID, &c.TicketID, &c.AuthorID, &c.AuthorName, &c.Body, &internal, &at); err != nil {
			return nil, err
		}
		c.Internal, c.CreatedAt = internal == 1, fromMS(at)
		out = append(out, c)
	}
	return out, rows.Err()
}

// AddTicketEvent appends to a ticket's change history.
func (s *Store) AddTicketEvent(ctx context.Context, tenant int64, e domain.TicketEvent) error {
	if e.At.IsZero() {
		e.At = time.Now().UTC()
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO ticket_events (tenant_id, ticket_id, type, from_value, to_value,
		actor_id, actor_name, at) VALUES (?,?,?,?,?,?,?,?)`, tenant, e.TicketID, e.Type, e.From, e.To, e.ActorID,
		e.ActorName, ms(e.At))
	return err
}

// TicketEvents lists a ticket's change history in order.
func (s *Store) TicketEvents(ctx context.Context, ticketID int64) ([]domain.TicketEvent, error) {
	cond, args, err := scoped(ctx, "tenant_id")
	if err != nil {
		return nil, err
	}
	rows, err := s.db.QueryContext(ctx, `SELECT id, ticket_id, type, from_value, to_value, actor_id, actor_name, at
		FROM ticket_events WHERE ticket_id = ?`+cond+` ORDER BY id`, append([]any{ticketID}, args...)...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.TicketEvent{}
	for rows.Next() {
		var e domain.TicketEvent
		var at int64
		if err := rows.Scan(&e.ID, &e.TicketID, &e.Type, &e.From, &e.To, &e.ActorID, &e.ActorName, &at); err != nil {
			return nil, err
		}
		e.At = fromMS(at)
		out = append(out, e)
	}
	return out, rows.Err()
}
