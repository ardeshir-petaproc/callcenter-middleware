package integration

import (
	"callcenter-middleware/internal/domain"
	"encoding/json"
	"strings"
	"testing"
)

func TestPermissions_CustomRolesAndPlatformAdmin(t *testing.T) {
	s := newStack(t, 20)
	admin := s.login("admin", "admin123")
	var role domain.CustomRole
	s.call("POST", "/roles", admin, domain.CustomRole{Name: "Analyst", Permissions: []string{domain.PermReportsView}}, &role)
	var users []domain.User
	s.call("GET", "/users", admin, nil, &users)
	var carol domain.User
	for _, u := range users {
		if u.Username == "carol" {
			carol = u
		}
	}
	agent := s.login("carol", "agent123")
	if code := s.call("GET", "/reports/summary", agent, nil, nil); code != 403 {
		t.Fatalf("agent without the role must be refused, got %d", code)
	}
	carol.CustomRoleID = role.ID
	if code := s.call("PUT", "/users/"+itoa(carol.ID), admin, carol, nil); code != 200 {
		t.Fatalf("assign role: %d", code)
	}
	if code := s.call("GET", "/reports/summary", agent, nil, nil); code != 200 {
		t.Fatalf("custom role should grant reports, got %d", code)
	}
	if code := s.call("POST", "/roles", admin, domain.CustomRole{Name: "x", Permissions: []string{"root"}}, nil); code != 400 {
		t.Fatalf("unknown permission must be rejected, got %d", code)
	}
	shopAdmin := s.loginTenant("admin", "admin123", "parsian")
	if code := s.call("GET", "/platform/tenants", shopAdmin, nil, nil); code != 403 {
		t.Fatalf("tenant admins are not platform admins, got %d", code)
	}
	body := map[string]any{"slug": "new-co", "name": "New Co", "owner": map[string]string{"username": "owner", "password": "ownerpass1"}}
	if code := s.call("POST", "/platform/tenants", admin, body, nil); code != 201 {
		t.Fatalf("platform admin creates tenant: %d", code)
	}
	owner := s.loginTenant("owner", "ownerpass1", "new-co")
	var disp []json.RawMessage
	s.call("GET", "/dispositions", owner, nil, &disp)
	if len(disp) == 0 {
		t.Fatal("new tenant should get default dispositions")
	}
}

func itoa(v int64) string { return strings.TrimSpace(strings.Replace(jsonNum(v), "\n", "", -1)) }

func jsonNum(v int64) string { b, _ := json.Marshal(v); return string(b) }
