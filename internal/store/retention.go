package store

import (
	"context"
	"time"
)

// StoredRecording is a recording file referenced by a call or voicemail.
type StoredRecording struct {
	Kind string // call or voicemail
	ID   string
	Name string
}

// ExpiredRecordings lists recordings of the tenant older than before.
func (s *Store) ExpiredRecordings(ctx context.Context, before time.Time, limit int) ([]StoredRecording, error) {
	cond, args, err := scoped(ctx, "tenant_id")
	if err != nil {
		return nil, err
	}
	q := `SELECT 'call', id, recording FROM calls WHERE recording != '' AND started_at < ?` + cond +
		` UNION ALL SELECT 'voicemail', CAST(id AS TEXT), recording FROM voicemails WHERE recording != '' AND created_at < ?` + cond +
		` LIMIT ?`
	all := append(append(append([]any{ms(before)}, args...), ms(before)), args...)
	rows, err := s.db.QueryContext(ctx, q, append(all, limit)...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []StoredRecording
	for rows.Next() {
		var r StoredRecording
		if err := rows.Scan(&r.Kind, &r.ID, &r.Name); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// ClearRecording removes the reference to a deleted recording.
func (s *Store) ClearRecording(ctx context.Context, r StoredRecording) error {
	cond, args, err := scoped(ctx, "tenant_id")
	if err != nil {
		return err
	}
	table := map[string]string{"call": "calls", "voicemail": "voicemails"}[r.Kind]
	if table == "" {
		return nil
	}
	_, err = s.db.ExecContext(ctx, `UPDATE `+table+` SET recording = '' WHERE id = ?`+cond, append([]any{r.ID}, args...)...)
	return err
}

// PurgeTranscripts deletes call transcripts and voicemail transcriptions
// older than before.
func (s *Store) PurgeTranscripts(ctx context.Context, before time.Time) (int64, error) {
	cond, args, err := scoped(ctx, "tenant_id")
	if err != nil {
		return 0, err
	}
	res, err := s.db.ExecContext(ctx, `DELETE FROM transcripts WHERE call_id IN (SELECT id FROM calls WHERE started_at < ?`+
		cond+`)`, append([]any{ms(before)}, args...)...)
	if err != nil {
		return 0, err
	}
	n, _ := res.RowsAffected()
	res, err = s.db.ExecContext(ctx, `UPDATE voicemails SET transcript = '' WHERE transcript != '' AND created_at < ?`+cond,
		append([]any{ms(before)}, args...)...)
	if err != nil {
		return n, err
	}
	m, _ := res.RowsAffected()
	return n + m, nil
}

// PurgeEmailBodies erases the bodies of imported emails older than before.
func (s *Store) PurgeEmailBodies(ctx context.Context, before time.Time) (int64, error) {
	cond, args, err := scoped(ctx, "tenant_id")
	if err != nil {
		return 0, err
	}
	res, err := s.db.ExecContext(ctx, `UPDATE emails SET body = '' WHERE body != '' AND date < ?`+cond,
		append([]any{before.Unix()}, args...)...)
	if err != nil {
		return 0, err
	}
	n, _ := res.RowsAffected()
	return n, nil
}
