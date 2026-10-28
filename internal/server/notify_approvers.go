package server

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"callcenter-middleware/internal/domain"
	"callcenter-middleware/internal/joow"
	"callcenter-middleware/internal/store"
)

// notifyApprovers tells everyone who may approve routing rules (and has a
// Joow account) that drafts are waiting. It runs in the background.
func (s *Server) notifyApprovers(r *http.Request, drafts int, from string) {
	if s.d.Joow == nil || drafts == 0 {
		return
	}
	ctx := store.ForTenant(context.Background(), tenantOf(r))
	author, link := principalOf(r).user.DisplayName, strings.TrimRight(s.d.Config.PublicURL, "/")+"/routing"
	go func() {
		users, err := s.d.Store.ListUsers(ctx)
		if err != nil {
			return
		}
		roles := map[int64]*domain.CustomRole{}
		for i := range users {
			u := &users[i]
			if u.JoowAccountID == "" || !u.Enabled {
				continue
			}
			if u.CustomRoleID > 0 && roles[u.CustomRoleID] == nil {
				roles[u.CustomRoleID], _ = s.d.Store.Roles.Get(ctx, u.CustomRoleID)
			}
			if !domain.PermissionsFor(u.Role, roles[u.CustomRoleID]).Has(domain.PermRoutingApprove) {
				continue
			}
			_ = s.d.Joow.Notify(ctx, u.JoowAccountID, joow.Notification{Kind: "routing_rule_pending",
				Title: fmt.Sprintf("%d routing rule(s) awaiting approval", drafts),
				Body:  fmt.Sprintf("%s submitted %s.", author, from), URL: link})
		}
	}()
}
