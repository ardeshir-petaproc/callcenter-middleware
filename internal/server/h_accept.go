package server

import (
	"net/http"
	"strings"

	"callcenter-middleware/internal/auth"
	"callcenter-middleware/internal/domain"
	"callcenter-middleware/internal/store"
)

// acceptInvite turns an invitation into an employee account, either linked
// to the person's Joow account (preferred) or with a local password.
func (s *Server) acceptInvite(w http.ResponseWriter, r *http.Request) error {
	var req struct {
		Token     string `json:"token"`
		JoowToken string `json:"joow_token"`
		Username  string `json:"username"`
		Password  string `json:"password"`
		Name      string `json:"display_name"`
	}
	if err := decode(r, &req); err != nil {
		return err
	}
	inv, _, err := s.findInvite(r, req.Token)
	if err != nil {
		return err
	}
	u := &domain.User{TenantID: inv.TenantID, Role: inv.Role, CustomRoleID: inv.CustomRoleID, Email: inv.Email,
		DisplayName: firstNonEmpty(req.Name, inv.Name), Enabled: true, Status: domain.UserActive, Skills: []string{}}
	u.DepartmentIDs, u.Phone = inv.DepartmentIDs, inv.Phone
	switch {
	case req.JoowToken != "" && s.d.Joow != nil:
		acct, err := s.d.Joow.VerifyToken(r.Context(), req.JoowToken)
		if err != nil {
			return errStatus(http.StatusUnauthorized, "invalid Joow session")
		}
		if inv.JoowAccountID != "" && inv.JoowAccountID != acct.ID {
			return errStatus(http.StatusForbidden, "this invitation belongs to another Joow account")
		}
		u.JoowAccountID, u.Username = acct.ID, "joow-"+acct.ID
		u.DisplayName = firstNonEmpty(u.DisplayName, acct.Name)
		u.Email = firstNonEmpty(u.Email, acct.Email)
	case inv.JoowAccountID != "":
		return bad("sign in with Joow to accept this invitation")
	default:
		if strings.TrimSpace(req.Username) == "" || len(req.Password) < 8 {
			return bad("username and a password of at least 8 characters are required")
		}
		u.Username = strings.TrimSpace(req.Username)
		if u.PasswordHash, err = auth.HashPassword(req.Password); err != nil {
			return err
		}
	}
	ctx := store.ForTenant(r.Context(), inv.TenantID)
	if err := s.d.Store.CreateUser(ctx, u); err != nil {
		return err
	}
	inv.Status, inv.UserID = domain.InviteAccepted, u.ID
	if err := s.d.Store.Invitations.Update(ctx, inv); err != nil {
		return err
	}
	_ = s.d.Store.Audit(ctx, domain.AuditEntry{UserID: u.ID, Username: u.Username, Action: "accept_invite",
		Entity: "invitation", EntityID: itoa(inv.ID)})
	if s.d.Core != nil {
		_ = s.d.Core.Reload(r.Context())
	}
	return s.issueSession(w, r, u, "login")
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

// searchJoow finds Joow accounts to add as employees.
func (s *Server) searchJoow(w http.ResponseWriter, r *http.Request) error {
	if s.d.Joow == nil {
		return errStatus(http.StatusServiceUnavailable, "Joow integration not configured")
	}
	accts, err := s.d.Joow.Search(r.Context(), r.URL.Query().Get("q"), qInt(r, "limit", 20))
	if err != nil {
		return errStatus(http.StatusBadGateway, "Joow: "+err.Error())
	}
	writeJSON(w, http.StatusOK, accts)
	return nil
}

// userFromJoow creates an employee record for an existing Joow account.
func (s *Server) userFromJoow(w http.ResponseWriter, r *http.Request) error {
	var req userReq
	req.Enabled = true
	if err := decode(r, &req); err != nil {
		return err
	}
	if s.d.Joow == nil || req.JoowAccountID == "" {
		return bad("joow_account_id is required")
	}
	acct, err := s.d.Joow.Get(r.Context(), req.JoowAccountID)
	if err != nil {
		return errStatus(http.StatusNotFound, "Joow account not found")
	}
	if req.Username == "" {
		req.Username = "joow-" + acct.ID
	}
	req.DisplayName = firstNonEmpty(req.DisplayName, acct.Name)
	req.Email = firstNonEmpty(req.Email, acct.Email)
	req.Phone = firstNonEmpty(req.Phone, acct.Phone)
	req.Password = ""
	return s.saveNewUser(w, r, &req)
}
