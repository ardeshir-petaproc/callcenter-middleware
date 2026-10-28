package seed

import (
	"context"
	"time"

	"callcenter-middleware/internal/auth"
	"callcenter-middleware/internal/domain"
	"callcenter-middleware/internal/store"
)

// demoOrg adds departments, job profiles, employee profiles linked to Joow
// accounts and phone numbers to the default demo organization.
func demoOrg(ctx context.Context, st *store.Store, queues []*domain.Queue, users []*domain.User) error {
	depts := []*domain.Department{
		{Name: "Sales", Description: "فروش", Priority: 2, Capacity: 20, Enabled: true},
		{Name: "Support", Description: "پشتیبانی", Priority: 1, Enabled: true},
		{Name: "Finance", Description: "حسابداری", Enabled: true},
	}
	for _, d := range depts {
		d.Fallback, d.Languages = domain.Action{Type: domain.ActionVoicemail}, []string{"en", "fa"}
		if err := st.Departments.Create(ctx, d); err != nil {
			return err
		}
	}
	for i, q := range queues { // Sales, Support, Billing -> Sales, Support, Finance
		q.DepartmentID = depts[i].ID
		if err := st.Queues.Update(ctx, q); err != nil {
			return err
		}
	}
	jobs := []*domain.JobProfile{
		{Title: "Sales representative", DepartmentID: depts[0].ID, RequestTypes: []string{"sales", "pricing", "upgrade"},
			Description: "Pre-sales questions, quotes and plan upgrades.", AuthorityLimits: "Discounts up to 10%.",
			EscalationConditions: "Larger discounts and enterprise contracts go to the sales manager."},
		{Title: "Support agent", DepartmentID: depts[1].ID, RequestTypes: []string{"login_issue", "technical_support"},
			Description: "Account access and product troubleshooting.", AuthorityLimits: "No refunds.",
			EscalationConditions: "Outages and security incidents go to the supervisor."},
		{Title: "Billing specialist", DepartmentID: depts[2].ID, RequestTypes: []string{"billing", "refund", "payment"},
			Description: "Invoices, payments and refunds.", AuthorityLimits: "Refunds up to 200 USD.",
			EscalationConditions: "Chargebacks and disputed contracts go to finance."},
	}
	for _, j := range jobs {
		j.Version, j.UpdatedAt, j.Responsibilities = 1, time.Now().UTC().Format(time.RFC3339), []string{}
		if err := st.JobProfiles.Create(ctx, j); err != nil {
			return err
		}
	}
	profiles := []domain.Profile{
		{Title: "Sales representative", DepartmentIDs: []int64{depts[0].ID}, Languages: []string{"en", "es"}, JobProfileID: jobs[0].ID},
		{Title: "Support agent", DepartmentIDs: []int64{depts[0].ID, depts[1].ID}, Languages: []string{"en"}, JobProfileID: jobs[1].ID},
		{Title: "Billing specialist", DepartmentIDs: []int64{depts[1].ID, depts[2].ID}, Languages: []string{"fa", "en", "ar"}, JobProfileID: jobs[2].ID},
		{Title: "Billing specialist", DepartmentIDs: []int64{depts[2].ID}, Languages: []string{"en", "fr"}, JobProfileID: jobs[2].ID},
		{Title: "Contact center supervisor", DepartmentIDs: []int64{depts[0].ID, depts[1].ID, depts[2].ID}, Languages: []string{"en", "sv"}},
	}
	for i, u := range users {
		u.Profile, u.JoowAccountID = profiles[i], "j-"+u.Username
		if i < 4 {
			u.ManagerID = users[4].ID
		}
		if err := st.UpdateUser(ctx, u); err != nil {
			return err
		}
	}
	flows, err := st.Flows.List(ctx)
	if err != nil {
		return err
	}
	for _, f := range flows {
		dnis := map[string]string{"*": "100", "700": "700"}[f.DNIS]
		n := &domain.PhoneNumber{Number: dnis, Label: f.Name, FlowID: f.ID, Enabled: true}
		if dnis == "" {
			continue
		}
		if err := st.Numbers.Create(ctx, n); err != nil {
			return err
		}
	}
	return nil
}

// DemoSecondTenant creates a second organization so tenant isolation can be
// demonstrated: its own admin, agents, queue and phone number (200).
func DemoSecondTenant(root context.Context, st *store.Store) error {
	if _, err := st.GetTenantBySlug(root, "parsian"); err == nil {
		return nil
	}
	t := &domain.Tenant{Slug: "parsian", Name: "Parsian Online Shop", Plan: "pro"}
	if err := st.CreateTenant(root, t); err != nil {
		return err
	}
	ctx := store.ForTenant(root, t.ID)
	adminHash, _ := auth.HashPassword("admin123")
	agentHash, _ := auth.HashPassword("agent123")
	admin := &domain.User{Username: "admin", DisplayName: "مدیر پارسیان", Role: domain.RoleAdmin, Enabled: true, PasswordHash: adminHash}
	reza := &domain.User{Username: "reza", DisplayName: "رضا احمدی", Role: domain.RoleAgent, Extension: "2001",
		Enabled: true, PasswordHash: agentHash, JoowAccountID: "j-reza", Profile: domain.Profile{Languages: []string{"fa"}}}
	// Bob also works for this shop part-time: same Joow account, two organizations.
	bob := &domain.User{Username: "bob", DisplayName: "Bob Chen", Role: domain.RoleAgent, Extension: "2002",
		Enabled: true, PasswordHash: agentHash, JoowAccountID: "j-bob", Profile: domain.Profile{Languages: []string{"en"}}}
	for _, u := range []*domain.User{admin, reza, bob} {
		if err := st.CreateUser(ctx, u); err != nil {
			return err
		}
	}
	q := &domain.Queue{Name: "پشتیبانی فروشگاه", Mode: domain.QueueModeNative, Strategy: domain.StrategyLongestIdle,
		RingTimeoutSec: 20, MaxWaitSec: 300, SLASec: 30, Enabled: true, RequiredSkills: []string{},
		Overflow: domain.Action{Type: domain.ActionVoicemail}}
	if err := st.Queues.Create(ctx, q); err != nil {
		return err
	}
	if err := st.SetQueueMembers(ctx, q.ID, []domain.QueueMember{{UserID: reza.ID}, {UserID: bob.ID, Penalty: 1}}); err != nil {
		return err
	}
	flow := &domain.IVRFlow{Name: "Shop line", DNIS: "200", Enabled: true, TimeoutSec: 5, Options: map[string]domain.Action{},
		DirectAction: domain.Action{Type: domain.ActionQueue, Target: itoa(q.ID)}}
	if err := st.Flows.Create(ctx, flow); err != nil {
		return err
	}
	if err := st.Numbers.Create(ctx, &domain.PhoneNumber{Number: "200", Label: "Shop line", FlowID: flow.ID, Enabled: true}); err != nil {
		return err
	}
	return EnsureDefaults(ctx, st)
}

func itoa(v int64) string { return id(v) }
