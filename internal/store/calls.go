package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"

	"callcenter-middleware/internal/domain"
)

const callCols = `id, direction, caller_number, caller_name, dnis, state, outcome, flow_id, queue_id, agent_id,
	ai_profile_id, contact_id, ivr_path, started_at, queued_at, answered_at, ended_at, wait_sec, talk_sec,
	hold_sec, sla_met, disposition, notes, tags, recording, summary, sentiment, hangup_by`

// SaveCall inserts or fully replaces a call record.
func (s *Store) SaveCall(ctx context.Context, c *domain.Call) error {
	tags, _ := json.Marshal(nonNil(c.Tags))
	_, err := s.db.ExecContext(ctx, `INSERT OR REPLACE INTO calls (`+callCols+`)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		c.ID, c.Direction, c.CallerNumber, c.CallerName, c.DNIS, c.State, c.Outcome, c.FlowID, c.QueueID,
		c.AgentID, c.AIProfileID, c.ContactID, c.IVRPath, ms(c.StartedAt), msPtr(c.QueuedAt),
		msPtr(c.AnsweredAt), msPtr(c.EndedAt), c.WaitSec, c.TalkSec, c.HoldSec, boolInt(c.SLAMet),
		c.Disposition, c.Notes, string(tags), c.Recording, c.Summary, c.Sentiment, c.HangupBy)
	return err
}

// UpdateCallWrapup stores agent-entered disposition, notes and tags.
func (s *Store) UpdateCallWrapup(ctx context.Context, id, disposition, notes string, tags []string) error {
	t, _ := json.Marshal(nonNil(tags))
	res, err := s.db.ExecContext(ctx, `UPDATE calls SET disposition = ?, notes = ?, tags = ? WHERE id = ?`,
		disposition, notes, string(t), id)
	if err != nil {
		return err
	}
	return requireAffected(res)
}

// UpdateCallAnalysis stores an AI-generated summary and sentiment.
func (s *Store) UpdateCallAnalysis(ctx context.Context, id, summary, sentiment string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE calls SET summary = ?, sentiment = ? WHERE id = ?`, summary, sentiment, id)
	return err
}

func scanCall(sc interface{ Scan(...any) error }) (*domain.Call, error) {
	var c domain.Call
	var started int64
	var queued, answered, ended sql.NullInt64
	var sla int
	var tags string
	err := sc.Scan(&c.ID, &c.Direction, &c.CallerNumber, &c.CallerName, &c.DNIS, &c.State, &c.Outcome,
		&c.FlowID, &c.QueueID, &c.AgentID, &c.AIProfileID, &c.ContactID, &c.IVRPath, &started, &queued,
		&answered, &ended, &c.WaitSec, &c.TalkSec, &c.HoldSec, &sla, &c.Disposition, &c.Notes, &tags,
		&c.Recording, &c.Summary, &c.Sentiment, &c.HangupBy)
	if err != nil {
		return nil, err
	}
	c.StartedAt = fromMS(started)
	c.QueuedAt, c.AnsweredAt, c.EndedAt = fromNullMS(queued), fromNullMS(answered), fromNullMS(ended)
	c.SLAMet = sla == 1
	_ = json.Unmarshal([]byte(tags), &c.Tags)
	if c.Tags == nil {
		c.Tags = []string{}
	}
	return &c, nil
}

// GetCall loads one call by id.
func (s *Store) GetCall(ctx context.Context, id string) (*domain.Call, error) {
	c, err := scanCall(s.db.QueryRowContext(ctx, `SELECT `+callCols+` FROM calls WHERE id = ?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return c, err
}

// ListCalls returns calls matching f, newest first, plus the total count.
func (s *Store) ListCalls(ctx context.Context, f domain.CallFilter) ([]domain.Call, int, error) {
	where, args := callWhere(f)
	var total int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM calls`+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	limit := f.Limit
	if limit <= 0 || limit > 500 {
		limit = 50
	}
	q := `SELECT ` + callCols + ` FROM calls` + where + ` ORDER BY started_at DESC LIMIT ? OFFSET ?`
	rows, err := s.db.QueryContext(ctx, q, append(args, limit, f.Offset)...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	out := []domain.Call{}
	for rows.Next() {
		c, err := scanCall(rows)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, *c)
	}
	return out, total, rows.Err()
}

func callWhere(f domain.CallFilter) (string, []any) {
	var conds []string
	var args []any
	if !f.From.IsZero() {
		conds, args = append(conds, "started_at >= ?"), append(args, ms(f.From))
	}
	if !f.To.IsZero() {
		conds, args = append(conds, "started_at < ?"), append(args, ms(f.To))
	}
	if f.QueueID > 0 {
		conds, args = append(conds, "queue_id = ?"), append(args, f.QueueID)
	}
	if f.AgentID > 0 {
		conds, args = append(conds, "agent_id = ?"), append(args, f.AgentID)
	}
	if f.Outcome != "" {
		conds, args = append(conds, "outcome = ?"), append(args, f.Outcome)
	}
	if f.Direction != "" {
		conds, args = append(conds, "direction = ?"), append(args, f.Direction)
	}
	if f.Search != "" {
		like := "%" + f.Search + "%"
		conds = append(conds, "(caller_number LIKE ? OR caller_name LIKE ? OR notes LIKE ? OR id = ?)")
		args = append(args, like, like, like, f.Search)
	}
	if len(conds) == 0 {
		return "", nil
	}
	return " WHERE " + strings.Join(conds, " AND "), args
}

// RecentCallsByNumber returns the last calls from a caller number.
func (s *Store) RecentCallsByNumber(ctx context.Context, number string, limit int) ([]domain.Call, error) {
	calls, _, err := s.ListCalls(ctx, domain.CallFilter{Search: number, Limit: limit})
	return calls, err
}
