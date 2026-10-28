package server

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"callcenter-middleware/internal/auth"
	"callcenter-middleware/internal/domain"
	"callcenter-middleware/internal/store"
)

type ctxKey int

const principalKey ctxKey = 1

// principal is the authenticated caller of a request.
type principal struct {
	claims *auth.Claims
	user   *domain.User
	tenant *domain.Tenant
	perms  domain.PermissionSet
}

// authenticate validates the bearer token (header or ?token=), re-checks
// that the user and tenant are still active, resolves permissions and scopes
// every store call of the request to the user's tenant.
func (s *Server) authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tok := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		if tok == "" || tok == r.Header.Get("Authorization") {
			tok = r.URL.Query().Get("token")
		}
		claims, err := s.d.Issuer.Verify(tok)
		if err != nil || claims.TenantID <= 0 {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
			return
		}
		p, err := s.loadPrincipal(r.Context(), claims)
		if err != nil {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": err.Error()})
			return
		}
		ctx := context.WithValue(store.ForTenant(r.Context(), claims.TenantID), principalKey, p)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (s *Server) loadPrincipal(ctx context.Context, claims *auth.Claims) (*principal, error) {
	scoped := store.ForTenant(ctx, claims.TenantID)
	u, err := s.d.Store.GetUser(scoped, claims.UserID)
	if err != nil || !u.Enabled || u.Status == domain.UserDisabled {
		return nil, fmt.Errorf("account disabled")
	}
	t, err := s.d.Store.GetTenant(ctx, claims.TenantID)
	if err != nil || (t.Status != domain.TenantActive && !u.PlatformAdmin) {
		return nil, fmt.Errorf("organization suspended")
	}
	var custom *domain.CustomRole
	if u.CustomRoleID > 0 {
		custom, _ = s.d.Store.Roles.Get(scoped, u.CustomRoleID)
	}
	claims.Role, claims.Username, claims.Platform = u.Role, u.Username, u.PlatformAdmin
	return &principal{claims: claims, user: u, tenant: t, perms: domain.PermissionsFor(u.Role, custom)}, nil
}

func principalOf(r *http.Request) *principal {
	if p, ok := r.Context().Value(principalKey).(*principal); ok {
		return p
	}
	return &principal{claims: &auth.Claims{}, user: &domain.User{}, tenant: &domain.Tenant{}, perms: domain.PermissionSet{}}
}

func claimsOf(r *http.Request) *auth.Claims { return principalOf(r).claims }

func tenantOf(r *http.Request) int64 { return principalOf(r).claims.TenantID }

// can reports whether the caller holds permission p.
func can(r *http.Request, p domain.Permission) bool { return principalOf(r).perms.Has(p) }

// anyone marks routes open to every signed-in user of the tenant.
const anyone domain.Permission = "*"

// requirePerm restricts a route to callers holding permission p.
func requirePerm(p domain.Permission) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		if p == anyone {
			return next
		}
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !can(r, p) {
				writeErr(w, errForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// requirePlatform restricts a route to platform administrators.
func requirePlatform(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !principalOf(r).user.PlatformAdmin {
			writeErr(w, errForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// audit records an administrative change in the caller's tenant.
func (s *Server) audit(r *http.Request, action, entity string, id any, detail string) {
	c := claimsOf(r)
	_ = s.d.Store.Audit(r.Context(), domain.AuditEntry{
		UserID: c.UserID, Username: c.Username, Action: action, Entity: entity,
		EntityID: fmt.Sprint(id), Detail: detail,
	})
}

// configChanged reloads the call core and notifies the caller's tenant.
func (s *Server) configChanged(r *http.Request, entity string) {
	if s.d.Core != nil {
		_ = s.d.Core.Reload(r.Context())
	}
	s.d.Hub.Publish(domain.Event{TenantID: tenantOf(r), Type: domain.EvConfigChanged, Data: map[string]string{"entity": entity}})
}

// cors allows configured origins (the desktop app, Joow mini-app hosts).
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
