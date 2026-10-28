package server

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"callcenter-middleware/internal/domain"
	"callcenter-middleware/internal/store"
)

// secretKeys are removed from exported configuration documents.
var secretKeys = []string{"password", "token_hash", "secret", "storage_key"}

// exportOrganization streams a ZIP with the organization's configuration,
// people, calls, tickets and audit log (no recordings or files; those are
// exported per item with their own permissions). Audited.
func (s *Server) exportOrganization(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	s.audit(r, "organization.export", "tenant", tenantOf(r), "")
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q",
		fmt.Sprintf("organization-%d-%s.zip", tenantOf(r), time.Now().UTC().Format("20060102"))))
	zw := zip.NewWriter(w)
	defer zw.Close()
	put := func(name string, v any) error {
		f, err := zw.Create(name)
		if err != nil {
			return err
		}
		enc := json.NewEncoder(f)
		enc.SetIndent("", " ")
		return enc.Encode(v)
	}
	config := map[string][]map[string]any{}
	if err := s.d.Store.EachEntity(ctx, func(kind string, id int64, data json.RawMessage) error {
		var doc map[string]any
		if json.Unmarshal(data, &doc) != nil {
			return nil
		}
		for _, k := range secretKeys {
			delete(doc, k)
		}
		doc["id"] = id
		config[kind] = append(config[kind], doc)
		return nil
	}); err != nil {
		return err
	}
	for kind, docs := range config {
		if err := put("config/"+kind+".json", docs); err != nil {
			return err
		}
	}
	users, err := s.d.Store.ListUsers(ctx)
	if err != nil {
		return err
	}
	if err := put("users.json", users); err != nil {
		return err
	}
	if err := s.exportCalls(r, put); err != nil {
		return err
	}
	if err := s.exportTickets(r, put); err != nil {
		return err
	}
	audit, err := s.d.Store.ListAudit(ctx, store.AuditFilter{Limit: 100000})
	if err != nil {
		return err
	}
	return put("audit.json", audit)
}

func (s *Server) exportCalls(r *http.Request, put func(string, any) error) error {
	var all []domain.Call
	for off := 0; ; off += 1000 {
		page, _, err := s.d.Store.ListCalls(r.Context(), domain.CallFilter{Limit: 1000, Offset: off})
		if err != nil {
			return err
		}
		all = append(all, page...)
		if len(page) < 1000 {
			break
		}
	}
	return put("calls.json", all)
}

func (s *Server) exportTickets(r *http.Request, put func(string, any) error) error {
	type full struct {
		domain.Ticket
		Comments []domain.TicketComment `json:"comments"`
		History  []domain.TicketEvent   `json:"history"`
	}
	var out []full
	for off := 0; ; off += 500 {
		page, _, err := s.d.Store.ListTickets(r.Context(), domain.TicketFilter{Limit: 500, Offset: off})
		if err != nil {
			return err
		}
		for _, t := range page {
			f := full{Ticket: t}
			f.Comments, _ = s.d.Store.TicketComments(r.Context(), t.ID)
			f.History, _ = s.d.Store.TicketEvents(r.Context(), t.ID)
			out = append(out, f)
		}
		if len(page) < 500 {
			break
		}
	}
	return put("tickets.json", out)
}
