package core

import (
	"fmt"
	"log/slog"
	"sort"
	"time"

	"callcenter-middleware/internal/domain"
)

// enqueue places a caller in a native ACD queue with music on hold.
func (c *Core) enqueue(call *liveCall, q *domain.Queue) {
	c.stopPlayback(call)
	if fb, why := c.departmentGate(call, q); why != "" {
		c.timeline(call, "department_"+why, q.Name)
		c.execute(call, fb)
		return
	}
	call.queue, call.QueueID = q, q.ID
	call.queueEnter = c.now()
	if call.QueuedAt == nil {
		call.QueuedAt = ptr(c.now())
	}
	call.tried = map[int64]bool{}
	call.State = domain.CallQueued
	c.timeline(call, "queued", q.Name)
	c.screenPop(call, q.Name, "queued")
	c.changed(call)

	if q.LeaveWhenEmpty && c.loggedIn(q) == 0 && call.targetAgent == 0 {
		c.timeline(call, "queue_empty", q.Name)
		c.overflow(call)
		return
	}
	join := func() { c.play(call, q.JoinMedia, func() { c.startMOH(call) }) }
	if q.Record && q.RecordNotice != "" {
		// Agents are rung only after the caller has heard the whole notice.
		call.announcing = true
		c.timeline(call, "recording_notice", q.RecordNotice)
		c.play(call, q.RecordNotice, func() { call.announcing = false; join(); c.dispatch() })
	} else {
		join()
	}
	c.dispatch()
}

func (c *Core) startMOH(call *liveCall) {
	if call.State != domain.CallQueued && call.State != domain.CallRinging &&
		call.State != domain.CallHold && call.State != domain.CallDialing {
		return
	}
	class := ""
	if call.queue != nil {
		class = call.queue.MOHClass
	}
	ctx, cancel := c.opCtx()
	defer cancel()
	_ = c.Switch.StartMOH(ctx, call.callerChan, class)
}

func (c *Core) stopMOH(call *liveCall) {
	ctx, cancel := c.opCtx()
	defer cancel()
	_ = c.Switch.StopMOH(ctx, call.callerChan)
}

// leaveQueue stops ringing agents and music before the caller moves on.
func (c *Core) leaveQueue(call *liveCall) {
	c.stopPlayback(call)
	c.cancelRinging(call, "")
	c.stopMOH(call)
	call.targetAgent = 0
}

func (c *Core) cancelRinging(call *liveCall, except string) {
	for chanID, uid := range call.ringing {
		if chanID == except {
			continue
		}
		delete(call.ringing, chanID)
		delete(c.legs, chanID)
		c.hangup(chanID, "normal")
		if a := c.agents[uid]; a != nil && a.CallID == call.ID && a.Status == domain.AgentRinging {
			c.setAgentStatus(a, domain.AgentAvailable, "")
		}
	}
}

// waitingCalls returns queued calls in service order.
func (c *Core) waitingCalls() []*liveCall {
	var out []*liveCall
	for _, call := range c.calls {
		if (call.State == domain.CallQueued || call.State == domain.CallRinging) && call.queue != nil {
			out = append(out, call)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		a, b := out[i], out[j]
		if a.queue.Priority != b.queue.Priority {
			return a.queue.Priority > b.queue.Priority
		}
		if a.priority != b.priority {
			return a.priority > b.priority
		}
		return a.queueEnter.Before(b.queueEnter)
	})
	return out
}

// dispatch offers waiting calls to available agents.
func (c *Core) dispatch() {
	for _, call := range c.waitingCalls() {
		if len(call.ringing) > 0 || call.announcing {
			continue
		}
		agents := c.selectAgents(call)
		for _, a := range agents {
			c.ring(call, a)
		}
	}
}

// ring originates a call to an agent endpoint that enters Stasis on answer.
func (c *Core) ring(call *liveCall, a *agentRT) {
	q := call.queue
	timeout := c.cfg.DefaultRingTimeout
	if q != nil && q.RingTimeoutSec > 0 {
		timeout = q.RingTimeoutSec
	}
	chanID := c.nextChan("agent", call.ID)
	l := &leg{chanID: chanID, call: call, role: roleAgent, agentID: a.UserID, since: c.now()}
	c.legs[chanID] = l
	call.ringing[chanID] = a.UserID
	call.tried[a.UserID] = true
	a.CallID = call.ID
	c.setAgentStatus(a, domain.AgentRinging, "")

	callerID := call.CallerNumber
	if call.CallerName != "" {
		callerID = fmt.Sprintf("\"%s\" <%s>", call.CallerName, call.CallerNumber)
	}
	ctx, cancel := c.opCtx()
	err := c.Switch.Originate(ctx, domain.OriginateRequest{
		Endpoint: a.user.DialEndpoint(), ChannelID: chanID, CallerID: callerID, TimeoutSec: timeout,
		AppArgs: roleAgent + "," + call.ID,
	})
	cancel()
	if err != nil {
		slog.Warn("Ring agent failed", "call_id", call.ID, "agent", a.Username, "err", err)
		delete(c.legs, chanID)
		delete(call.ringing, chanID)
		a.CallID = ""
		a.retryAfter = c.now().Add(5 * time.Second)
		c.setAgentStatus(a, domain.AgentAvailable, "")
		return
	}
	c.timeline(call, "ringing", a.Username)
	c.publish(call.TenantID, domain.EvScreenPop, map[string]any{"user_id": a.UserID, "call": c.view(call)})
	if call.State == domain.CallQueued {
		call.State = domain.CallRinging
	}
	c.changed(call)
}
