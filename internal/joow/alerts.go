package joow

import (
	"context"
	"fmt"
	"strings"

	"callcenter-middleware/internal/domain"
	"callcenter-middleware/internal/store"
)

// UserLookup loads an employee in a tenant-scoped context.
type UserLookup func(ctx context.Context, id int64) (*domain.User, error)

// RunAlerts forwards "your phone is ringing" screen-pops to the agent's
// Joow account, so employees working in the Joow app see incoming calls.
func RunAlerts(ctx context.Context, events <-chan domain.Event, p Provider, users UserLookup, panelURL string) {
	for {
		select {
		case <-ctx.Done():
			return
		case ev, ok := <-events:
			if !ok {
				return
			}
			if ev.Type != domain.EvScreenPop {
				continue
			}
			if n, account := incomingCall(store.ForTenant(ctx, ev.TenantID), ev, users, panelURL); account != "" {
				_ = p.Notify(ctx, account, n)
			}
		}
	}
}

func incomingCall(ctx context.Context, ev domain.Event, users UserLookup, panelURL string) (Notification, string) {
	data, _ := ev.Data.(map[string]any)
	uid, _ := data["user_id"].(int64)
	u, err := users(ctx, uid)
	if err != nil || u.JoowAccountID == "" {
		return Notification{}, ""
	}
	caller, callID := "", ""
	if v, ok := data["call"].(interface{ CallerLabel() (string, string) }); ok {
		caller, callID = v.CallerLabel()
	}
	n := Notification{Kind: "incoming_call", Title: "Incoming call", Body: strings.TrimSpace("From " + caller),
		URL: strings.TrimRight(panelURL, "/") + "/console", Data: map[string]any{"call_id": callID}}
	if caller == "" {
		n.Body = fmt.Sprintf("A call is ringing on extension %s", u.Extension)
	}
	return n, u.JoowAccountID
}
