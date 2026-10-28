package seed

import (
	"context"
	"fmt"
	"math/rand/v2"
	"time"

	"callcenter-middleware/internal/domain"
	"callcenter-middleware/internal/store"

	"github.com/google/uuid"
)

func id(v int64) string { return fmt.Sprint(v) }

func demoConfig(ctx context.Context, st *store.Store, queues []*domain.Queue) error {
	ai := &domain.AIProfile{
		Name: "Concierge", Voice: "marin", Language: "en", Enabled: true, MaxDurationSec: 600,
		HandoffQueueID: queues[1].ID,
		Greeting:       "Thanks for calling Acme. I'm the virtual assistant. How can I help you today?",
		Instructions: "You are Acme's friendly virtual receptionist. Answer questions about opening hours, " +
			"order status and plans. Collect the caller's name and reason for calling before transferring.",
		KnowledgeBase: "Opening hours: Monday to Friday 9:00-17:00.\nPlans: Basic $10/month, Pro $25/month, " +
			"Enterprise on request.\nRefunds are possible within 30 days.",
		Tools: []string{domain.ToolTransferToQueue, domain.ToolEndCall, domain.ToolCreateCallback,
			domain.ToolLookupContact, domain.ToolTakeMessage, domain.ToolCheckHours, domain.ToolRouteCall,
			domain.ToolSearchKnowledge, domain.ToolCreateTicket},
	}
	if err := st.AIProfiles.Create(ctx, ai); err != nil {
		return err
	}
	sch := &domain.Schedule{Name: "Business hours", Timezone: "UTC", Holidays: []domain.Holiday{{Date: "2026-12-25", Name: "Christmas"}}}
	for d := 1; d <= 5; d++ {
		sch.Weekly = append(sch.Weekly, domain.TimeRange{Day: d, Start: "09:00", End: "17:00"})
	}
	if err := st.Schedules.Create(ctx, sch); err != nil {
		return err
	}
	flow := &domain.IVRFlow{
		Name: "Main menu", DNIS: "*", GreetingMedia: "sound:custom/ivr-welcome", InvalidMedia: "sound:option-is-invalid",
		TimeoutSec: 5, MaxRetries: 2, Enabled: true,
		Options: map[string]domain.Action{
			"1": {Type: domain.ActionQueue, Target: id(queues[0].ID)},
			"2": {Type: domain.ActionQueue, Target: id(queues[1].ID)},
			"3": {Type: domain.ActionQueue, Target: id(queues[2].ID)},
			"8": {Type: domain.ActionCallback},
			"9": {Type: domain.ActionAI, Target: id(ai.ID)},
			"0": {Type: domain.ActionVoicemail},
		},
		TimeoutAction: domain.Action{Type: domain.ActionAI, Target: id(ai.ID)},
	}
	if err := st.Flows.Create(ctx, flow); err != nil {
		return err
	}
	aiLine := &domain.IVRFlow{Name: "AI hotline", DNIS: "700", Enabled: true, TimeoutSec: 5,
		DirectAction: domain.Action{Type: domain.ActionAI, Target: id(ai.ID)}, Options: map[string]domain.Action{}}
	if err := st.Flows.Create(ctx, aiLine); err != nil {
		return err
	}
	contacts := []domain.Contact{
		{Name: "Maria Rossi", Phone: "+15550100", Email: "maria@rossi.example", Company: "Rossi Imports", VIP: true, Tags: []string{"enterprise"}},
		{Name: "John Smith", Phone: "+15550101", Email: "john@smith.example", Company: "Smith & Co", Tags: []string{"pro"}},
		{Name: "Leila Ahmadi", Phone: "+989121234567", Email: "leila@example.com", Tags: []string{"basic"}},
		{Name: "Tom Becker", Phone: "+4915112345678", Company: "Becker GmbH", Notes: "Prefers email follow-up", Tags: []string{}},
	}
	for i := range contacts {
		if err := st.Contacts.Create(ctx, &contacts[i]); err != nil {
			return err
		}
	}
	cp := &domain.Campaign{Name: "Q4 renewals", Description: "Call customers whose plan renews this month",
		QueueID: queues[0].ID, Status: domain.CampaignDraft, MaxAttempts: 3, RetryMinutes: 60,
		Script: "Hi, this is {agent} from Acme. Your plan renews soon. Can I walk you through the Pro upgrade?"}
	if err := st.Campaigns.Create(ctx, cp); err != nil {
		return err
	}
	leads := []domain.Lead{{Number: "+15550101", Name: "John Smith"}, {Number: "+15550123", Name: "Ava Patel"},
		{Number: "+15550145", Name: "Noah Kim"}, {Number: "+15550167", Name: "Emma Garcia"}}
	_, err := st.AddLeads(ctx, cp.ID, leads)
	return err
}

