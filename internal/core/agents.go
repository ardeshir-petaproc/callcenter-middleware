package core

import (
	"errors"
	"time"

	"callcenter-middleware/internal/domain"
)

// ErrUnknownAgent is returned for status changes of unknown users.
var ErrUnknownAgent = errors.New("unknown agent")

// syncAgents reconciles runtime agent state with the user list.
func (c *Core) syncAgents() {
	for id, a := range c.agents {
		u, ok := c.cache.users[id]
		if !ok {
			delete(c.agents, id)
			continue
		}
		a.user = *u
		a.Username, a.DisplayName, a.Extension = u.Username, u.DisplayName, u.Extension
		a.QueueIDs = c.cache.queuesOf(id)
		if !u.Enabled && a.Status != domain.AgentOffline && a.CallID == "" {
			c.setAgentStatus(a, domain.AgentOffline, "disabled")
		}
	}
	for id, u := range c.cache.users {
		if _, ok := c.agents[id]; ok {
			continue
		}
		c.agents[id] = &agentRT{
			AgentState: domain.AgentState{
				UserID: id, Username: u.Username, DisplayName: u.DisplayName, Extension: u.Extension,
				Status: domain.AgentOffline, Since: c.now().UTC(), QueueIDs: c.cache.queuesOf(id),
			},
			user:      *u,
			idleSince: c.now(),
		}
	}
}

// setAgentStatus changes presence, records it and broadcasts the change.
func (c *Core) setAgentStatus(a *agentRT, st domain.AgentStatus, reason string) {
	if a.Status == st && a.PauseReason == reason {
		return
	}
	prev := a.Status
	a.Status, a.PauseReason, a.Since = st, reason, c.now().UTC()
	if st != domain.AgentWrapup {
		a.WrapupUntil = time.Time{}
	}
	if st == domain.AgentAvailable && prev != domain.AgentRinging {
		a.idleSince = c.now()
	}
	ctx, cancel := c.opCtx()
	_ = c.Store.AddAgentEvent(ctx, domain.AgentEvent{UserID: a.UserID, Status: st, Reason: reason, At: a.Since})
	cancel()
	c.publish(domain.EvAgentUpdated, a.AgentState)
}

// agentCallEnded moves an agent into wrap-up after a call.
func (c *Core) agentCallEnded(a *agentRT, call *liveCall) {
	a.CallID = ""
	a.CallsToday++
	a.TalkSecs += call.TalkSec
	a.LastCallEnd = c.now().UTC()
	a.idleSince = c.now()
	wrap := c.cfg.DefaultWrapupSec
	if call.queue != nil && call.queue.WrapupSec > 0 {
		wrap = call.queue.WrapupSec
	}
	if a.pendingStatus != "" {
		st, reason := a.pendingStatus, a.pendingReason
		a.pendingStatus, a.pendingReason = "", ""
		c.setAgentStatus(a, st, reason)
		return
	}
	if wrap > 0 {
		a.WrapupUntil = c.now().Add(time.Duration(wrap) * time.Second).UTC()
		c.setAgentStatus(a, domain.AgentWrapup, "")
		return
	}
	c.setAgentStatus(a, domain.AgentAvailable, "")
}

func (c *Core) tickAgents(now time.Time) {
	day := now.YearDay()
	for _, a := range c.agents {
		if a.Status == domain.AgentWrapup && !a.WrapupUntil.IsZero() && now.After(a.WrapupUntil) {
			c.setAgentStatus(a, domain.AgentAvailable, "")
		}
		if a.LastCallEnd.YearDay() != day && !a.LastCallEnd.IsZero() && a.CallID == "" {
			a.CallsToday, a.TalkSecs, a.LastCallEnd = 0, 0, time.Time{}
		}
	}
}

// SetAgentStatus is called from the API when an agent logs in, pauses,
// becomes ready or logs out. Changes during a call apply after it ends.
func (c *Core) SetAgentStatus(userID int64, st domain.AgentStatus, reason string) error {
	return c.do(func() error {
		a := c.agents[userID]
		if a == nil {
			return ErrUnknownAgent
		}
		switch st {
		case domain.AgentAvailable, domain.AgentPaused, domain.AgentOffline:
		default:
			return errors.New("status must be available, paused or offline")
		}
		if st == domain.AgentAvailable {
			a.MissedInRow = 0
		}
		if a.Status == domain.AgentBusy || a.Status == domain.AgentRinging {
			if a.Status == domain.AgentRinging && st != domain.AgentAvailable {
				c.abandonRing(a)
			} else {
				a.pendingStatus, a.pendingReason = st, reason
				c.publish(domain.EvAgentUpdated, a.AgentState)
				return nil
			}
		}
		c.setAgentStatus(a, st, reason)
		c.dispatch()
		return nil
	})
}

// abandonRing stops ringing an agent who just went unavailable.
func (c *Core) abandonRing(a *agentRT) {
	call := c.calls[a.CallID]
	a.CallID = ""
	if call == nil {
		return
	}
	for chanID, uid := range call.ringing {
		if uid == a.UserID {
			delete(call.ringing, chanID)
			delete(c.legs, chanID)
			c.hangup(chanID, "normal")
		}
	}
	if call.State == domain.CallRinging && len(call.ringing) == 0 {
		call.State = domain.CallQueued
		c.changed(call)
	}
}
