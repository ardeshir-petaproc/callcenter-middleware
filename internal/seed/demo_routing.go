package seed

import (
	"context"
	"time"

	"callcenter-middleware/internal/domain"
	"callcenter-middleware/internal/store"
)

// demoRouting adds request topics and routing rules (mostly approved, one
// AI draft awaiting approval) to the demo organization.
func demoRouting(ctx context.Context, st *store.Store, depts []*domain.Department, jobs []*domain.JobProfile, supervisor *domain.User) error {
	jobs[2].EscalateToUserID = supervisor.ID
	if err := st.JobProfiles.Update(ctx, jobs[2]); err != nil {
		return err
	}
	topics := []*domain.Topic{
		{Name: "Sales", Description: "Buying, quotes and upgrades", DepartmentID: depts[0].ID,
			Keywords: []string{"buy", "price", "quote", "upgrade", "plan", "خرید", "قیمت", "ارتقا"}},
		{Name: "Billing", Description: "Invoices and payments", DepartmentID: depts[2].ID,
			Keywords: []string{"invoice", "bill", "payment", "charged", "صورتحساب", "فاکتور", "پرداخت"}},
		{Name: "Refund", Description: "Money back requests", DepartmentID: depts[2].ID,
			Keywords: []string{"refund", "money back", "بازپرداخت", "استرداد", "پس گرفتن پول"}},
		{Name: "Technical support", Description: "Login problems and errors", DepartmentID: depts[1].ID,
			Keywords: []string{"login", "password", "error", "not working", "رمز", "خطا", "کار نمی کند"}},
	}
	for _, t := range topics {
		if err := st.Topics.Create(ctx, t); err != nil {
			return err
		}
	}
	now := time.Now().UTC().Format(time.RFC3339)
	approved := func(r *domain.RoutingRule) *domain.RoutingRule {
		r.Status, r.Source, r.Version, r.ApprovedBy, r.ApprovedAt, r.UpdatedAt = domain.RuleApproved, "manual", 1, supervisor.ID, now, now
		return r
	}
	rules := []*domain.RoutingRule{
		approved(&domain.RoutingRule{Name: "Refunds go to billing specialists", Topics: []string{"Refund"},
			TargetType: domain.TargetJobProfile, TargetID: jobs[2].ID, Priority: 1, JobProfileID: jobs[2].ID,
			Rationale: "Billing specialists may issue refunds up to 200 USD."}),
		approved(&domain.RoutingRule{Name: "Chargebacks and disputes go to the supervisor",
			Keywords: []string{"chargeback", "dispute", "lawyer", "شکایت", "وکیل"}, TargetType: domain.TargetUser,
			TargetID: supervisor.ID, Priority: 5, Rationale: "Beyond billing specialists' authority.",
			Fallback: domain.Action{Type: domain.ActionVoicemail}}),
		approved(&domain.RoutingRule{Name: "Technical problems go to support agents", Topics: []string{"Technical support"},
			TargetType: domain.TargetJobProfile, TargetID: jobs[1].ID, JobProfileID: jobs[1].ID,
			Rationale: "Support agents handle account access and troubleshooting."}),
		{Name: "Pricing questions go to sales representatives", Topics: []string{"Sales"}, Status: domain.RuleDraft,
			Source: "ai", Version: 1, UpdatedAt: now, TargetType: domain.TargetJobProfile, TargetID: jobs[0].ID,
			JobProfileID: jobs[0].ID, Rationale: "Job profile lists pricing and upgrades as request types."},
	}
	for _, r := range rules {
		if r.Languages == nil {
			r.Languages, r.Keywords = []string{}, append([]string{}, r.Keywords...)
		}
		if err := st.Rules.Create(ctx, r); err != nil {
			return err
		}
	}
	return nil
}
