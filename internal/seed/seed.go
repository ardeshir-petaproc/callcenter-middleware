// Package seed bootstraps the admin account and optional demo data.
package seed

import (
	"context"
	"fmt"
	"log/slog"

	"callcenter-middleware/internal/auth"
	"callcenter-middleware/internal/domain"
	"callcenter-middleware/internal/store"
)

// EnsureAdmin creates the bootstrap admin when the user table is empty.
func EnsureAdmin(ctx context.Context, st *store.Store, username, password string) error {
	n, err := st.CountUsers(ctx)
	if err != nil || n > 0 {
		return err
	}
	hash, err := auth.HashPassword(password)
	if err != nil {
		return err
	}
	u := &domain.User{Username: username, DisplayName: "Administrator", Role: domain.RoleAdmin,
		Enabled: true, PasswordHash: hash, Skills: []string{}}
	if err := st.CreateUser(ctx, u); err != nil {
		return err
	}
	slog.Warn("Bootstrap admin created; change its password", "username", username)
	return nil
}

// EnsureDefaults adds dispositions and pause reasons on first start.
func EnsureDefaults(ctx context.Context, st *store.Store) error {
	if n, _ := st.Dispositions.Count(ctx); n == 0 {
		for _, d := range [][2]string{{"resolved", "Resolved"}, {"follow_up", "Needs follow-up"},
			{"sale", "Sale"}, {"complaint", "Complaint"}, {"wrong_number", "Wrong number"}, {"escalated", "Escalated"}} {
			if err := st.Dispositions.Create(ctx, &domain.Disposition{Code: d[0], Label: d[1], Enabled: true}); err != nil {
				return err
			}
		}
	}
	if n, _ := st.PauseReasons.Count(ctx); n == 0 {
		for _, l := range []string{"Break", "Lunch", "Meeting", "Training", "After-call work", "Missed calls"} {
			if err := st.PauseReasons.Create(ctx, &domain.PauseReason{Label: l, Enabled: true}); err != nil {
				return err
			}
		}
	}
	return nil
}

// Demo creates sample agents, queues, flows, an AI profile, contacts and
// two weeks of call history. It only runs when no queues exist yet.
func Demo(ctx context.Context, st *store.Store) error {
	if n, _ := st.Queues.Count(ctx); n > 0 {
		return nil
	}
	hash, err := auth.HashPassword("agent123")
	if err != nil {
		return err
	}
	people := []struct {
		user, name, ext string
		role            domain.Role
		skills          []string
	}{
		{"alice", "Alice Moreno", "1001", domain.RoleAgent, []string{"english", "sales"}},
		{"bob", "Bob Chen", "1002", domain.RoleAgent, []string{"english", "support"}},
		{"carol", "Carol Haddad", "1003", domain.RoleAgent, []string{"farsi", "support", "billing"}},
		{"dave", "Dave Okafor", "1004", domain.RoleAgent, []string{"english", "billing"}},
		{"sara", "Sara Lindqvist", "1005", domain.RoleSupervisor, []string{"english"}},
	}
	var users []*domain.User
	for _, p := range people {
		u := &domain.User{Username: p.user, DisplayName: p.name, Email: p.user + "@example.com", Role: p.role,
			Extension: p.ext, Skills: p.skills, Enabled: true, PasswordHash: hash}
		if err := st.CreateUser(ctx, u); err != nil {
			return fmt.Errorf("seed user %s: %w", p.user, err)
		}
		users = append(users, u)
	}
	queues := []*domain.Queue{
		{Name: "Sales", Description: "New customers and upgrades", Strategy: domain.StrategyLongestIdle, Priority: 5,
			RingTimeoutSec: 20, MaxWaitSec: 300, WrapupSec: 15, SLASec: 20, Record: true, AllowCallback: true},
		{Name: "Support", Description: "Technical support", Strategy: domain.StrategyRingAll, Priority: 3,
			RingTimeoutSec: 20, MaxWaitSec: 600, WrapupSec: 20, SLASec: 30, Record: true, AllowCallback: true, Survey: true},
		{Name: "Billing", Description: "Invoices and payments", Strategy: domain.StrategyRoundRobin, Priority: 1,
			RingTimeoutSec: 20, MaxWaitSec: 240, WrapupSec: 10, SLASec: 30, LeaveWhenEmpty: true},
	}
	for _, q := range queues {
		q.Mode, q.Enabled, q.RequiredSkills = domain.QueueModeNative, true, []string{}
		q.Overflow = domain.Action{Type: domain.ActionVoicemail}
		if err := st.Queues.Create(ctx, q); err != nil {
			return err
		}
	}
	members := map[int][]int{0: {0, 1}, 1: {1, 2, 3}, 2: {2, 3}}
	for qi, uis := range members {
		var ms []domain.QueueMember
		for i, ui := range uis {
			ms = append(ms, domain.QueueMember{QueueID: queues[qi].ID, UserID: users[ui].ID, Penalty: i})
		}
		if err := st.SetQueueMembers(ctx, queues[qi].ID, ms); err != nil {
			return err
		}
	}
	if err := demoConfig(ctx, st, queues); err != nil {
		return err
	}
	return demoHistory(ctx, st, queues, users[:4])
}
