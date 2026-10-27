package store

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"callcenter-middleware/internal/domain"
)

// CreateVoicemail stores a voicemail record.
func (s *Store) CreateVoicemail(ctx context.Context, v *domain.Voicemail) error {
	if v.CreatedAt.IsZero() {
		v.CreatedAt = time.Now().UTC()
	}
	res, err := s.db.ExecContext(ctx, `INSERT INTO voicemails
		(call_id, caller_number, caller_name, queue_id, recording, duration_sec, transcript, heard, created_at)
		VALUES (?,?,?,?,?,?,?,?,?)`, v.CallID, v.CallerNumber, v.CallerName, v.QueueID, v.Recording,
		v.DurationSec, v.Transcript, boolInt(v.Heard), ms(v.CreatedAt))
	if err != nil {
		return err
	}
	v.ID, _ = res.LastInsertId()
	return nil
}

const vmCols = `id, call_id, caller_number, caller_name, queue_id, recording, duration_sec, transcript, heard, created_at`

func scanVoicemail(sc interface{ Scan(...any) error }) (*domain.Voicemail, error) {
	var v domain.Voicemail
	var heard int
	var created int64
	if err := sc.Scan(&v.ID, &v.CallID, &v.CallerNumber, &v.CallerName, &v.QueueID, &v.Recording,
		&v.DurationSec, &v.Transcript, &heard, &created); err != nil {
		return nil, err
	}
	v.Heard, v.CreatedAt = heard == 1, fromMS(created)
	return &v, nil
}

// ListVoicemails returns voicemails newest first.
func (s *Store) ListVoicemails(ctx context.Context, limit int) ([]domain.Voicemail, error) {
	if limit <= 0 {
		limit = 200
	}
	rows, err := s.db.QueryContext(ctx, `SELECT `+vmCols+` FROM voicemails ORDER BY created_at DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.Voicemail{}
	for rows.Next() {
		v, err := scanVoicemail(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *v)
	}
	return out, rows.Err()
}

// GetVoicemail loads one voicemail.
func (s *Store) GetVoicemail(ctx context.Context, id int64) (*domain.Voicemail, error) {
	v, err := scanVoicemail(s.db.QueryRowContext(ctx, `SELECT `+vmCols+` FROM voicemails WHERE id = ?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return v, err
}

// UpdateVoicemail saves heard flag and transcript.
func (s *Store) UpdateVoicemail(ctx context.Context, v *domain.Voicemail) error {
	res, err := s.db.ExecContext(ctx, `UPDATE voicemails SET heard = ?, transcript = ? WHERE id = ?`,
		boolInt(v.Heard), v.Transcript, v.ID)
	if err != nil {
		return err
	}
	return requireAffected(res)
}

// DeleteVoicemail removes a voicemail.
func (s *Store) DeleteVoicemail(ctx context.Context, id int64) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM voicemails WHERE id = ?`, id)
	if err != nil {
		return err
	}
	return requireAffected(res)
}

// Audit appends to the audit log; failures are not fatal to the caller.
func (s *Store) Audit(ctx context.Context, e domain.AuditEntry) error {
	if e.At.IsZero() {
		e.At = time.Now().UTC()
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO audit_log (user_id, username, action, entity, entity_id, detail, at)
		VALUES (?,?,?,?,?,?,?)`, e.UserID, e.Username, e.Action, e.Entity, e.EntityID, e.Detail, ms(e.At))
	return err
}

// ListAudit returns the newest audit entries.
func (s *Store) ListAudit(ctx context.Context, limit, offset int) ([]domain.AuditEntry, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	rows, err := s.db.QueryContext(ctx, `SELECT id, user_id, username, action, entity, entity_id, detail, at
		FROM audit_log ORDER BY id DESC LIMIT ? OFFSET ?`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.AuditEntry{}
	for rows.Next() {
		var e domain.AuditEntry
		var at int64
		if err := rows.Scan(&e.ID, &e.UserID, &e.Username, &e.Action, &e.Entity, &e.EntityID, &e.Detail, &at); err != nil {
			return nil, err
		}
		e.At = fromMS(at)
		out = append(out, e)
	}
	return out, rows.Err()
}

// AddAgentEvent records an agent status transition.
func (s *Store) AddAgentEvent(ctx context.Context, e domain.AgentEvent) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO agent_events (user_id, status, reason, at) VALUES (?,?,?,?)`,
		e.UserID, e.Status, e.Reason, ms(e.At))
	return err
}
