package store

import (
	"context"
	"encoding/json"
	"time"

	"callcenter-middleware/internal/domain"
)

const emailCols = `id, tenant_id, connector_id, uid, message_id, in_reply_to, from_addr, from_name, subject, date,
	body, ticket_id, kb_source_id, created_at`

// SaveEmail records an imported message; it reports false when the
// connector already imported that UID.
func (s *Store) SaveEmail(ctx context.Context, m *domain.EmailMessage) (bool, error) {
	tenant, err := ownerFor(ctx, m.TenantID)
	if err != nil {
		return false, err
	}
	m.TenantID, m.CreatedAt = tenant, time.Now().UTC()
	refs, _ := json.Marshal(m.InReplyTo)
	res, err := s.db.ExecContext(ctx, `INSERT OR IGNORE INTO emails (tenant_id, connector_id, uid, message_id, in_reply_to,
		from_addr, from_name, subject, date, body, ticket_id, kb_source_id, created_at) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		tenant, m.ConnectorID, m.UID, m.MessageID, string(refs), m.FromAddr, m.FromName, m.Subject, m.Date.Unix(),
		m.Body, m.TicketID, m.KBSourceID, m.CreatedAt.Unix())
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return false, nil
	}
	m.ID, _ = res.LastInsertId()
	return true, nil
}

// LinkEmail records the ticket or knowledge source created from a message.
func (s *Store) LinkEmail(ctx context.Context, id, ticketID, kbSourceID int64) error {
	cond, args, err := scoped(ctx, "tenant_id")
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, `UPDATE emails SET ticket_id = ?, kb_source_id = ? WHERE id = ?`+cond,
		append([]any{ticketID, kbSourceID, id}, args...)...)
	return err
}

// ThreadTicket finds the ticket of an earlier message referenced by
// In-Reply-To / References.
func (s *Store) ThreadTicket(ctx context.Context, messageIDs []string) (int64, error) {
	cond, args, err := scoped(ctx, "tenant_id")
	if err != nil {
		return 0, err
	}
	for _, id := range messageIDs {
		var ticket int64
		err := s.db.QueryRowContext(ctx, `SELECT ticket_id FROM emails WHERE message_id = ? AND ticket_id > 0`+cond+
			` ORDER BY id DESC LIMIT 1`, append([]any{id}, args...)...).Scan(&ticket)
		if err == nil {
			return ticket, nil
		}
	}
	return 0, nil
}

// ListEmails returns a connector's most recent imported messages without
// bodies.
func (s *Store) ListEmails(ctx context.Context, connectorID int64, limit int) ([]domain.EmailMessage, error) {
	cond, args, err := scoped(ctx, "tenant_id")
	if err != nil {
		return nil, err
	}
	rows, err := s.db.QueryContext(ctx, `SELECT `+emailCols+` FROM emails WHERE connector_id = ?`+cond+
		` ORDER BY date DESC LIMIT ?`, append(append([]any{connectorID}, args...), limit)...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.EmailMessage{}
	for rows.Next() {
		var m domain.EmailMessage
		var refs string
		var date, created int64
		if err := rows.Scan(&m.ID, &m.TenantID, &m.ConnectorID, &m.UID, &m.MessageID, &refs, &m.FromAddr, &m.FromName,
			&m.Subject, &date, &m.Body, &m.TicketID, &m.KBSourceID, &created); err != nil {
			return nil, err
		}
		_ = json.Unmarshal([]byte(refs), &m.InReplyTo)
		m.Date, m.CreatedAt, m.Body = time.Unix(date, 0).UTC(), time.Unix(created, 0).UTC(), ""
		out = append(out, m)
	}
	return out, rows.Err()
}

// PurgeEmails deletes every message imported by a connector.
func (s *Store) PurgeEmails(ctx context.Context, connectorID int64) error {
	cond, args, err := scoped(ctx, "tenant_id")
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, `DELETE FROM emails WHERE connector_id = ?`+cond, append([]any{connectorID}, args...)...)
	return err
}

// TicketMailbox returns the connector and Message-ID of the latest email
// imported into a ticket, so replies go out through the same mailbox.
func (s *Store) TicketMailbox(ctx context.Context, ticketID int64) (int64, string, error) {
	cond, args, err := scoped(ctx, "tenant_id")
	if err != nil {
		return 0, "", err
	}
	var connector int64
	var msgID string
	err = s.db.QueryRowContext(ctx, `SELECT connector_id, message_id FROM emails WHERE ticket_id = ?`+cond+
		` ORDER BY id DESC LIMIT 1`, append([]any{ticketID}, args...)...).Scan(&connector, &msgID)
	return connector, msgID, err
}

// DeleteEmail forgets an imported message (used when its import failed, so
// the next sync retries it).
func (s *Store) DeleteEmail(ctx context.Context, id int64) error {
	cond, args, err := scoped(ctx, "tenant_id")
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, `DELETE FROM emails WHERE id = ?`+cond, append([]any{id}, args...)...)
	return err
}
