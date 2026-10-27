package ari

import (
	"time"

	"callcenter-middleware/internal/domain"
)

func normalizeEvent(raw *RawEvent) (domain.TelephonyEvent, bool) {
	ev := domain.TelephonyEvent{Timestamp: time.Now()}
	if raw.Channel != nil {
		ev.ChannelID = raw.Channel.ID
		ev.ChannelName = raw.Channel.Name
		ev.CallerNumber = raw.Channel.Caller.Number
		ev.CallerName = raw.Channel.Caller.Name
		ev.DNIS = raw.Channel.Dialplan.Exten
	}
	switch raw.Type {
	case "StasisStart":
		if raw.Channel == nil {
			return ev, false
		}
		ev.Type = domain.EventCallStarted
		ev.Args = raw.Args
	case "PlaybackFinished":
		if raw.Playback == nil {
			return ev, false
		}
		ev.Type = domain.EventPlaybackFinished
		ev.PlaybackID = raw.Playback.ID
	case "ChannelDtmfReceived":
		if raw.Channel == nil {
			return ev, false
		}
		ev.Type = domain.EventDTMFReceived
		ev.Digit = raw.Digit
	case "StasisEnd", "ChannelHangupRequest":
		if raw.Channel == nil {
			return ev, false
		}
		ev.Type = domain.EventCallEnded
	case "ChannelDestroyed":
		if raw.Channel == nil {
			return ev, false
		}
		ev.Type = domain.EventChannelDestroyed
		ev.Cause, ev.CauseText = raw.Cause, raw.CauseTxt
	case "RecordingFinished", "RecordingFailed":
		if raw.Recording == nil {
			return ev, false
		}
		ev.Type = domain.EventRecordingFinished
		if raw.Type == "RecordingFailed" {
			ev.Type = domain.EventRecordingFailed
		}
		ev.RecordingName = raw.Recording.Name
		ev.DurationSec = raw.Recording.Duration
	default:
		return ev, false
	}
	return ev, true
}
