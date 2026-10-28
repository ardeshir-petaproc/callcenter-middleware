package store

import (
	"context"
	"strings"
	"time"

	"callcenter-middleware/internal/domain"
)

func ticketWhere(ctx context.Context, f domain.TicketFilter) (string, []any, error) {
	cond, args, err := scoped(ctx, "tenant_id")
	if err != nil {
		return "", nil, err
	}
	where := " WHERE 1=1" + cond
	add := func(c string, v ...any) { where += " AND " + c; args = append(args, v...) }
	if f.Status == "open" {
		add("status NOT IN (?, ?)", domain.TicketResolved, domain.TicketClosed)
	} else if f.Status != "" {
		add("status = ?", f.Status)
	}
	if f.Priority != "" {
		add("priority = ?", f.Priority)
	}
	if f.Category != "" {
		add("category = ?", f.Category)
	}
	if f.DepartmentID > 0 {
		add("department_id = ?", f.DepartmentID)
	}
	if f.AssigneeID > 0 {
		add("assignee_id = ?", f.AssigneeID)
	}
	if f.CallID != "" {
		add("call_ids LIKE ?", `%"`+f.CallID+`"%`)
	}
	if f.Search != "" {
		like := "%" + f.Search + "%"
		add("(subject LIKE ? OR description LIKE ? OR requester_name LIKE ? OR requester_phone LIKE ? OR 'T-' || seq = ?)",
			like, like, like, like, strings.ToUpper(f.Search))
	}
	if f.OverdueAt != nil {
		now := ms(*f.OverdueAt)
		add("status NOT IN (?, ?) AND ((resolution_due_at IS NOT NULL AND resolution_due_at < ?) OR "+
			"(first_response_at IS NULL AND response_due_at IS NOT NULL AND response_due_at < ?))",
			domain.TicketResolved, domain.TicketClosed, now, now)
	}
	if v := f.VisibleTo; v != nil {
		c := "(assignee_id = ? OR created_by = ?"
		vals := []any{v.UserID, v.UserID}
		for _, d := range v.DepartmentIDs {
			c += " OR department_id = ?"
			vals = append(vals, d)
		}
		add(c+")", vals...)
	}
	return where, args, nil
}

// ListTickets returns tickets matching f (most recently updated first).
func (s *Store) ListTickets(ctx context.Context, f domain.TicketFilter) ([]domain.Ticket, int, error) {
	where, args, err := ticketWhere(ctx, f)
	if err != nil {
		return nil, 0, err
	}
	var total int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM tickets`+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	if f.Limit <= 0 || f.Limit > 500 {
		f.Limit = 50
	}
	rows, err := s.db.QueryContext(ctx, `SELECT `+ticketCols+` FROM tickets`+where+
		` ORDER BY CASE priority WHEN 'urgent' THEN 0 WHEN 'high' THEN 1 WHEN 'normal' THEN 2 ELSE 3 END,
		updated_at DESC LIMIT ? OFFSET ?`, append(args, f.Limit, f.Offset)...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	out := []domain.Ticket{}
	for rows.Next() {
		t, err := scanTicket(rows)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, *t)
	}
	return out, total, rows.Err()
}

// TicketStats aggregates ticket workload and SLA performance.
type TicketStats struct {
	ByStatus          map[string]int `json:"by_status"`
	ByDepartment      map[int64]int  `json:"by_department_open"`
	Open              int            `json:"open"`
	Overdue           int            `json:"overdue"`
	CreatedInRange    int            `json:"created"`
	AvgFirstResponseM float64        `json:"avg_first_response_min"`
	AvgResolutionM    float64        `json:"avg_resolution_min"`
}

// TicketStatistics computes stats for tickets created in [from, to).
func (s *Store) TicketStatistics(ctx context.Context, from, to time.Time) (*TicketStats, error) {
	cond, args, err := scoped(ctx, "tenant_id")
	if err != nil {
		return nil, err
	}
	st := &TicketStats{ByStatus: map[string]int{}, ByDepartment: map[int64]int{}}
	rows, err := s.db.QueryContext(ctx, `SELECT status, department_id, COUNT(*) FROM tickets WHERE 1=1`+cond+
		` GROUP BY status, department_id`, args...)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var status string
		var dept int64
		var n int
		if err := rows.Scan(&status, &dept, &n); err != nil {
			rows.Close()
			return nil, err
		}
		st.ByStatus[status] += n
		if status != domain.TicketResolved && status != domain.TicketClosed {
			st.Open += n
			st.ByDepartment[dept] += n
		}
	}
	rows.Close()
	now := time.Now()
	if _, n, err := s.ListTickets(ctx, domain.TicketFilter{OverdueAt: &now, Limit: 1}); err == nil {
		st.Overdue = n
	}
	err = s.db.QueryRowContext(ctx, `SELECT COUNT(*),
		COALESCE(AVG(CASE WHEN first_response_at IS NOT NULL THEN (first_response_at - created_at) / 60000.0 END), 0),
		COALESCE(AVG(CASE WHEN resolved_at IS NOT NULL THEN (resolved_at - created_at) / 60000.0 END), 0)
		FROM tickets WHERE created_at >= ? AND created_at < ?`+cond, append([]any{ms(from), ms(to)}, args...)...).
		Scan(&st.CreatedInRange, &st.AvgFirstResponseM, &st.AvgResolutionM)
	return st, err
}
