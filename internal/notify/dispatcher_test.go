package notify

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"callcenter-middleware/internal/domain"
)

type hooks []domain.Webhook

func (h hooks) ListWebhooks(context.Context) ([]domain.Webhook, error) { return h, nil }

func TestDispatcherSignsAndFilters(t *testing.T) {
	got := make(chan *http.Request, 4)
	bodies := make(chan []byte, 4)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		got <- r
		bodies <- b
	}))
	defer srv.Close()

	d := NewDispatcher(hooks{
		{URL: srv.URL, Events: []string{domain.EvCallEnded}, Secret: "s3cret", Enabled: true},
		{URL: srv.URL, Events: []string{"*"}, Enabled: false},
	})
	evs := make(chan domain.Event, 2)
	evs <- domain.Event{Type: domain.EvAgentUpdated, Data: 1}
	evs <- domain.Event{Type: domain.EvCallEnded, Data: map[string]string{"id": "c1"}}
	close(evs)
	d.Run(context.Background(), evs)

	select {
	case r := <-got:
		body := <-bodies
		if r.Header.Get("X-Callcenter-Signature") != "sha256="+Sign("s3cret", body) {
			t.Fatalf("bad signature %q", r.Header.Get("X-Callcenter-Signature"))
		}
	case <-time.After(2 * time.Second):
		t.Fatal("webhook not delivered")
	}
	select {
	case <-got:
		t.Fatal("unexpected extra delivery (filtered or disabled hook)")
	case <-time.After(200 * time.Millisecond):
	}
}
