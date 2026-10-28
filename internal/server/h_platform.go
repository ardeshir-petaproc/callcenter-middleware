package server

import (
	"net/http"
	"regexp"
	"strings"

	"callcenter-middleware/internal/auth"
	"callcenter-middleware/internal/domain"
	"callcenter-middleware/internal/seed"
	"callcenter-middleware/internal/store"

	"github.com/go-chi/chi/v5"
)

var slugRe = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{1,40}$`)

func (s *Server) platformRoutes(r chi.Router) {
	r.Group(func(r chi.Router) {
		r.Use(requirePlatform)
		r.Get("/platform/tenants", handler(s.listTenants).ServeHTTP)
		r.Post("/platform/tenants", handler(s.createTenant).ServeHTTP)
		r.Put("/platform/tenants/{id}", handler(s.updateTenant).ServeHTTP)
	})
}

func (s *Server) listTenants(w http.ResponseWriter, r *http.Request) error {
	ts, err := s.d.Store.ListTenants(r.Context())
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, ts)
	return nil
}

// createTenant provisions an organization with its owner (a local admin or
// an existing Joow account) and default dispositions/pause reasons.
func (s *Server) createTenant(w http.ResponseWriter, r *http.Request) error {
	var req struct {
		Slug  string `json:"slug"`
		Name  string `json:"name"`
		Plan  string `json:"plan"`
		Owner struct {
			Username      string `json:"username"`
			Password      string `json:"password"`
			DisplayName   string `json:"display_name"`
			Email         string `json:"email"`
			JoowAccountID string `json:"joow_account_id"`
		} `json:"owner"`
	}
	if err := decode(r, &req); err != nil {
		return err
	}
	req.Slug = strings.ToLower(strings.TrimSpace(req.Slug))
	if !slugRe.MatchString(req.Slug) || strings.TrimSpace(req.Name) == "" {
		return bad("slug (a-z, 0-9, -) and name are required")
	}
	owner := &domain.User{Username: strings.TrimSpace(req.Owner.Username), DisplayName: req.Owner.DisplayName,
		Email: req.Owner.Email, Role: domain.RoleAdmin, Enabled: true, JoowAccountID: req.Owner.JoowAccountID}
	switch {
	case owner.JoowAccountID != "":
		owner.Username = firstNonEmpty(owner.Username, "joow-"+owner.JoowAccountID)
	case owner.Username == "" || len(req.Owner.Password) < 8:
		return bad("owner needs a joow_account_id or a username and a password of at least 8 characters")
	default:
		h, err := auth.HashPassword(req.Owner.Password)
		if err != nil {
			return err
		}
		owner.PasswordHash = h
	}
	t := &domain.Tenant{Slug: req.Slug, Name: strings.TrimSpace(req.Name), Plan: req.Plan}
	if err := s.d.Store.CreateTenant(r.Context(), t); err != nil {
		return err
	}
	ctx := store.ForTenant(r.Context(), t.ID)
	owner.DisplayName = firstNonEmpty(owner.DisplayName, owner.Username)
	if err := s.d.Store.CreateUser(ctx, owner); err != nil {
		return err
	}
	if err := seed.EnsureDefaults(ctx, s.d.Store); err != nil {
		return err
	}
	s.audit(r, "create_tenant", "tenant", t.ID, t.Slug)
	if s.d.Core != nil {
		_ = s.d.Core.Reload(r.Context())
	}
	writeJSON(w, http.StatusCreated, map[string]any{"tenant": t, "owner": owner})
	return nil
}

func (s *Server) updateTenant(w http.ResponseWriter, r *http.Request) error {
	id, err := pathID(r)
	if err != nil {
		return err
	}
	t, err := s.d.Store.GetTenant(r.Context(), id)
	if err != nil {
		return err
	}
	var req domain.Tenant
	if err := decode(r, &req); err != nil {
		return err
	}
	if req.Status != domain.TenantActive && req.Status != domain.TenantSuspended {
		return bad("status must be active or suspended")
	}
	if id == tenantOf(r) && req.Status == domain.TenantSuspended {
		return bad("you cannot suspend your own organization")
	}
	t.Name, t.Status, t.Plan = firstNonEmpty(req.Name, t.Name), req.Status, req.Plan
	if err := s.d.Store.UpdateTenant(r.Context(), t); err != nil {
		return err
	}
	s.audit(r, "update_tenant", "tenant", id, t.Status)
	writeJSON(w, http.StatusOK, t)
	return nil
}
