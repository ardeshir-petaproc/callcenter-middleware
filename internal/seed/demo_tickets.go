package seed

import (
	"context"
	"time"

	"callcenter-middleware/internal/domain"
	"callcenter-middleware/internal/store"
)

// demoTickets adds SLA policies and a handful of tickets in various states.
func demoTickets(ctx context.Context, st *store.Store, users []*domain.User) error {
	for _, p := range []domain.SLAPolicy{
		{Priority: domain.PriorityUrgent, FirstResponseMinutes: 15, ResolutionMinutes: 240},
		{Priority: domain.PriorityHigh, FirstResponseMinutes: 60, ResolutionMinutes: 480},
		{Priority: domain.PriorityNormal, FirstResponseMinutes: 240, ResolutionMinutes: 1440},
		{Priority: domain.PriorityLow, FirstResponseMinutes: 1440, ResolutionMinutes: 4320},
	} {
		if err := st.SLAPolicies.Create(ctx, &p); err != nil {
			return err
		}
	}
	depts, err := st.Departments.List(ctx)
	if err != nil || len(depts) < 3 {
		return err
	}
	now := time.Now().UTC()
	ago := func(h int) *time.Time { t := now.Add(-time.Duration(h) * time.Hour); return &t }
	samples := []domain.Ticket{
		{Subject: "Double charge on October invoice", Category: "billing", Priority: domain.PriorityHigh,
			Status: domain.TicketInProgress, DepartmentID: depts[2].ID, AssigneeID: users[3].ID, Source: "call",
			RequesterName: "Maria Rossi", RequesterPhone: "+15550100", ResolutionDue: ago(-4)},
		{Subject: "Cannot log in after password reset", Category: "login_issue", Priority: domain.PriorityUrgent,
			Status: domain.TicketNew, DepartmentID: depts[1].ID, Source: "ai", RequesterPhone: "+15550101",
			ResponseDue: ago(1), ResolutionDue: ago(-2)},
		{Subject: "Quote for 40 Pro seats", Category: "sales", Priority: domain.PriorityNormal,
			Status: domain.TicketWaiting, DepartmentID: depts[0].ID, AssigneeID: users[0].ID, Source: "email",
			RequesterName: "Tom Becker", RequesterEmail: "tom@becker.example"},
		{Subject: "Export to CSV fails with large reports", Category: "technical_support", Priority: domain.PriorityNormal,
			Status: domain.TicketAssigned, DepartmentID: depts[1].ID, AssigneeID: users[1].ID, Source: "manual"},
		{Subject: "Complaint about wait times", Category: "complaint", Priority: domain.PriorityLow,
			Status: domain.TicketResolved, DepartmentID: depts[1].ID, AssigneeID: users[4].ID, Source: "call",
			FirstResponse: ago(20), ResolvedAt: ago(2)},
	}
	for i := range samples {
		if err := st.CreateTicket(ctx, &samples[i]); err != nil {
			return err
		}
		if samples[i].FirstResponse != nil || samples[i].ResolvedAt != nil {
			_ = st.UpdateTicket(ctx, &samples[i])
		}
	}
	return nil
}
