package store

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"callcenter-middleware/internal/domain"
)

func TestTenantIsolation(t *testing.T) {
	st := open(t)
	bg := context.Background()
	acme := &domain.Tenant{Slug: "acme", Name: "Acme"}
	if err := st.CreateTenant(bg, acme); err != nil {
		t.Fatal(err)
	}
	t1, t2 := ForTenant(bg, 1), ForTenant(bg, acme.ID)

	q := &domain.Queue{Name: "Sales"}
	if err := st.Queues.Create(t1, q); err != nil || q.TenantID != 1 {
		t.Fatalf("create: %v %+v", err, q)
	}
	if _, err := st.Queues.Get(t2, q.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("tenant 2 must not see tenant 1 queue, got %v", err)
	}
	if list, _ := st.Queues.List(t2); len(list) != 0 {
		t.Fatalf("tenant 2 list leaked %+v", list)
	}
	q.Name = "hijack"
	if err := st.Queues.Update(t2, q); !errors.Is(err, ErrNotFound) {
		t.Fatalf("cross-tenant update must fail, got %v", err)
	}
	if err := st.Queues.Delete(t2, q.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("cross-tenant delete must fail, got %v", err)
	}

	// Same username in two tenants is allowed; lookups stay scoped.
	for _, ctx := range []context.Context{t1, t2} {
		if err := st.CreateUser(ctx, &domain.User{Username: "ali", Role: domain.RoleAgent, PasswordHash: "h"}); err != nil {
			t.Fatal(err)
		}
	}
	u1, _ := st.GetUserByUsername(t1, "ali")
	if _, err := st.GetUser(t2, u1.ID); !errors.Is(err, ErrNotFound) {
		t.Fatal("tenant 2 must not load tenant 1 user")
	}
	if err := st.SetUserQueues(t2, u1.ID, []domain.QueueMember{{QueueID: q.ID}}); err == nil {
		t.Fatal("cross-tenant membership must be rejected")
	}

	call := &domain.Call{ID: "c1", Direction: "inbound", State: domain.CallEnded, StartedAt: time.Now()}
	if err := st.SaveCall(t1, call); err != nil {
		t.Fatal(err)
	}
	_ = st.AddTranscript(bg, domain.TranscriptEntry{CallID: "c1", Speaker: "caller", Text: "secret", At: time.Now()})
	if _, err := st.GetCall(t2, "c1"); !errors.Is(err, ErrNotFound) {
		t.Fatal("tenant 2 must not load tenant 1 call")
	}
	if tr, _ := st.Transcript(t2, "c1"); len(tr) != 0 {
		t.Fatal("transcript leaked across tenants")
	}
	if calls, total, _ := st.ListCalls(t2, domain.CallFilter{}); total != 0 || len(calls) != 0 {
		t.Fatal("call history leaked across tenants")
	}
	_ = st.SetSetting(t1, "company_name", "One")
	if v := st.GetSetting(t2, "company_name", "none"); v != "none" {
		t.Fatalf("setting leaked: %s", v)
	}
	if _, err := st.ListUsers(bg); !errors.Is(err, ErrNoTenant) {
		t.Fatal("unscoped access must fail closed")
	}
	all, _ := st.ListUsers(System(bg))
	if len(all) != 2 {
		t.Fatalf("system scope should see both tenants, got %d", len(all))
	}
}

func TestMigratesSingleTenantDatabase(t *testing.T) {
	path := filepath.Join(t.TempDir(), "old.db")
	raw, err := sql.Open("sqlite", "file:"+path)
	if err != nil {
		t.Fatal(err)
	}
	for _, stmt := range migrations {
		if _, err := raw.Exec(stmt); err != nil {
			t.Fatal(err)
		}
	}
	now := time.Now().UnixMilli()
	mustExec(t, raw, `INSERT INTO users (username, role, password_hash, created_at, updated_at) VALUES ('boss', 'admin', 'h', ?, ?)`, now, now)
	mustExec(t, raw, `INSERT INTO entities (kind, data, created_at, updated_at) VALUES ('queue', '{"name":"Old"}', ?, ?)`, now, now)
	mustExec(t, raw, `INSERT INTO queue_members (queue_id, user_id, penalty) VALUES (1, 1, 0)`)
	mustExec(t, raw, `INSERT INTO settings (key, value) VALUES ('company_name', 'Legacy Inc')`)
	raw.Close()

	st, err := Open(path)
	if err != nil {
		t.Fatalf("migrate: %v", err)
	}
	defer st.Close()
	ctx := ForTenant(context.Background(), 1)
	u, err := st.GetUserByUsername(ctx, "boss")
	if err != nil || u.TenantID != 1 || !u.PlatformAdmin {
		t.Fatalf("user not migrated: %v %+v", err, u)
	}
	if qs, _ := st.Queues.List(ctx); len(qs) != 1 || qs[0].Name != "Old" {
		t.Fatalf("queue not migrated: %+v", qs)
	}
	if ms, _ := st.ListMembers(ctx); len(ms) != 1 {
		t.Fatal("membership lost during users table rebuild")
	}
	if st.GetSetting(ctx, "company_name", "") != "Legacy Inc" {
		t.Fatal("settings not migrated")
	}
}

func mustExec(t *testing.T, db *sql.DB, q string, args ...any) {
	t.Helper()
	if _, err := db.Exec(q, args...); err != nil {
		t.Fatal(err)
	}
}
