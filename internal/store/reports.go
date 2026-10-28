package store

import (
	"context"
	"database/sql"

	"callcenter-middleware/internal/domain"
)

// Summary aggregates KPIs for a set of calls.
type Summary struct {
	Total       int     `json:"total"`
	Inbound     int     `json:"inbound"`
	Outbound    int     `json:"outbound"`
	Queued      int     `json:"queued"`
	Answered    int     `json:"answered"`
	Abandoned   int     `json:"abandoned"`
	Voicemail   int     `json:"voicemail"`
	Callbacks   int     `json:"callbacks"`
	AIResolved  int     `json:"ai_resolved"`
	SLAMet      int     `json:"sla_met"`
	SLAPercent  float64 `json:"sla_percent"`
	AbandonRate float64 `json:"abandon_rate"`
	AvgWaitSec  float64 `json:"avg_wait_sec"`
	AvgTalkSec  float64 `json:"avg_talk_sec"`
	MaxWaitSec  int     `json:"max_wait_sec"`
	TotalTalk   int     `json:"total_talk_sec"`
	CSATCount   int     `json:"csat_count"`
	CSATAvg     float64 `json:"csat_avg"`
}

// GroupStat is a Summary for one queue or agent.
type GroupStat struct {
	ID int64 `json:"id"`
	Summary
}

// Bucket is a time-bucketed call count.
type Bucket struct {
	Key       string `json:"key"`
	Total     int    `json:"total"`
	Answered  int    `json:"answered"`
	Abandoned int    `json:"abandoned"`
}

// LabelCount is a generic label/count pair.
type LabelCount struct {
	Label string `json:"label"`
	Count int    `json:"count"`
}

const summarySelect = `SELECT COUNT(*),
	COALESCE(SUM(direction = 'inbound'), 0), COALESCE(SUM(direction = 'outbound'), 0),
	COALESCE(SUM(queued_at IS NOT NULL), 0), COALESCE(SUM(answered_at IS NOT NULL), 0),
	COALESCE(SUM(outcome = 'abandoned'), 0), COALESCE(SUM(outcome = 'voicemail'), 0),
	COALESCE(SUM(outcome = 'callback_requested'), 0), COALESCE(SUM(outcome = 'ai_resolved'), 0),
	COALESCE(SUM(sla_met), 0),
	COALESCE(AVG(CASE WHEN answered_at IS NOT NULL AND queued_at IS NOT NULL THEN wait_sec END), 0),
	COALESCE(AVG(CASE WHEN answered_at IS NOT NULL THEN talk_sec END), 0),
	COALESCE(MAX(wait_sec), 0), COALESCE(SUM(talk_sec), 0),
	COALESCE(SUM(csat > 0), 0), COALESCE(AVG(CASE WHEN csat > 0 THEN csat END), 0)`

func scanSummary(sc interface{ Scan(...any) error }, extra ...any) (Summary, error) {
	var s Summary
	dest := append(extra, &s.Total, &s.Inbound, &s.Outbound, &s.Queued, &s.Answered, &s.Abandoned,
		&s.Voicemail, &s.Callbacks, &s.AIResolved, &s.SLAMet, &s.AvgWaitSec, &s.AvgTalkSec, &s.MaxWaitSec, &s.TotalTalk, &s.CSATCount, &s.CSATAvg)
	if err := sc.Scan(dest...); err != nil {
		return s, err
	}
	if s.Queued > 0 {
		s.SLAPercent = 100 * float64(s.SLAMet) / float64(s.Queued)
		s.AbandonRate = 100 * float64(s.Abandoned) / float64(s.Queued)
	}
	return s, nil
}

// ReportSummary computes KPIs for calls matching f.
func (s *Store) ReportSummary(ctx context.Context, f domain.CallFilter) (Summary, error) {
	where, args, err := callWhere(ctx, f)
	if err != nil {
		return Summary{}, err
	}
	return scanSummary(s.db.QueryRowContext(ctx, summarySelect+` FROM calls`+where, args...))
}

// ReportByGroup computes KPIs per queue_id or agent_id.
func (s *Store) ReportByGroup(ctx context.Context, f domain.CallFilter, column string) ([]GroupStat, error) {
	if column != "queue_id" && column != "agent_id" {
		column = "queue_id"
	}
	where, args, err := callWhere(ctx, f)
	if err != nil {
		return nil, err
	}
	where += " AND " + column + " > 0"
	q := `SELECT ` + column + `, ` + summarySelect[len("SELECT "):] + ` FROM calls` + where + ` GROUP BY ` + column
	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []GroupStat{}
	for rows.Next() {
		var g GroupStat
		sum, err := scanSummary(rows, &g.ID)
		if err != nil {
			return nil, err
		}
		g.Summary = sum
		out = append(out, g)
	}
	return out, rows.Err()
}

// ReportBuckets groups call volume by hour-of-day ("hour") or date ("day").
// offsetSec shifts timestamps into the viewer's timezone.
func (s *Store) ReportBuckets(ctx context.Context, f domain.CallFilter, by string, offsetSec int) ([]Bucket, error) {
	format := "%Y-%m-%d"
	if by == "hour" {
		format = "%H"
	}
	where, args, err := callWhere(ctx, f)
	if err != nil {
		return nil, err
	}
	q := `SELECT strftime('` + format + `', started_at / 1000 + ?, 'unixepoch') AS k, COUNT(*),
		COALESCE(SUM(answered_at IS NOT NULL), 0), COALESCE(SUM(outcome = 'abandoned'), 0)
		FROM calls` + where + ` GROUP BY k ORDER BY k`
	rows, err := s.db.QueryContext(ctx, q, append([]any{offsetSec}, args...)...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Bucket{}
	for rows.Next() {
		var b Bucket
		if err := rows.Scan(&b.Key, &b.Total, &b.Answered, &b.Abandoned); err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

// ReportCounts counts calls by a categorical column (outcome or disposition).
func (s *Store) ReportCounts(ctx context.Context, f domain.CallFilter, column string) ([]LabelCount, error) {
	if column != "outcome" && column != "disposition" && column != "sentiment" {
		column = "outcome"
	}
	where, args, err := callWhere(ctx, f)
	if err != nil {
		return nil, err
	}
	rows, err := s.db.QueryContext(ctx, `SELECT `+column+`, COUNT(*) FROM calls`+where+
		` GROUP BY `+column+` ORDER BY COUNT(*) DESC`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []LabelCount{}
	for rows.Next() {
		var l LabelCount
		var label sql.NullString
		if err := rows.Scan(&label, &l.Count); err != nil {
			return nil, err
		}
		l.Label = label.String
		out = append(out, l)
	}
	return out, rows.Err()
}
