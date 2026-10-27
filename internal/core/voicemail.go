package core

import (
	"log/slog"
	"time"

	"callcenter-middleware/internal/domain"
)

// startVoicemail plays the voicemail prompt then records the caller.
func (c *Core) startVoicemail(call *liveCall) {
	c.stopPlayback(call)
	call.State = domain.CallVoicemail
	c.changed(call)
	c.timeline(call, "voicemail", "")
	c.play(call, c.cfg.VoicemailMedia, func() {
		if call.ended {
			return
		}
		c.recordVoicemail(call, "vm-"+call.ID, true)
	})
}

func (c *Core) recordVoicemail(call *liveCall, name string, beep bool) {
	c.recordings[name] = recordingRef{callID: call.ID, number: call.CallerNumber,
		name: call.CallerName, queueID: call.QueueID, beep: beep, started: c.now()}
	ctx, cancel := c.opCtx()
	defer cancel()
	if err := c.Switch.RecordChannel(ctx, call.callerChan, name, c.cfg.VoicemailMaxSec, beep); err != nil {
		slog.Error("Voicemail record failed", "call_id", call.ID, "err", err)
		delete(c.recordings, name)
		c.hangup(call.callerChan, "normal")
	}
}

// onRecordingFinished stores a voicemail once Asterisk finalizes the file.
func (c *Core) onRecordingFinished(ev domain.TelephonyEvent) {
	ref, ok := c.recordings[ev.RecordingName]
	if !ok {
		return
	}
	delete(c.recordings, ev.RecordingName)
	call := c.calls[ref.callID]
	instant := c.now().Sub(ref.started) < 2*time.Second
	if call != nil && call.State == domain.CallVoicemail && ref.beep && instant &&
		(ev.Type == domain.EventRecordingFailed || ev.DurationSec < 1) {
		// Recording aborted at once (e.g. missing beep sound): retry silently.
		slog.Warn("Voicemail recording aborted immediately, retrying without beep", "call_id", call.ID)
		c.recordVoicemail(call, ev.RecordingName+"-nb", false)
		return
	}
	if ev.Type == domain.EventRecordingFailed || ev.DurationSec < 1 {
		slog.Info("Voicemail discarded", "recording", ev.RecordingName, "duration", ev.DurationSec)
	} else {
		vm := &domain.Voicemail{CallID: ref.callID, CallerNumber: ref.number, CallerName: ref.name,
			QueueID: ref.queueID, Recording: ev.RecordingName, DurationSec: ev.DurationSec}
		ctx, cancel := c.opCtx()
		err := c.Store.CreateVoicemail(ctx, vm)
		cancel()
		if err != nil {
			slog.Error("Store voicemail failed", "err", err)
		} else {
			c.publish(domain.EvVoicemail, vm)
		}
	}
	if call != nil && call.State == domain.CallVoicemail {
		call.Outcome = domain.OutcomeVoicemail
		c.hangup(call.callerChan, "normal")
		c.finish(call, "middleware")
	}
}
