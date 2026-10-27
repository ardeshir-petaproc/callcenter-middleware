package core

import (
	"callcenter-middleware/internal/domain"
	"fmt"
	"time"
)

func (c *Core) tickQueued(call *liveCall, now time.Time) {
	q := call.queue
	if q == nil || q.MaxWaitSec <= 0 {
		return
	}
	if now.Sub(call.queueEnter) >= time.Duration(q.MaxWaitSec)*time.Second {
		c.timeline(call, "max_wait", fmt.Sprintf("%ds", q.MaxWaitSec))
		c.overflow(call)
	}
}

// overflow applies the queue overflow action, defaulting to voicemail.
func (c *Core) overflow(call *liveCall) {
	c.leaveQueue(call)
	action := domain.Action{Type: domain.ActionVoicemail}
	if call.queue != nil && call.queue.Overflow.Type != domain.ActionNone {
		action = call.queue.Overflow
	}
	if action.Type == domain.ActionQueue && call.queue != nil && c.cache.findQueue(action.Target) == call.queue {
		action = domain.Action{Type: domain.ActionVoicemail}
	}
	if c.Metrics != nil && call.queue != nil {
		c.Metrics.RecordHandoff(call.queue.Name, "overflow")
	}
	c.execute(call, action)
}

// directToAgent rings one specific agent, keeping the call's queue for fallback.
func (c *Core) directToAgent(call *liveCall, userID int64) {
	q := call.queue
	if q == nil {
		q = c.cache.findQueue(c.defaultAction().Target)
	}
	if q == nil {
		q = &domain.Queue{Name: "direct", Strategy: domain.StrategyRingAll, Enabled: true}
	}
	call.targetAgent = userID
	c.enqueue(call, q)
}
