package store

import (
	"context"
	"errors"
)

// ErrNoTenant is returned when a query runs without a tenant scope. Data
// access fails closed: no scope means no data, never "all tenants".
var ErrNoTenant = errors.New("store: no tenant scope in context")

type scopeKey struct{}

type scope struct {
	tenant int64
	system bool
}

// ForTenant scopes every store call made with ctx to one tenant.
func ForTenant(ctx context.Context, tenantID int64) context.Context {
	return context.WithValue(ctx, scopeKey{}, scope{tenant: tenantID})
}

// System grants cross-tenant access. Only the call core, background jobs
// and platform administration may use it.
func System(ctx context.Context) context.Context {
	return context.WithValue(ctx, scopeKey{}, scope{system: true})
}

// TenantOf returns the tenant of ctx and whether ctx is a system scope.
func TenantOf(ctx context.Context) (tenant int64, system bool) {
	sc, _ := ctx.Value(scopeKey{}).(scope)
	return sc.tenant, sc.system
}

func scopeOf(ctx context.Context) (scope, error) {
	sc, ok := ctx.Value(scopeKey{}).(scope)
	if !ok || (!sc.system && sc.tenant <= 0) {
		return scope{}, ErrNoTenant
	}
	return sc, nil
}

// filter returns a SQL condition (prefixed with " AND ") restricting col to
// the scope's tenant; system scopes are unrestricted.
func (sc scope) filter(col string) (string, []any) {
	if sc.system {
		return "", nil
	}
	return " AND " + col + " = ?", []any{sc.tenant}
}

// owner resolves the tenant that a new row belongs to: the scope tenant, or
// for system scopes the tenant already set on the object.
func (sc scope) owner(objTenant int64) (int64, error) {
	if !sc.system {
		return sc.tenant, nil
	}
	if objTenant <= 0 {
		return 0, ErrNoTenant
	}
	return objTenant, nil
}

// scoped returns the tenant condition (prefixed with " AND ") for col.
func scoped(ctx context.Context, col string) (string, []any, error) {
	sc, err := scopeOf(ctx)
	if err != nil {
		return "", nil, err
	}
	cond, args := sc.filter(col)
	return cond, args, nil
}

// ownerFor resolves the tenant a new row belongs to.
func ownerFor(ctx context.Context, objTenant int64) (int64, error) {
	sc, err := scopeOf(ctx)
	if err != nil {
		return 0, err
	}
	return sc.owner(objTenant)
}
