package core

import (
	"sort"

	"callcenter-middleware/internal/domain"
)

// Snapshot returns a consistent copy of one tenant's live state
// (tenant 0 returns every tenant, for platform tooling and tests).
func (c *Core) Snapshot(tenant int64) Snapshot {
	var s Snapshot
	_ = c.do(func() error {
		s = c.snapshot(tenant)
		return nil
	})
	return s
}

func (c *Core) snapshot(tenant int64) Snapshot {
	s := Snapshot{Calls: []CallView{}, Agents: []domain.AgentState{}, Queues: []QueueView{}, At: c.now().UTC()}
	for _, call := range c.calls {
		if tenant == 0 || call.TenantID == tenant {
			s.Calls = append(s.Calls, c.view(call))
		}
	}
	sort.Slice(s.Calls, func(i, j int) bool { return s.Calls[i].StartedAt.Before(s.Calls[j].StartedAt) })
	for _, a := range c.agents {
		if tenant != 0 && a.TenantID != tenant {
			continue
		}
		st := a.AgentState
		if a.pendingStatus != "" {
			st.PauseReason = "next: " + string(a.pendingStatus)
		}
		s.Agents = append(s.Agents, st)
	}
	sort.Slice(s.Agents, func(i, j int) bool { return s.Agents[i].UserID < s.Agents[j].UserID })
	s.Queues = c.queueViews(tenant)
	s.Alerts = []QueueAlert{}
	for _, v := range s.Queues {
		if c.alerts[v.ID] {
			s.Alerts = append(s.Alerts, QueueAlert{QueueID: v.ID, Name: v.Name, Level: "alert",
				Waiting: v.Waiting, LongestWaitSec: v.LongestWaitSec, Available: v.Available})
		}
	}
	if c.AI != nil {
		s.AI = c.AI.Active()
	}
	return s
}

func (c *Core) queueViews(tenant int64) []QueueView {
	out := []QueueView{}
	now := c.now()
	for _, q := range c.cache.sortedQueues(tenant) {
		v := QueueView{ID: q.ID, Name: q.Name, Members: len(c.cache.members[q.ID])}
		for _, m := range c.cache.members[q.ID] {
			a := c.agents[m.UserID]
			if a == nil {
				continue
			}
			switch a.Status {
			case domain.AgentAvailable:
				v.Available++
			case domain.AgentBusy, domain.AgentRinging, domain.AgentWrapup:
				v.Busy++
			case domain.AgentPaused:
				v.Paused++
			}
			if a.Status != domain.AgentOffline {
				v.LoggedIn++
			}
		}
		for _, call := range c.calls {
			if call.queue == nil || call.queue.ID != q.ID {
				continue
			}
			if call.State == domain.CallQueued || call.State == domain.CallRinging {
				v.Waiting++
				if w := secs(now.Sub(call.queueEnter)); w > v.LongestWaitSec {
					v.LongestWaitSec = w
				}
			}
		}
		out = append(out, v)
	}
	return out
}

// Agent returns the live state of one agent.
func (c *Core) Agent(userID int64) (domain.AgentState, bool) {
	var st domain.AgentState
	ok := false
	_ = c.do(func() error {
		if a := c.agents[userID]; a != nil {
			st, ok = a.AgentState, true
		}
		return nil
	})
	return st, ok
}

func (c *Core) updateGauges() {
	if c.Metrics == nil {
		c.checkAlerts()
		return
	}
	for _, v := range c.queueViews(0) {
		c.Metrics.SetQueueWaiting(v.Name, v.Waiting)
	}
	c.checkAlerts()
	counts := map[string]int{}
	for _, a := range c.agents {
		counts[string(a.Status)]++
	}
	c.Metrics.SetAgentsByStatus(counts)
	if c.AI != nil {
		c.Metrics.SetAISessions(c.AI.Active())
	}
}
