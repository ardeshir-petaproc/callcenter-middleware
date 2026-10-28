package joow

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHTTPProviderContract(t *testing.T) {
	var notified map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/v1/me":
			if r.Header.Get("Authorization") != "Bearer good" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			_ = json.NewEncoder(w).Encode(Account{ID: "u1", Name: "Ana"})
		case r.Header.Get("X-Api-Key") != "key":
			w.WriteHeader(http.StatusForbidden)
		case r.URL.Path == "/v1/accounts" && r.URL.Query().Get("q") == "an":
			_ = json.NewEncoder(w).Encode(map[string]any{"accounts": []Account{{ID: "u1", Name: "Ana"}}})
		case r.URL.Path == "/v1/accounts/missing":
			w.WriteHeader(http.StatusNotFound)
		case r.URL.Path == "/v1/notifications" && r.Method == http.MethodPost:
			_ = json.NewDecoder(r.Body).Decode(&notified)
			w.WriteHeader(http.StatusAccepted)
		default:
			w.WriteHeader(http.StatusTeapot)
		}
	}))
	defer srv.Close()
	p, ctx := NewHTTP(srv.URL, "key"), context.Background()
	if a, err := p.VerifyToken(ctx, "good"); err != nil || a.ID != "u1" {
		t.Fatalf("verify: %v %+v", err, a)
	}
	if _, err := p.VerifyToken(ctx, "bad"); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("expected invalid token, got %v", err)
	}
	if list, err := p.Search(ctx, "an", 5); err != nil || len(list) != 1 {
		t.Fatalf("search: %v %+v", err, list)
	}
	if _, err := p.Get(ctx, "missing"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected not found, got %v", err)
	}
	if err := p.Notify(ctx, "u1", Notification{Kind: "incoming_call", Title: "Call"}); err != nil || notified["account_id"] != "u1" {
		t.Fatalf("notify: %v %+v", err, notified)
	}
}
