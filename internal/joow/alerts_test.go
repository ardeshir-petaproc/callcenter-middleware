package joow

import (
	"context"
	"errors"
	"sync"
	"testing"

	"callcenter-middleware/internal/domain"
)

type recorder struct {
	Local
	mu  sync.Mutex
	got map[string]Notification
}

func (r *recorder) Notify(_ context.Context, account string, n Notification) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.got[account] = n
	return nil
}

type view struct{ name, number, id string }

func (v view) CallerLabel() (string, string) { return v.name + " (" + v.number + ")", v.id }

func TestRunAlerts_IncomingCallReachesTheAgentsJoowAccount(t *testing.T) {
	users := func(_ context.Context, id int64) (*domain.User, error) {
		switch id {
		case 1:
			return &domain.User{ID: 1, JoowAccountID: "j-alice", Extension: "1001"}, nil
		case 2:
			return &domain.User{ID: 2}, nil // no Joow account
		}
		return nil, errors.New("not found")
	}
	rec := &recorder{got: map[string]Notification{}}
	events := make(chan domain.Event, 3)
	events <- domain.Event{TenantID: 1, Type: domain.EvScreenPop, Data: map[string]any{"user_id": int64(2), "call": view{}}}
	events <- domain.Event{TenantID: 1, Type: "call.updated", Data: map[string]any{"user_id": int64(1)}}
	events <- domain.Event{TenantID: 1, Type: domain.EvScreenPop,
		Data: map[string]any{"user_id": int64(1), "call": view{"Sara", "+1555", "c-1"}}}
	close(events)
	RunAlerts(context.Background(), events, rec, users, "https://cc.example/")
	if len(rec.got) != 1 {
		t.Fatalf("expected one alert, got %+v", rec.got)
	}
	n := rec.got["j-alice"]
	if n.Kind != "incoming_call" || n.Body != "From Sara (+1555)" || n.URL != "https://cc.example/console" || n.Data["call_id"] != "c-1" {
		t.Fatalf("alert: %+v", n)
	}
}
