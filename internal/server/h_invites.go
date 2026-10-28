package server

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strings"
	"time"

	"callcenter-middleware/internal/domain"
	"callcenter-middleware/internal/joow"
	"callcenter-middleware/internal/store"

	"github.com/go-chi/chi/v5"
)

func (s *Server) inviteRoutes(r chi.Router) {
	users := requirePerm(domain.PermUsersManage)
	r.With(users).Get("/invitations", handler(s.listInvites).ServeHTTP)
	r.With(users).Post("/invitations", handler(s.createInvite).ServeHTTP)
	r.With(users).Post("/invitations/{id}/revoke", handler(s.revokeInvite).ServeHTTP)
	r.With(users).Get("/joow/accounts", handler(s.searchJoow).ServeHTTP)
	r.With(users).Post("/users/from-joow", handler(s.userFromJoow).ServeHTTP)
}

func hashToken(t string) string {
	sum := sha256.Sum256([]byte(t))
	return hex.EncodeToString(sum[:])
}

func (s *Server) listInvites(w http.ResponseWriter, r *http.Request) error {
	list, err := s.d.Store.Invitations.List(r.Context())
	if err != nil {
		return err
	}
	for i := range list {
		list[i].TokenHash = ""
	}
	writeJSON(w, http.StatusOK, list)
	return nil
}

// createInvite starts onboarding for a person without an employee record.
// When a Joow account is given, Joow notifies them in-app.
func (s *Server) createInvite(w http.ResponseWriter, r *http.Request) error {
	var inv domain.Invitation
	if err := decode(r, &inv); err != nil {
		return err
	}
	if strings.TrimSpace(inv.Email) == "" && inv.JoowAccountID == "" && inv.Phone == "" {
		return bad("email, phone or joow_account_id is required")
	}
	if inv.Role == "" {
		inv.Role = domain.RoleAgent
	}
	if !inv.Role.Valid() {
		return bad("invalid role")
	}
	if err := s.exists(r, "custom_role", inv.CustomRoleID); err != nil {
		return err
	}
	for _, d := range inv.DepartmentIDs {
		if err := s.exists(r, "department", d); err != nil {
			return err
		}
	}
	raw := make([]byte, 24)
	_, _ = rand.Read(raw)
	token := hex.EncodeToString(raw)
	inv.TokenHash, inv.Status, inv.InvitedBy = hashToken(token), domain.InvitePending, claimsOf(r).UserID
	inv.ExpiresAt = time.Now().Add(7 * 24 * time.Hour).UTC().Format(time.RFC3339)
	if err := s.d.Store.Invitations.Create(r.Context(), &inv); err != nil {
		return err
	}
	link := strings.TrimRight(s.d.Config.PublicURL, "/") + "/invite/" + token
	if inv.JoowAccountID != "" && s.d.Joow != nil {
		_ = s.d.Joow.Notify(r.Context(), inv.JoowAccountID, joow.Notification{Kind: "invitation",
			Title: "Invitation to " + principalOf(r).tenant.Name, Body: "You were invited to join the customer service team.",
			URL: link})
	}
	s.audit(r, "invite", "invitation", inv.ID, inv.Email+inv.JoowAccountID)
	inv.TokenHash = ""
	writeJSON(w, http.StatusCreated, map[string]any{"invitation": inv, "link": link, "token": token})
	return nil
}

func (s *Server) revokeInvite(w http.ResponseWriter, r *http.Request) error {
	id, err := pathID(r)
	if err != nil {
		return err
	}
	inv, err := s.d.Store.Invitations.Get(r.Context(), id)
	if err != nil {
		return err
	}
	inv.Status = domain.InviteRevoked
	if err := s.d.Store.Invitations.Update(r.Context(), inv); err != nil {
		return err
	}
	s.audit(r, "revoke", "invitation", id, "")
	inv.TokenHash = ""
	writeJSON(w, http.StatusOK, inv)
	return nil
}

// findInvite resolves a pending, unexpired invitation by its secret token.
func (s *Server) findInvite(r *http.Request, token string) (*domain.Invitation, *domain.Tenant, error) {
	all, err := s.d.Store.Invitations.List(store.System(r.Context()))
	if err != nil {
		return nil, nil, err
	}
	h := hashToken(token)
	for i := range all {
		inv := &all[i]
		if inv.TokenHash != h || inv.Status != domain.InvitePending {
			continue
		}
		if exp, err := time.Parse(time.RFC3339, inv.ExpiresAt); err == nil && time.Now().After(exp) {
			break
		}
		t, err := s.d.Store.GetTenant(r.Context(), inv.TenantID)
		if err != nil {
			return nil, nil, err
		}
		return inv, t, nil
	}
	return nil, nil, errStatus(http.StatusNotFound, "invitation not found or expired")
}

func (s *Server) peekInvite(w http.ResponseWriter, r *http.Request) error {
	inv, t, err := s.findInvite(r, chi.URLParam(r, "token"))
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, map[string]any{"organization": t.Name, "name": inv.Name, "email": inv.Email,
		"role": inv.Role, "joow_account_required": inv.JoowAccountID != "", "expires_at": inv.ExpiresAt})
	return nil
}