// demoHistory fabricates plausible call records for the last 14 days so
// reports and dashboards have something to show in a fresh install.
func demoHistory(ctx context.Context, st *store.Store, queues []*domain.Queue, agents []*domain.User) error {
	rng := rand.New(rand.NewPCG(42, 7))
	now := time.Now().UTC()
	dispositions := []string{"resolved", "resolved", "follow_up", "sale", "complaint"}
	for day := 13; day >= 0; day-- {
		base := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC).AddDate(0, 0, -day)
		n := 15 + rng.IntN(25)
		for i := 0; i < n; i++ {
			start := base.Add(time.Duration(8*60+rng.IntN(10*60)) * time.Minute)
			if start.After(now) {
				continue
			}
			q := queues[rng.IntN(len(queues))]
			c := &domain.Call{ID: uuid.NewString(), Direction: domain.DirectionInbound, State: domain.CallEnded,
				CallerNumber: fmt.Sprintf("+1555%07d", rng.IntN(10000000)), DNIS: "100", QueueID: q.ID,
				StartedAt: start, IVRPath: fmt.Sprint(rng.IntN(3) + 1), Tags: []string{}}
			queued := start.Add(time.Duration(5+rng.IntN(10)) * time.Second)
			c.QueuedAt = &queued
			wait := min(int(rng.ExpFloat64()*14), 150)
			roll := rng.IntN(100)
			switch {
			case roll < 72:
				ans := queued.Add(time.Duration(wait) * time.Second)
				end := ans.Add(time.Duration(60+rng.IntN(540)) * time.Second)
				c.AnsweredAt, c.EndedAt = &ans, &end
				c.AgentID = agents[rng.IntN(len(agents))].ID
				c.Outcome, c.WaitSec, c.TalkSec = domain.OutcomeAnswered, wait, int(end.Sub(ans).Seconds())
				c.SLAMet = wait <= q.SLASec
				c.Disposition = dispositions[rng.IntN(len(dispositions))]
				c.Sentiment = []string{"positive", "neutral", "neutral", "negative"}[rng.IntN(4)]
				if rng.IntN(3) == 0 {
					c.CSAT = 3 + rng.IntN(3)
				}
			case roll < 84:
				end := queued.Add(time.Duration(wait) * time.Second)
				c.EndedAt, c.Outcome, c.WaitSec = &end, domain.OutcomeAbandoned, wait
			case roll < 92:
				end := start.Add(time.Duration(30+rng.IntN(120)) * time.Second)
				c.EndedAt, c.Outcome, c.QueuedAt, c.QueueID = &end, domain.OutcomeAIResolved, nil, 0
				c.Summary = "Caller asked about opening hours; AI answered and ended the call."
			default:
				end := queued.Add(time.Duration(wait+40) * time.Second)
				c.EndedAt, c.Outcome, c.WaitSec = &end, domain.OutcomeVoicemail, wait
			}
			if err := st.SaveCall(ctx, c); err != nil {
				return err
			}
		}
	}
	return nil
}
