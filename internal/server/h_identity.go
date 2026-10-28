package server

import (
	"errors"
	"net/http"

	"callcenter-middleware/internal/domain"
	"callcenter-middleware/internal/joow"
	"callcenter-middleware/internal/store"
)

// tenantChoice is one organization a Joow account belongs to.
type tenantChoice struct {
	ID     int64  `json:"id"`
	Slug   string `json:"slug"`
	Name   string `json:"name"`
	UserID int64  `json:"user_id"`
	Role   string `json:"role"`
}

func (s *Server) memberships(r *http.Request, accountID string) ([]domain.User, []tenantChoice, error) {
	users, err := s.d.Store.UsersByJoowAccount(store.System(r.Context()), accountID)
	if err != nil {
		return nil, nil, err
	}
	var active []domain.User
	choices := []tenantChoice{}
	for _, u := range users {
		if !u.Enabled || u.Status == domain.UserDisabled {
			continue
		}
		t, err := s.d.Store.GetTenant(r.Context(), u.TenantID)
		if err != nil || t.Status != domain.TenantActive {
			continue
		}
		active = append(active, u)
		choices = append(choices, tenantChoice{ID: t.ID, Slug: t.Slug, Name: t.Name, UserID: u.ID, Role: string(u.Role)})
	}
	return active, choices, nil
}

// loginJoow signs an employee in with their Joow session (single sign-on
// from the Joow mini-app or main app). When the account works for several
// organizations and none is chosen, the list of organizations is returned.
func (s *Server) loginJoow(w http.ResponseWriter, r *http.Request) error {
	var req struct {
		Token    string `json:"token"`
		TenantID int64  `json:"tenant_id"`
	}
	if err := decode(r, &req); err != nil {
		return err
	}
	if s.d.Joow == nil {
		return errStatus(http.StatusServiceUnavailable, "Joow integration not configured")
	}
	acct, err := s.d.Joow.VerifyToken(r.Context(), req.Token)
	if err != nil {
		if errors.Is(err, joow.ErrInvalidToken) || errors.Is(err, joow.ErrNotFound) {
			return errStatus(http.StatusUnauthorized, "invalid Joow session")
		}
		return errStatus(http.StatusBadGateway, "Joow: "+err.Error())
	}
	users, choices, err := s.memberships(r, acct.ID)
	if err != nil {
		return err
	}
	if len(users) == 0 {
		return errStatus(http.StatusForbidden, "this Joow account is not a member of any organization")
	}
	for i := range users {
		if users[i].TenantID == req.TenantID || (req.TenantID == 0 && len(users) == 1) {
			return s.issueSession(w, r, &users[i], "login_joow")
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"choose_tenant": true, "tenants": choices, "account": acct})
	return nil
}

// myTenants lists the organizations the signed-in Joow account belongs to.
func (s *Server) myTenants(w http.ResponseWriter, r *http.Request) error {
	p := principalOf(r)
	if p.user.JoowAccountID == "" {
		writeJSON(w, http.StatusOK, []tenantChoice{{ID: p.tenant.ID, Slug: p.tenant.Slug, Name: p.tenant.Name,
			UserID: p.user.ID, Role: string(p.user.Role)}})
		return nil
	}
	_, choices, err := s.memberships(r, p.user.JoowAccountID)
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, choices)
	return nil
}

// switchTenant re-issues the session for another organization of the same
// Joow account (or any organization, for platform administrators).
func (s *Server) switchTenant(w http.ResponseWriter, r *http.Request) error {
	var req struct {
		TenantID int64 `json:"tenant_id"`
	}
	if err := decode(r, &req); err != nil {
		return err
	}
	p := principalOf(r)
	if p.user.JoowAccountID != "" {
		users, _, err := s.memberships(r, p.user.JoowAccountID)
		if err != nil {
			return err
		}
		for i := range users {
			if users[i].TenantID == req.TenantID {
				return s.issueSession(w, r, &users[i], "switch_tenant")
			}
		}
	}
	return errForbidden
}

// devJoowToken mints a local Joow session token (development provider only).
func (s *Server) devJoowToken(w http.ResponseWriter, r *http.Request) error {
	local, ok := s.d.Joow.(*joow.Local)
	if !ok {
		return errStatus(http.StatusNotFound, "not found")
	}
	var req struct {
		AccountID string `json:"account_id"`
	}
	if err := decode(r, &req); err != nil {
		return err
	}
	if _, err := local.Get(r.Context(), req.AccountID); err != nil {
		return errStatus(http.StatusNotFound, "unknown account")
	}
	writeJSON(w, http.StatusOK, map[string]string{"token": local.MintToken(req.AccountID)})
	return nil
}

// listPermissions returns every permission plus the caller's own.
func (s *Server) listPermissions(w http.ResponseWriter, r *http.Request) error {
	writeJSON(w, http.StatusOK, map[string]any{"all": domain.AllPermissions, "mine": principalOf(r).perms.List()})
	return nil
}
