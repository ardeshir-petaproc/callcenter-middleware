package store

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"callcenter-middleware/internal/domain"
)

const leadCols = `id, campaign_id, number, name, status, attempts, next_attempt, call_id, notes, updated_at`

func scanLead(sc interface{ Scan(...any) error }) (*domain.Lead, error) {
	var l domain.Lead
	var next, updated int64
	if err := sc.Scan(&l.ID, &l.CampaignID, &l.Number, &l.Name, &l.Status, &l.Attempts, &next,
		&l.CallID, &l.Notes, &updated); err != nil {
		return nil, err
	}
	l.NextAttempt, l.UpdatedAt = fromMS(next), fromMS(updated)
	return &l, nil
}

// AddLeads inserts pending leads into a campaign.
func (s *Store) AddLeads(ctx context.Context, campaignID int64, leads []domain.Lead) (int, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	now := ms(time.Now())
	for _, l := range leads {
		if _, err := tx.ExecContext(ctx, `INSERT INTO leads (campaign_id, number, name, status, next_attempt, notes, updated_at)
			VALUES (?,?,?,?,?,?,?)`, campaignID, l.Number, l.Name, domain.LeadPending, 0, l.Notes, now); err != nil {
			return 0, err
		}
	}
	return len(leads), tx.Commit()
}

// ListLeads returns leads of a campaign, optionally filtered by status.
func (s *Store) ListLeads(ctx context.Context, campaignID int64, status string, limit, offset int) ([]domain.Lead, error) {
	if limit <= 0 || limit > 1000 {
		limit = 200
	}
	q, args := `SELECT `+leadCols+` FROM leads WHERE campaign_id = ?`, []any{campaignID}
	if status != "" {
		q, args = q+` AND status = ?`, append(args, status)
	}
	rows, err := s.db.QueryContext(ctx, q+` ORDER BY id LIMIT ? OFFSET ?`, append(args, limit, offset)...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.Lead{}
	for rows.Next() {
		l, err := scanLead(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *l)
	}
	return out, rows.Err()
}

// NextLead returns the next due pending lead, fewest attempts first.
func (s *Store) NextLead(ctx context.Context, campaignID int64, now time.Time) (*domain.Lead, error) {
	l, err := scanLead(s.db.QueryRowContext(ctx, `SELECT `+leadCols+` FROM leads
		WHERE campaign_id = ? AND status = ? AND next_attempt <= ? ORDER BY attempts, id LIMIT 1`,
		campaignID, domain.LeadPending, ms(now)))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return l, err
}

// GetLead loads one lead.
func (s *Store) GetLead(ctx context.Context, id int64) (*domain.Lead, error) {
	l, err := scanLead(s.db.QueryRowContext(ctx, `SELECT `+leadCols+` FROM leads WHERE id = ?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return l, err
}

// UpdateLead saves mutable lead fields.
func (s *Store) UpdateLead(ctx context.Context, l *domain.Lead) error {
	l.UpdatedAt = time.Now().UTC()
	res, err := s.db.ExecContext(ctx, `UPDATE leads SET number = ?, name = ?, status = ?, attempts = ?,
		next_attempt = ?, call_id = ?, notes = ?, updated_at = ? WHERE id = ?`, l.Number, l.Name, l.Status,
		l.Attempts, ms(l.NextAttempt), l.CallID, l.Notes, ms(l.UpdatedAt), l.ID)
	if err != nil {
		return err
	}
	return requireAffected(res)
}

// LeadStats counts leads per status for a campaign.
func (s *Store) LeadStats(ctx context.Context, campaignID int64) (map[string]int, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT status, COUNT(*) FROM leads WHERE campaign_id = ? GROUP BY status`, campaignID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]int{"total": 0}
	for rows.Next() {
		var st string
		var n int
		if err := rows.Scan(&st, &n); err != nil {
			return nil, err
		}
		out[st] = n
		out["total"] += n
	}
	return out, rows.Err()
}

// DeleteLeads removes leads of a campaign (all when status is empty).
func (s *Store) DeleteLeads(ctx context.Context, campaignID int64, status string) error {
	q, args := `DELETE FROM leads WHERE campaign_id = ?`, []any{campaignID}
	if status != "" {
		q, args = q+` AND status = ?`, append(args, status)
	}
	_, err := s.db.ExecContext(ctx, q, args...)
	return err
}
