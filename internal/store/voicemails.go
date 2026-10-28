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
	owner, err := ownerFor(ctx, v.TenantID)
	if err != nil {
		return err
	}
	v.TenantID = owner
	if v.CreatedAt.IsZero() {
		v.CreatedAt = time.Now().UTC()
	}
	res, err := s.db.ExecContext(ctx, `INSERT INTO voicemails
		(tenant_id, call_id, caller_number, caller_name, queue_id, recording, duration_sec, transcript, heard, created_at)
		VALUES (?,?,?,?,?,?,?,?,?,?)`, owner, v.CallID, v.CallerNumber, v.CallerName, v.QueueID, v.Recording,
		v.DurationSec, v.Transcript, boolInt(v.Heard), ms(v.CreatedAt))
	if err != nil {
		return err
	}
	v.ID, _ = res.LastInsertId()
	return nil
}

const vmCols = `id, tenant_id, call_id, caller_number, caller_name, queue_id, recording, duration_sec, transcript, heard, created_at`

func scanVoicemail(sc interface{ Scan(...any) error }) (*domain.Voicemail, error) {
	var v domain.Voicemail
	var heard int
	var created int64
	if err := sc.Scan(&v.ID, &v.TenantID, &v.CallID, &v.CallerNumber, &v.CallerName, &v.QueueID, &v.Recording,
		&v.DurationSec, &v.Transcript, &heard, &created); err != nil {
		return nil, err
	}
	v.Heard, v.CreatedAt = heard == 1, fromMS(created)
	return &v, nil
}

// ListVoicemails returns voicemails newest first.
func (s *Store) ListVoicemails(ctx context.Context, limit int) ([]domain.Voicemail, error) {
	cond, args, err := scoped(ctx, "tenant_id")
	if err != nil {
		return nil, err
	}
	if limit <= 0 {
		limit = 200
	}
	rows, err := s.db.QueryContext(ctx, `SELECT `+vmCols+` FROM voicemails WHERE 1=1`+cond+
		` ORDER BY created_at DESC LIMIT ?`, append(args, limit)...)
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
	cond, args, err := scoped(ctx, "tenant_id")
	if err != nil {
		return nil, err
	}
	v, err := scanVoicemail(s.db.QueryRowContext(ctx, `SELECT `+vmCols+` FROM voicemails WHERE id = ?`+cond,
		append([]any{id}, args...)...))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return v, err
}

// UpdateVoicemail saves heard flag and transcript.
func (s *Store) UpdateVoicemail(ctx context.Context, v *domain.Voicemail) error {
	cond, args, err := scoped(ctx, "tenant_id")
	if err != nil {
		return err
	}
	res, err := s.db.ExecContext(ctx, `UPDATE voicemails SET heard = ?, transcript = ? WHERE id = ?`+cond,
		append([]any{boolInt(v.Heard), v.Transcript, v.ID}, args...)...)
	if err != nil {
		return err
	}
	return requireAffected(res)
}

// DeleteVoicemail removes a voicemail.
func (s *Store) DeleteVoicemail(ctx context.Context, id int64) error {
	cond, args, err := scoped(ctx, "tenant_id")
	if err != nil {
		return err
	}
	res, err := s.db.ExecContext(ctx, `DELETE FROM voicemails WHERE id = ?`+cond, append([]any{id}, args...)...)
	if err != nil {
		return err
	}
	return requireAffected(res)
}
