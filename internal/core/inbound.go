package core

import (
	"log/slog"
	"strings"

	"callcenter-middleware/internal/domain"
	"callcenter-middleware/internal/store"

	"github.com/google/uuid"
)

func (c *Core) onStasisStart(ev domain.TelephonyEvent) {
	if l, ok := c.legs[ev.ChannelID]; ok {
		c.onLegAnswered(l)
		return
	}
	if len(ev.Args) > 0 {
		// An originated leg we no longer track (call already finished).
		slog.Info("Hanging up orphan originated channel", "channel_id", ev.ChannelID, "args", ev.Args)
		c.hangup(ev.ChannelID, "normal")
		return
	}
	if strings.HasPrefix(ev.ChannelName, "UnicastRTP/") || strings.HasPrefix(ev.ChannelName, "Snoop/") {
		return
	}
	c.startInbound(ev)
}

func (c *Core) startInbound(ev domain.TelephonyEvent) {
	call := newLiveCall(uuid.NewString())
	call.Direction = domain.DirectionInbound
	call.CallerNumber, call.CallerName, call.DNIS = ev.CallerNumber, ev.CallerName, ev.DNIS
	call.StartedAt = c.now().UTC()
	call.State = domain.CallIVR
	call.callerChan = ev.ChannelID
	tenant, number := c.cache.tenantFor(ev.DNIS)
	call.TenantID = tenant
	c.calls[call.ID] = call
	c.legs[ev.ChannelID] = &leg{chanID: ev.ChannelID, call: call, role: roleCaller, answered: true, since: c.now()}
	if c.Metrics != nil {
		c.Metrics.IncActiveCalls()
	}
	c.matchContact(call)

	ctx, cancel := c.opCtx()
	err := c.Switch.AnswerChannel(ctx, ev.ChannelID)
	cancel()
	if err != nil {
		slog.Error("Failed to answer channel", "channel_id", ev.ChannelID, "err", err)
	}
	c.timeline(call, "started", "from "+call.CallerNumber+" to "+call.DNIS)

	flow := c.cache.flowFor(tenant, ev.DNIS, number)
	if flow == nil && tenant == DefaultTenant {
		flow = c.legacyFlow()
	} else if flow == nil {
		// A tenant without flows still answers: straight to its first queue.
		flow = &domain.IVRFlow{TenantID: tenant, Name: "default", Enabled: true, TimeoutSec: 5,
			DirectAction: c.defaultAction(tenant)}
	}
	call.flow, call.FlowID = flow, flow.ID
	c.changed(call)

	if sch, ok := c.cache.schedules[flow.ScheduleID]; ok && flow.ScheduleID > 0 && !IsOpen(sch, c.now()) {
		c.timeline(call, "after_hours", sch.Name)
		action := flow.AfterHoursAction
		if action.Type == domain.ActionNone {
			action = domain.Action{Type: domain.ActionHangup, Media: c.cfg.ClosedMedia}
		}
		c.execute(call, action)
		return
	}
	if flow.DirectAction.Type != domain.ActionNone {
		c.execute(call, flow.DirectAction)
		return
	}
	c.playGreeting(call)
}

// legacyFlow reproduces the original env-driven DTMF menu that hands calls
// to Asterisk dialplan queues; used when no flow is configured.
func (c *Core) legacyFlow() *domain.IVRFlow {
	opts := map[string]domain.Action{}
	for digit, ext := range c.cfg.QueueMappings {
		opts[digit] = domain.Action{Type: domain.ActionDialplan, Target: ext}
	}
	return &domain.IVRFlow{
		Name:          "legacy",
		GreetingMedia: c.cfg.GreetingMedia,
		TimeoutSec:    int(c.cfg.IVRTimeout.Seconds()),
		MaxRetries:    c.cfg.MaxRetries,
		Options:       opts,
		TimeoutAction: domain.Action{Type: domain.ActionDialplan, Target: c.cfg.DefaultQueue},
		Enabled:       true,
	}
}

func (c *Core) matchContact(call *liveCall) {
	if call.CallerNumber == "" {
		return
	}
	ctx, cancel := c.opCtx()
	defer cancel()
	if ct := FindContact(store.ForTenant(ctx, call.TenantID), c.Store, call.CallerNumber); ct != nil {
		call.ContactID = ct.ID
		if call.CallerName == "" {
			call.CallerName = ct.Name
		}
		if ct.VIP {
			call.priority = 10
			call.Tags = append(call.Tags, "vip")
		}
	}
}

// onLegAnswered handles a known originated/media/snoop channel entering Stasis.
func (c *Core) onLegAnswered(l *leg) {
	l.answered = true
	switch l.role {
	case roleAgent:
		c.onAgentAnswered(l)
	case roleMedia:
		c.onMediaReady(l)
	case roleSupervisor:
		c.onSupervisorAnswered(l)
	case roleSnoop:
		c.onSnoopReady(l)
	case roleOutAgent:
		c.onOutboundAgentAnswered(l)
	case roleOutCust, roleExternal:
		c.onRemotePartyAnswered(l)
	case roleConsult:
		c.onConsultAnswered(l)
	}
}
