package server

import (
	"net/http"
	"strings"
	"time"

	"callcenter-middleware/internal/auth"
	"callcenter-middleware/internal/domain"
	"callcenter-middleware/internal/store"
)

// login authenticates a local (username/password) account. Organizations
// are selected by slug; the default organization is used when omitted.
func (s *Server) login(w http.ResponseWriter, r *http.Request) error {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
		Tenant   string `json:"tenant"`
	}
	if err := decode(r, &req); err != nil {
		return err
	}
	slug := strings.TrimSpace(req.Tenant)
	if slug == "" {
		slug = "default"
	}
	key := limiterKey(r, slug+"/"+req.Username)
	if s.limiter.blocked(key) {
		return errStatus(http.StatusTooManyRequests, "too many failed attempts, try again later")
	}
	var u *domain.User
	t, err := s.d.Store.GetTenantBySlug(r.Context(), slug)
	if err == nil {
		u, err = s.d.Store.GetUserByUsername(store.ForTenant(r.Context(), t.ID), strings.TrimSpace(req.Username))
	}
	if err != nil || !u.Enabled || u.Status == domain.UserDisabled || u.PasswordHash == "" ||
		!auth.CheckPassword(u.PasswordHash, req.Password) {
		s.limiter.fail(key)
		// Constant-ish failure path to avoid username probing.
		time.Sleep(200 * time.Millisecond)
		return errStatus(http.StatusUnauthorized, "invalid username or password")
	}
	s.limiter.reset(key)
	return s.issueSession(w, r, u, "login")
}

// issueSession signs a token for u and returns the session payload.
func (s *Server) issueSession(w http.ResponseWriter, r *http.Request, u *domain.User, how string) error {
	t, err := s.d.Store.GetTenant(r.Context(), u.TenantID)
	if err != nil {
		return err
	}
	if t.Status != domain.TenantActive && !u.PlatformAdmin {
		return errStatus(http.StatusForbidden, "organization suspended")
	}
	tok, exp, err := s.d.Issuer.Issue(u)
	if err != nil {
		return err
	}
	ctx := store.ForTenant(r.Context(), u.TenantID)
	_ = s.d.Store.Audit(ctx, domain.AuditEntry{UserID: u.ID, Username: u.Username, Action: how, Entity: "user",
		EntityID: itoa(u.ID)})
	var custom *domain.CustomRole
	if u.CustomRoleID > 0 {
		custom, _ = s.d.Store.Roles.Get(ctx, u.CustomRoleID)
	}
	writeJSON(w, http.StatusOK, map[string]any{"token": tok, "expires_at": exp.UTC(), "user": u, "tenant": t,
		"permissions": domain.PermissionsFor(u.Role, custom).List()})
	return nil
}

func (s *Server) me(w http.ResponseWriter, r *http.Request) error {
	u, err := s.d.Store.GetUser(r.Context(), claimsOf(r).UserID)
	if err != nil {
		return errStatus(http.StatusUnauthorized, "user no longer exists")
	}
	p := principalOf(r)
	resp := map[string]any{"user": u, "agent": nil, "tenant": p.tenant, "permissions": p.perms.List()}
	if st, ok := s.d.Core.Agent(u.ID); ok {
		resp["agent"] = st
	}
	writeJSON(w, http.StatusOK, resp)
	return nil
}

func (s *Server) changePassword(w http.ResponseWriter, r *http.Request) error {
	var req struct {
		Current string `json:"current_password"`
		New     string `json:"new_password"`
	}
	if err := decode(r, &req); err != nil {
		return err
	}
	u, err := s.d.Store.GetUser(r.Context(), claimsOf(r).UserID)
	if err != nil {
		return err
	}
	if !auth.CheckPassword(u.PasswordHash, req.Current) {
		return errStatus(http.StatusBadRequest, "current password is incorrect")
	}
	if len(req.New) < 8 {
		return bad("new password must be at least 8 characters")
	}
	if u.PasswordHash, err = auth.HashPassword(req.New); err != nil {
		return err
	}
	if err := s.d.Store.UpdateUser(r.Context(), u); err != nil {
		return err
	}
	s.audit(r, "change_password", "user", u.ID, "")
	ok(w)
	return nil
}
