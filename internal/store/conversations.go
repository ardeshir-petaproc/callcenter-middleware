package store

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"callcenter-middleware/internal/domain"
)

func (s *Store) conversation(ctx context.Context, where string, arg ...any) (*domain.Conversation, error) {
	cond, args, err := scoped(ctx, "tenant_id")
	if err != nil {
		return nil, err
	}
	var c domain.Conversation
	var created int64
	err = s.db.QueryRowContext(ctx, `SELECT id, tenant_id, channel, external_id, ticket_id, created_at FROM conversations
		WHERE `+where+cond+` ORDER BY id DESC LIMIT 1`, append(arg, args...)...).Scan(&c.ID, &c.TenantID, &c.Channel,
		&c.ExternalID, &c.TicketID, &created)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	c.CreatedAt = fromMS(created)
	return &c, err
}

// FindConversation returns the conversation for a channel thread.
func (s *Store) FindConversation(ctx context.Context, channel, externalID string) (*domain.Conversation, error) {
	return s.conversation(ctx, `channel = ? AND external_id = ?`, channel, externalID)
}

// TicketConversation returns the chat thread a ticket came from.
func (s *Store) TicketConversation(ctx context.Context, ticketID int64) (*domain.Conversation, error) {
	return s.conversation(ctx, `ticket_id = ?`, ticketID)
}

// SaveConversation links a thread to a ticket (replacing an older link, e.g.
// after the previous ticket was closed).
func (s *Store) SaveConversation(ctx context.Context, c *domain.Conversation) error {
	tenant, err := ownerFor(ctx, c.TenantID)
	if err != nil {
		return err
	}
	c.TenantID, c.CreatedAt = tenant, time.Now().UTC()
	_, err = s.db.ExecContext(ctx, `INSERT INTO conversations (tenant_id, channel, external_id, ticket_id, created_at)
		VALUES (?,?,?,?,?) ON CONFLICT (tenant_id, channel, external_id) DO UPDATE SET ticket_id = excluded.ticket_id`,
		tenant, c.Channel, c.ExternalID, c.TicketID, ms(c.CreatedAt))
	return err
}
