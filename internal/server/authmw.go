package server

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"callcenter-middleware/internal/auth"
	"callcenter-middleware/internal/domain"
)

type ctxKey int

const claimsKey ctxKey = 1

// authenticate validates the bearer token (header or ?token=) and stores claims.
func (s *Server) authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tok := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		if tok == "" || tok == r.Header.Get("Authorization") {
			tok = r.URL.Query().Get("token")
		}
		claims, err := s.d.Issuer.Verify(tok)
		if err != nil {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), claimsKey, claims)))
	})
}

func claimsOf(r *http.Request) *auth.Claims {
	c, _ := r.Context().Value(claimsKey).(*auth.Claims)
	if c == nil {
		return &auth.Claims{}
	}
	return c
}

// require restricts a route to users with at least the given role.
func require(min domain.Role) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if claimsOf(r).Role.Rank() < min.Rank() {
				writeErr(w, errForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func isSup(r *http.Request) bool { return claimsOf(r).Role.Rank() >= domain.RoleSupervisor.Rank() }

// audit records an administrative change.
func (s *Server) audit(r *http.Request, action, entity string, id any, detail string) {
	c := claimsOf(r)
	_ = s.d.Store.Audit(r.Context(), domain.AuditEntry{
		UserID: c.UserID, Username: c.Username, Action: action, Entity: entity,
		EntityID: fmt.Sprint(id), Detail: detail,
	})
}

// configChanged reloads the call core and notifies listeners.
func (s *Server) configChanged(r *http.Request, entity string) {
	if s.d.Core != nil {
		_ = s.d.Core.Reload(r.Context())
	}
	s.d.Hub.Publish(domain.Event{Type: domain.EvConfigChanged, Data: map[string]string{"entity": entity}})
}

// cors allows configured origins (the desktop app and dev servers).
func (s *Server) cors(next http.Handler) http.Handler {
	allowed := map[string]bool{}
	for _, o := range s.d.Config.CORSOrigins {
		allowed[o] = true
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" && (allowed[origin] || allowed["*"]) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}
