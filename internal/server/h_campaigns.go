package server

import (
	"encoding/csv"
	"net/http"
	"strings"

	"callcenter-middleware/internal/core"
	"callcenter-middleware/internal/domain"

	"github.com/go-chi/chi/v5"
)

func (s *Server) campaignRoutes(r chi.Router) {
	sup := requirePerm(domain.PermCampaignsManage)
	mountCRUD(r, "/campaigns", s, s.d.Store.Campaigns, crudOpts[domain.Campaign]{entity: "campaign",
		check: s.prepareCampaign, reload: true, writePerm: domain.PermCampaignsManage,
		onDelete: func(r *http.Request, id int64) { _ = s.d.Store.DeleteLeads(r.Context(), id, "") }})
	r.Get("/campaigns/{id}/leads", handler(s.listLeads).ServeHTTP)
	r.Get("/campaigns/{id}/stats", handler(s.campaignStats).ServeHTTP)
	r.With(sup).Post("/campaigns/{id}/leads", handler(s.addLeads).ServeHTTP)
	r.With(sup).Delete("/campaigns/{id}/leads", handler(s.deleteLeads).ServeHTTP)
	r.With(sup).Post("/campaigns/{id}/start", handler(s.setCampaignStatus(domain.CampaignRunning)).ServeHTTP)
	r.With(sup).Post("/campaigns/{id}/pause", handler(s.setCampaignStatus(domain.CampaignPaused)).ServeHTTP)
}

func (s *Server) prepareCampaign(r *http.Request, c *domain.Campaign) error {
	if c.Name = strings.TrimSpace(c.Name); c.Name == "" {
		return bad("name is required")
	}
	if _, err := s.d.Store.Queues.Get(r.Context(), c.QueueID); err != nil {
		return bad("queue_id must reference an existing queue (its members make the calls)")
	}
	switch c.Status {
	case "":
		c.Status = domain.CampaignDraft
	case domain.CampaignDraft, domain.CampaignRunning, domain.CampaignPaused, domain.CampaignCompleted:
	default:
		return bad("invalid status")
	}
	if c.MaxAttempts <= 0 {
		c.MaxAttempts = 3
	}
	if c.RetryMinutes <= 0 {
		c.RetryMinutes = 30
	}
	return nil
}

func (s *Server) listLeads(w http.ResponseWriter, r *http.Request) error {
	id, err := pathID(r)
	if err != nil {
		return err
	}
	leads, err := s.d.Store.ListLeads(r.Context(), id, r.URL.Query().Get("status"), qInt(r, "limit", 200), qInt(r, "offset", 0))
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, leads)
	return nil
}

func (s *Server) campaignStats(w http.ResponseWriter, r *http.Request) error {
	id, err := pathID(r)
	if err != nil {
		return err
	}
	stats, err := s.d.Store.LeadStats(r.Context(), id)
	if err != nil {
		return err
	}
	sum, err := s.d.Store.ReportSummary(r.Context(), domain.CallFilter{CampaignID: id})
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, map[string]any{"leads": stats, "calls": sum})
	return nil
}

// addLeads accepts {"leads":[{number,name,notes}]} or {"csv":"number,name\n..."}.
func (s *Server) addLeads(w http.ResponseWriter, r *http.Request) error {
	id, err := pathID(r)
	if err != nil {
		return err
	}
	if _, err := s.d.Store.Campaigns.Get(r.Context(), id); err != nil {
		return err
	}
	var req struct {
		Leads []domain.Lead `json:"leads"`
		CSV   string        `json:"csv"`
	}
	if err := decode(r, &req); err != nil {
		return err
	}
	if req.CSV != "" {
		rows, err := csv.NewReader(strings.NewReader(req.CSV)).ReadAll()
		if err != nil {
			return bad("invalid CSV: " + err.Error())
		}
		for _, row := range rows {
			l := domain.Lead{Number: strings.TrimSpace(row[0])}
			if len(row) > 1 {
				l.Name = strings.TrimSpace(row[1])
			}
			if len(row) > 2 {
				l.Notes = strings.TrimSpace(row[2])
			}
			req.Leads = append(req.Leads, l)
		}
	}
	valid := req.Leads[:0]
	for _, l := range req.Leads {
		if core.NormalizePhone(l.Number) != "" {
			l.Number = core.NormalizePhoneKeepPlus(l.Number)
			valid = append(valid, l)
		}
	}
	n, err := s.d.Store.AddLeads(r.Context(), id, valid)
	if err != nil {
		return err
	}
	s.audit(r, "add_leads", "campaign", id, itoa(int64(n)))
	writeJSON(w, http.StatusOK, map[string]int{"added": n, "skipped": len(req.Leads) - n})
	return nil
}

func (s *Server) deleteLeads(w http.ResponseWriter, r *http.Request) error {
	id, err := pathID(r)
	if err != nil {
		return err
	}
	if err := s.d.Store.DeleteLeads(r.Context(), id, r.URL.Query().Get("status")); err != nil {
		return err
	}
	s.audit(r, "delete_leads", "campaign", id, r.URL.Query().Get("status"))
	ok(w)
	return nil
}

func (s *Server) setCampaignStatus(status string) handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		id, err := pathID(r)
		if err != nil {
			return err
		}
		cp, err := s.d.Store.Campaigns.Get(r.Context(), id)
		if err != nil {
			return err
		}
		cp.Status = status
		if err := s.d.Store.Campaigns.Update(r.Context(), cp); err != nil {
			return err
		}
		s.audit(r, status, "campaign", id, "")
		s.configChanged(r, "campaign")
		writeJSON(w, http.StatusOK, cp)
		return nil
	}
}
