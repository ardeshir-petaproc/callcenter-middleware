package core

import (
	"log/slog"
	"time"

	"callcenter-middleware/internal/domain"
)

// tickCampaigns runs the progressive dialer: every idle agent of a running
// campaign's queue gets the next due lead (agent first, then the customer).
// Inbound queues are served first because dispatch runs before this.
func (c *Core) tickCampaigns(now time.Time) {
	if now.Sub(c.lastDial) < 2*time.Second {
		return
	}
	c.lastDial = now
	for i := range c.cache.campaigns {
		cp := &c.cache.campaigns[i]
		if cp.Status != domain.CampaignRunning {
			continue
		}
		if sch, ok := c.cache.schedules[cp.ScheduleID]; ok && cp.ScheduleID > 0 && !IsOpen(sch, now) {
			continue
		}
		if c.waitingInQueue(cp.QueueID) > 0 {
			continue
		}
		for _, m := range c.cache.members[cp.QueueID] {
			a := c.agents[m.UserID]
			if a == nil || !c.ringable(a) {
				continue
			}
			if !c.dialLead(cp, a, now) {
				c.maybeComplete(cp)
				break
			}
		}
	}
}

func (c *Core) waitingInQueue(queueID int64) int {
	n := 0
	for _, call := range c.calls {
		if call.queue != nil && call.queue.ID == queueID && (call.State == domain.CallQueued || call.State == domain.CallRinging) {
			n++
		}
	}
	return n
}

// dialLead connects agent a to the next lead; false means no lead is due.
func (c *Core) dialLead(cp *domain.Campaign, a *agentRT, now time.Time) bool {
	ctx, cancel := c.opCtx()
	defer cancel()
	lead, err := c.Store.NextLead(ctx, cp.ID, now)
	if err != nil || lead == nil {
		return false
	}
	lead.Status = domain.LeadDialing
	lead.Attempts++
	call, err := c.dialLocked(a, lead.Number, func(call *liveCall) {
		call.CampaignID, call.leadID = cp.ID, lead.ID
		if call.CallerName == "" {
			call.CallerName = lead.Name
		}
	})
	if call != nil {
		lead.CallID = call.ID
	}
	if err != nil {
		slog.Warn("Campaign dial failed", "campaign", cp.Name, "lead", lead.ID, "err", err)
		lead.Status = domain.LeadPending
		lead.NextAttempt = now.Add(time.Minute)
		a.retryAfter = now.Add(5 * time.Second)
	}
	_ = c.Store.UpdateLead(ctx, lead)
	return true
}

// updateLead records the outcome of a campaign call on its lead.
func (c *Core) updateLead(call *liveCall) {
	if call.leadID == 0 {
		return
	}
	ctx, cancel := c.opCtx()
	defer cancel()
	lead, err := c.Store.GetLead(ctx, call.leadID)
	if err != nil {
		return
	}
	maxAttempts, retry := 3, 30
	for _, cp := range c.cache.campaigns {
		if cp.ID == call.CampaignID {
			if cp.MaxAttempts > 0 {
				maxAttempts = cp.MaxAttempts
			}
			if cp.RetryMinutes > 0 {
				retry = cp.RetryMinutes
			}
		}
	}
	switch {
	case call.AnsweredAt != nil:
		lead.Status = domain.LeadAnswered
	case call.Outcome == domain.OutcomeFailed && call.HangupBy == "agent":
		// The agent missed the call: retry soon without burning an attempt.
		lead.Status, lead.NextAttempt = domain.LeadPending, c.now().Add(time.Minute)
		lead.Attempts--
	case lead.Attempts >= maxAttempts:
		lead.Status = domain.LeadFailed
	default:
		lead.Status, lead.NextAttempt = domain.LeadPending, c.now().Add(time.Duration(retry)*time.Minute)
	}
	_ = c.Store.UpdateLead(ctx, lead)
}

// maybeComplete marks a running campaign completed when no lead remains.
func (c *Core) maybeComplete(cp *domain.Campaign) {
	ctx, cancel := c.opCtx()
	defer cancel()
	stats, err := c.Store.LeadStats(ctx, cp.ID)
	if err != nil || stats["total"] == 0 || stats[domain.LeadPending] > 0 || stats[domain.LeadDialing] > 0 {
		return
	}
	cp.Status = domain.CampaignCompleted
	if err := c.Store.Campaigns.Update(ctx, cp); err == nil {
		slog.Info("Campaign completed", "campaign", cp.Name)
		c.publish(cp.TenantID, domain.EvConfigChanged, map[string]string{"entity": "campaign"})
	}
}
