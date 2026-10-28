package store

import (
	"context"

	"callcenter-middleware/internal/domain"
)

// callScope restricts call-child rows (transcripts, events) to calls of
// the scope's tenant.
func callScope(ctx context.Context) (string, []any, error) {
	sc, err := scopeOf(ctx)
	if err != nil {
		return "", nil, err
	}
	if sc.system {
		return "", nil, nil
	}
	return " AND call_id IN (SELECT id FROM calls WHERE id = call_id AND tenant_id = ?)", []any{sc.tenant}, nil
}

// AddTranscript appends one utterance to a call transcript.
func (s *Store) AddTranscript(ctx context.Context, t domain.TranscriptEntry) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO transcripts (call_id, speaker, text, at) VALUES (?, ?, ?, ?)`,
		t.CallID, t.Speaker, t.Text, ms(t.At))
	return err
}

// Transcript returns the transcript of a call in order.
func (s *Store) Transcript(ctx context.Context, callID string) ([]domain.TranscriptEntry, error) {
	cond, args, err := callScope(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := s.db.QueryContext(ctx, `SELECT call_id, speaker, text, at FROM transcripts WHERE call_id = ?`+cond+
		` ORDER BY at, id`, append([]any{callID}, args...)...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.TranscriptEntry{}
	for rows.Next() {
		var t domain.TranscriptEntry
		var at int64
		if err := rows.Scan(&t.CallID, &t.Speaker, &t.Text, &at); err != nil {
			return nil, err
		}
		t.At = fromMS(at)
		out = append(out, t)
	}
	return out, rows.Err()
}

// AddCallEvent appends an entry to a call timeline.
func (s *Store) AddCallEvent(ctx context.Context, e domain.CallEvent) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO call_events (call_id, type, detail, at) VALUES (?, ?, ?, ?)`,
		e.CallID, e.Type, e.Detail, ms(e.At))
	return err
}

// CallEvents returns the timeline of a call in order.
func (s *Store) CallEvents(ctx context.Context, callID string) ([]domain.CallEvent, error) {
	cond, args, err := callScope(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := s.db.QueryContext(ctx, `SELECT call_id, type, detail, at FROM call_events WHERE call_id = ?`+cond+
		` ORDER BY at, id`, append([]any{callID}, args...)...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.CallEvent{}
	for rows.Next() {
		var e domain.CallEvent
		var at int64
		if err := rows.Scan(&e.CallID, &e.Type, &e.Detail, &at); err != nil {
			return nil, err
		}
		e.At = fromMS(at)
		out = append(out, e)
	}
	return out, rows.Err()
}

// GetSetting returns a tenant setting or def when absent.
func (s *Store) GetSetting(ctx context.Context, key, def string) string {
	tenant, _ := TenantOf(ctx)
	var v string
	if err := s.db.QueryRowContext(ctx, `SELECT value FROM tenant_settings WHERE tenant_id = ? AND key = ?`,
		tenant, key).Scan(&v); err != nil {
		return def
	}
	return v
}

// SetSetting stores a tenant setting value.
func (s *Store) SetSetting(ctx context.Context, key, value string) error {
	sc, err := scopeOf(ctx)
	if err != nil || sc.system {
		return ErrNoTenant
	}
	_, err = s.db.ExecContext(ctx, `INSERT OR REPLACE INTO tenant_settings (tenant_id, key, value) VALUES (?, ?, ?)`,
		sc.tenant, key, value)
	return err
}

// AllSettings returns every setting of the scope's tenant.
func (s *Store) AllSettings(ctx context.Context) (map[string]string, error) {
	sc, err := scopeOf(ctx)
	if err != nil || sc.system {
		return nil, ErrNoTenant
	}
	rows, err := s.db.QueryContext(ctx, `SELECT key, value FROM tenant_settings WHERE tenant_id = ?`, sc.tenant)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]string{}
	for rows.Next() {
		var k, v string
		if err := rows.Scan(&k, &v); err != nil {
			return nil, err
		}
		out[k] = v
	}
	return out, rows.Err()
}
