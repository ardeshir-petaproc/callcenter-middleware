package server

import (
	"encoding/csv"
	"fmt"
	"net/http"
	"strings"
	"time"

	"callcenter-middleware/internal/domain"
)

// reportFilter defaults to "today" (server local time) when no range is given.
func reportFilter(r *http.Request) domain.CallFilter {
	f := callFilter(r)
	if f.From.IsZero() {
		now := time.Now()
		f.From = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	}
	if f.To.IsZero() {
		f.To = time.Now().Add(time.Minute)
	}
	f.Search, f.Outcome, f.Direction = "", "", ""
	return f
}

func (s *Server) reportSummary(w http.ResponseWriter, r *http.Request) error {
	sum, err := s.d.Store.ReportSummary(r.Context(), reportFilter(r))
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, sum)
	return nil
}

func (s *Server) reportQueues(w http.ResponseWriter, r *http.Request) error {
	out, err := s.d.Store.ReportByGroup(r.Context(), reportFilter(r), "queue_id")
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, out)
	return nil
}

func (s *Server) reportAgents(w http.ResponseWriter, r *http.Request) error {
	out, err := s.d.Store.ReportByGroup(r.Context(), reportFilter(r), "agent_id")
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, out)
	return nil
}

func (s *Server) reportAgentTime(w http.ResponseWriter, r *http.Request) error {
	f := reportFilter(r)
	out, err := s.d.Store.ReportAgentTime(r.Context(), f.From, f.To)
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, out)
	return nil
}

func (s *Server) reportTimeseries(w http.ResponseWriter, r *http.Request) error {
	by := r.URL.Query().Get("by")
	if by != "hour" {
		by = "day"
	}
	out, err := s.d.Store.ReportBuckets(r.Context(), reportFilter(r), by, qInt(r, "tz_offset", 0)*60)
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, out)
	return nil
}

func (s *Server) reportBreakdown(w http.ResponseWriter, r *http.Request) error {
	out, err := s.d.Store.ReportCounts(r.Context(), reportFilter(r), r.URL.Query().Get("by"))
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, out)
	return nil
}

func fmtTime(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}

// exportCSV streams call records matching the filter as CSV.
func (s *Server) exportCSV(w http.ResponseWriter, r *http.Request) error {
	f := callFilter(r)
	f.Limit = 500
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=calls-%s.csv", time.Now().Format("20060102-1504")))
	cw := csv.NewWriter(w)
	_ = cw.Write([]string{"id", "direction", "caller_number", "caller_name", "dnis", "queue_id", "agent_id",
		"outcome", "ivr_path", "started_at", "answered_at", "ended_at", "wait_sec", "talk_sec", "hold_sec",
		"sla_met", "disposition", "tags", "notes", "sentiment", "summary"})
	for page := 0; page < 200; page++ {
		f.Offset = page * f.Limit
		calls, _, err := s.d.Store.ListCalls(r.Context(), f)
		if err != nil {
			return err
		}
		for _, c := range calls {
			_ = cw.Write([]string{c.ID, c.Direction, c.CallerNumber, c.CallerName, c.DNIS, itoa(c.QueueID),
				itoa(c.AgentID), string(c.Outcome), c.IVRPath, c.StartedAt.Format(time.RFC3339),
				fmtTime(c.AnsweredAt), fmtTime(c.EndedAt), fmt.Sprint(c.WaitSec), fmt.Sprint(c.TalkSec),
				fmt.Sprint(c.HoldSec), fmt.Sprint(c.SLAMet), c.Disposition, strings.Join(c.Tags, ";"),
				c.Notes, c.Sentiment, c.Summary})
		}
		if len(calls) < f.Limit {
			break
		}
	}
	cw.Flush()
	return cw.Error()
}
