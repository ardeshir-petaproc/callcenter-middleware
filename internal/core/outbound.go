package core

import (
	"errors"
	"log/slog"
	"strings"

	"callcenter-middleware/internal/domain"
	"callcenter-middleware/internal/store"

	"github.com/google/uuid"
)

// ErrAgentBusy is returned when an agent cannot place a call now.
var ErrAgentBusy = errors.New("agent is busy")

func (c *Core) dialString(number string) string {
	return strings.ReplaceAll(c.cfg.OutboundDialTemplate, "{number}", NormalizePhoneKeepPlus(number))
}

// NormalizePhoneKeepPlus strips formatting but keeps a leading '+'.
func NormalizePhoneKeepPlus(n string) string {
	n = strings.TrimSpace(n)
	plus := strings.HasPrefix(n, "+")
	d := NormalizePhone(n)
	if plus {
		return "+" + d
	}
	return d
}

// Dial starts a click-to-call: the agent's phone rings first, then the
// customer is dialed and both are bridged. Returns the new call id.
func (c *Core) Dial(agentID int64, number string, callbackID int64) (string, error) {
	var id string
	err := c.do(func() error {
		a := c.agents[agentID]
		if a == nil {
			return ErrUnknownAgent
		}
		call, err := c.dialLocked(a, number, func(call *liveCall) { call.callbackID = callbackID })
		if call != nil {
			id = call.ID
		}
		return err
	})
	return id, err
}

// dialLocked places an agent-first outbound call; setup customizes the call
// before it is published. Runs on the loop.
func (c *Core) dialLocked(a *agentRT, number string, setup func(*liveCall)) (*liveCall, error) {
	if a.CallID != "" || a.Status == domain.AgentBusy || a.Status == domain.AgentRinging {
		return nil, ErrAgentBusy
	}
	if NormalizePhone(number) == "" {
		return nil, errors.New("invalid number")
	}
	if a.user.DialEndpoint() == "" {
		return nil, errors.New("agent has no extension")
	}
	call := newLiveCall(uuid.NewString())
	call.Direction = domain.DirectionOutbound
	call.CallerNumber = NormalizePhoneKeepPlus(number)
	call.StartedAt = c.now().UTC()
	call.State = domain.CallDialing
	call.AgentID = a.UserID
	call.TenantID = a.TenantID
	if setup != nil {
		setup(call)
	}
	c.matchContact(call)
	c.calls[call.ID] = call
	if c.Metrics != nil {
		c.Metrics.IncActiveCalls()
	}
	a.CallID = call.ID
	c.setAgentStatus(a, domain.AgentBusy, "")

	chanID := c.nextChan("oagent", call.ID)
	c.legs[chanID] = &leg{chanID: chanID, call: call, role: roleOutAgent, agentID: a.UserID, since: c.now()}
	ctx, cancel := c.opCtx()
	err := c.Switch.Originate(ctx, domain.OriginateRequest{
		Endpoint: a.user.DialEndpoint(), ChannelID: chanID, TimeoutSec: 30,
		CallerID: "\"Outbound\" <" + call.CallerNumber + ">", AppArgs: roleOutAgent + "," + call.ID,
	})
	cancel()
	c.timeline(call, "dial", call.CallerNumber+" by "+a.Username)
	if err != nil {
		call.Outcome = domain.OutcomeFailed
		c.finish(call, "middleware")
		return call, err
	}
	c.changed(call)
	c.publish(call.TenantID, domain.EvScreenPop, map[string]any{"user_id": a.UserID, "call": c.view(call)})
	return call, nil
}

func (c *Core) onOutboundAgentAnswered(l *leg) {
	call := l.call
	call.agentChan = l.chanID
	ctx, cancel := c.opCtx()
	_ = c.Switch.StartRinging(ctx, l.chanID)
	cancel()
	c.originateRemote(call, roleOutCust, call.CallerNumber)
}

func (c *Core) originateRemote(call *liveCall, role, number string) {
	chanID := c.nextChan(role, call.ID)
	c.legs[chanID] = &leg{chanID: chanID, call: call, role: role, since: c.now()}
	if role == roleExternal {
		call.extChan = chanID
	}
	ctx, cancel := c.opCtx()
	err := c.Switch.Originate(ctx, domain.OriginateRequest{
		Endpoint: c.dialString(number), ChannelID: chanID, CallerID: c.cfg.OutboundCallerID,
		TimeoutSec: 45, AppArgs: role + "," + call.ID,
	})
	cancel()
	if err != nil {
		slog.Warn("Originate remote party failed", "call_id", call.ID, "err", err)
		l := c.legs[chanID]
		c.onLegGone(l, 0)
	}
}

// onRemotePartyAnswered bridges an outbound customer or external transfer target.
func (c *Core) onRemotePartyAnswered(l *leg) {
	call := l.call
	ctx, cancel := c.opCtx()
	defer cancel()
	if l.role == roleOutCust {
		_ = c.Switch.StopRinging(ctx, call.agentChan)
		call.callerChan = l.chanID
		l.role = roleCaller
		if err := c.bridge(call, call.agentChan, l.chanID); err != nil {
			slog.Error("Bridge outbound call failed", "call_id", call.ID, "err", err)
		}
		call.AnsweredAt = ptr(c.now())
		call.State = domain.CallConnected
		c.timeline(call, "answered", call.CallerNumber)
		if c.Store.GetSetting(store.ForTenant(ctx, call.TenantID), "record_outbound", "false") == "true" {
			c.startRecording(call)
		}
		c.changed(call)
		return
	}
	// External transfer target answered.
	c.stopMOH(call)
	if err := c.bridge(call, call.callerChan, l.chanID); err != nil {
		slog.Error("Bridge external transfer failed", "call_id", call.ID, "err", err)
	}
	call.State = domain.CallConnected
	call.Outcome = domain.OutcomeTransferred
	if call.AnsweredAt == nil {
		call.AnsweredAt = ptr(c.now())
	}
	c.timeline(call, "external_answered", "")
	c.changed(call)
}
