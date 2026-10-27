package core

import (
	"log/slog"
	"time"

	"callcenter-middleware/internal/domain"
)

// startAI connects the caller to an OpenAI realtime voice agent through an
// ExternalMedia RTP channel. Any failure falls back to a human queue.
func (c *Core) startAI(call *liveCall, profileID int64) {
	c.stopPlayback(call)
	profile := c.cache.profiles[profileID]
	if profile == nil {
		for _, p := range c.cache.profiles {
			if p.Enabled {
				profile = p
				break
			}
		}
	}
	if profile == nil || !profile.Enabled || c.AI == nil {
		slog.Warn("AI agent unavailable, falling back to queue", "call_id", call.ID, "profile", profileID)
		c.timeline(call, "ai_unavailable", "")
		c.execute(call, c.defaultAction())
		return
	}
	call.AIProfileID = profile.ID
	call.aiHandoff = profile.HandoffQueueID
	host, err := c.AI.Start(AISpec{CallID: call.ID, CallerNumber: call.CallerNumber,
		CallerName: call.CallerName, Profile: *profile}, c)
	if err != nil {
		c.aiFallback(call, "start: "+err.Error())
		return
	}
	mediaChan := "media-" + call.ID
	call.mediaChan = mediaChan
	c.legs[mediaChan] = &leg{chanID: mediaChan, call: call, role: roleMedia, since: c.now()}
	if err := c.bridge(call, call.callerChan); err != nil {
		c.aiFallback(call, "bridge: "+err.Error())
		return
	}
	ctx, cancel := c.opCtx()
	err = c.Switch.ExternalMedia(ctx, domain.ExternalMediaRequest{ChannelID: mediaChan, ExternalHost: host, Format: "ulaw"})
	cancel()
	if err != nil {
		c.aiFallback(call, "external media: "+err.Error())
		return
	}
	call.State = domain.CallAI
	call.aiStart = c.now()
	c.timeline(call, "ai_started", profile.Name)
	c.changed(call)
}

func (c *Core) onMediaReady(l *leg) {
	call := l.call
	if call.bridgeID == "" {
		return
	}
	ctx, cancel := c.opCtx()
	defer cancel()
	if err := c.Switch.AddToBridge(ctx, call.bridgeID, l.chanID); err != nil {
		c.aiFallback(call, "add media: "+err.Error())
	}
}

// stopAI tears down the AI side of a call, leaving the caller in Stasis.
func (c *Core) stopAI(call *liveCall) {
	if c.AI != nil {
		c.AI.Stop(call.ID)
	}
	if call.mediaChan != "" {
		delete(c.legs, call.mediaChan)
		c.hangup(call.mediaChan, "normal")
		call.mediaChan = ""
	}
	if call.bridgeID != "" {
		ctx, cancel := c.opCtx()
		_ = c.Switch.RemoveFromBridge(ctx, call.bridgeID, call.callerChan)
		cancel()
		c.destroyBridge(call.bridgeID)
		call.bridgeID = ""
	}
}

func (c *Core) aiFallback(call *liveCall, reason string) {
	slog.Warn("AI session failed, handing off to humans", "call_id", call.ID, "reason", reason)
	c.timeline(call, "ai_failed", reason)
	c.stopAI(call)
	c.aiHandoffQueue(call, "")
}

func (c *Core) aiHandoffQueue(call *liveCall, queueRef string) {
	var q *domain.Queue
	if queueRef != "" {
		q = c.cache.findQueue(queueRef)
	}
	if q == nil && call.aiHandoff > 0 {
		q = c.cache.queues[call.aiHandoff]
	}
	if q == nil || !q.Enabled {
		c.execute(call, c.defaultAction())
		return
	}
	c.execute(call, domain.Action{Type: domain.ActionQueue, Target: itoa(q.ID)})
}

func (c *Core) tickAI(call *liveCall, now time.Time) {
	p := c.cache.profiles[call.AIProfileID]
	if p == nil || p.MaxDurationSec <= 0 || call.aiStart.IsZero() {
		return
	}
	if now.Sub(call.aiStart) > time.Duration(p.MaxDurationSec)*time.Second {
		c.timeline(call, "ai_max_duration", "")
		c.stopAI(call)
		c.aiHandoffQueue(call, "")
	}
}
