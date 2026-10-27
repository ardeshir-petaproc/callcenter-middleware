package core

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"callcenter-middleware/internal/domain"
)

// opCtx bounds a single ARI or DB operation executed on the loop.
func (c *Core) opCtx() (context.Context, context.CancelFunc) {
	return context.WithTimeout(c.ctx, 5*time.Second)
}

func (c *Core) nextChan(prefix, callID string) string {
	c.seq++
	short := callID
	if len(short) > 8 {
		short = short[:8]
	}
	return fmt.Sprintf("%s-%s-%d", prefix, short, c.seq)
}

// persist saves the call record.
func (c *Core) persist(call *liveCall) {
	ctx, cancel := c.opCtx()
	defer cancel()
	if err := c.Store.SaveCall(ctx, &call.Call); err != nil {
		slog.Error("Persist call failed", "call_id", call.ID, "err", err)
	}
}

// timeline records an event in the call history and logs it.
func (c *Core) timeline(call *liveCall, typ, detail string) {
	ctx, cancel := c.opCtx()
	defer cancel()
	_ = c.Store.AddCallEvent(ctx, domain.CallEvent{CallID: call.ID, Type: typ, Detail: detail, At: c.now().UTC()})
	slog.Info("Call event", "call_id", call.ID, "type", typ, "detail", detail)
}

func (c *Core) publish(typ string, data any) {
	if c.Publisher != nil {
		c.Publisher.Publish(domain.Event{Type: typ, Data: data, At: c.now().UTC()})
	}
}

// changed persists and broadcasts a call update.
func (c *Core) changed(call *liveCall) {
	c.persist(call)
	c.publish(domain.EvCallUpdated, c.view(call))
}

func (c *Core) view(call *liveCall) CallView {
	v := CallView{Call: call.Call, RingingAgents: []int64{}}
	for _, uid := range call.ringing {
		v.RingingAgents = append(v.RingingAgents, uid)
	}
	now := c.now()
	v.DurationSec = int(now.Sub(call.StartedAt).Seconds())
	if (call.State == domain.CallQueued || call.State == domain.CallRinging) && !call.queueEnter.IsZero() {
		v.WaitingSec = int(now.Sub(call.queueEnter).Seconds())
	}
	for _, l := range c.legs {
		if l.call == call && l.role == roleSupervisor {
			v.Monitors++
		}
	}
	return v
}

func (c *Core) setState(call *liveCall, st domain.CallState) {
	if call.State == st {
		return
	}
	call.State = st
	c.changed(call)
}

func (c *Core) hangup(chanID, reason string) {
	if chanID == "" {
		return
	}
	ctx, cancel := c.opCtx()
	defer cancel()
	if err := c.Switch.HangupChannel(ctx, chanID, reason); err != nil {
		slog.Debug("Hangup failed", "channel_id", chanID, "err", err)
	}
}

func (c *Core) destroyBridge(id string) {
	if id == "" {
		return
	}
	ctx, cancel := c.opCtx()
	defer cancel()
	_ = c.Switch.DestroyBridge(ctx, id)
}

// play starts media on the caller channel and runs then when it finishes.
func (c *Core) play(call *liveCall, media string, then func()) {
	if media == "" {
		if then != nil {
			then()
		}
		return
	}
	c.stopPlayback(call)
	call.playID = c.nextChan("play", call.ID)
	call.afterPlay = then
	c.playbacks[call.playID] = call.ID
	ctx, cancel := c.opCtx()
	defer cancel()
	if err := c.Switch.PlayMedia(ctx, call.callerChan, media, call.playID); err != nil {
		slog.Warn("Play media failed", "call_id", call.ID, "media", media, "err", err)
		delete(c.playbacks, call.playID)
		call.playID, call.afterPlay = "", nil
		if then != nil {
			then()
		}
	}
}

func (c *Core) stopPlayback(call *liveCall) {
	if call.playID == "" {
		return
	}
	ctx, cancel := c.opCtx()
	defer cancel()
	_ = c.Switch.StopPlayback(ctx, call.playID)
	delete(c.playbacks, call.playID)
	call.playID, call.afterPlay = "", nil
}

func (c *Core) onPlaybackFinished(playID string) {
	callID, ok := c.playbacks[playID]
	if !ok {
		return
	}
	delete(c.playbacks, playID)
	call := c.calls[callID]
	if call == nil || call.playID != playID {
		return
	}
	then := call.afterPlay
	call.playID, call.afterPlay = "", nil
	if then != nil {
		then()
	}
}

func secs(d time.Duration) int { return int(d.Round(time.Second).Seconds()) }

func ptr(t time.Time) *time.Time {
	u := t.UTC()
	return &u
}
