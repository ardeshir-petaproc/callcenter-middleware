package sim

import (
	"fmt"
	"net/http"
	"time"
)

// Inbound creates a caller channel entering Stasis. Must hold s.mu.
func (s *Sim) Inbound(c SimCall) string {
	if c.Caller == "" {
		c.Caller = fmt.Sprintf("+1555%07d", s.rng.IntN(10000000))
	}
	if c.DNIS == "" {
		c.DNIS = s.opts.DNIS[s.rng.IntN(len(s.opts.DNIS))]
	}
	if c.Patience <= 0 {
		c.Patience = 30 + s.rng.Float64()*150
	}
	if c.TalkSec <= 0 {
		c.TalkSec = 20 + s.rng.Float64()*100
	}
	id := s.nextID("in")
	ch := &channel{ID: id, Name: "PJSIP/trunk-" + id, State: "Ring", Number: c.Caller, CallerName: c.Name,
		Exten: c.DNIS, Role: "caller", InStasis: true, Digits: c.Digits, Patience: c.Patience, Talk: c.TalkSec,
		created: time.Now()}
	s.channels[id] = ch
	s.stats["inbound"]++
	s.emit("StasisStart", map[string]any{"channel": s.channelJSON(ch), "args": []string{}})
	// A caller that is never answered gives up.
	s.after(ch, s.d(30), func() {
		if ch.State != "Up" {
			s.hangup(ch, 16, "caller")
		}
	})
	return id
}

// hangup destroys a channel and emits the matching ARI events. Must hold s.mu.
func (s *Sim) hangup(ch *channel, cause int, by string) {
	if _, ok := s.channels[ch.ID]; !ok {
		return
	}
	for _, t := range ch.timers {
		t.Stop()
	}
	delete(s.channels, ch.ID)
	if ch.Bridge != "" {
		s.leaveBridge(ch.Bridge, ch.ID)
	}
	for name, rec := range s.recordings {
		if rec.Channel == ch.ID {
			s.finishRecording(name)
		}
	}
	ch.State = "Down"
	if ch.InStasis {
		s.emit("ChannelHangupRequest", map[string]any{"channel": s.channelJSON(ch), "cause": cause})
		s.emit("StasisEnd", map[string]any{"channel": s.channelJSON(ch)})
	}
	s.emit("ChannelDestroyed", map[string]any{"channel": s.channelJSON(ch), "cause": cause,
		"cause_txt": causeText(cause)})
	s.stats["hangup_"+by]++
}

func causeText(c int) string {
	switch c {
	case 16:
		return "Normal Clearing"
	case 17:
		return "User busy"
	case 19:
		return "No user responding"
	case 21:
		return "Call Rejected"
	}
	return "Unknown"
}

func (s *Sim) answer(w http.ResponseWriter, r *http.Request) int {
	ch := s.channels[r.PathValue("id")]
	if ch == nil {
		return http.StatusNotFound
	}
	ch.State = "Up"
	return http.StatusNoContent
}

func (s *Sim) hangupREST(w http.ResponseWriter, r *http.Request) int {
	ch := s.channels[r.PathValue("id")]
	if ch == nil {
		return http.StatusNotFound
	}
	s.hangup(ch, 16, "app")
	return http.StatusNoContent
}

func (s *Sim) continueDialplan(w http.ResponseWriter, r *http.Request) int {
	ch := s.channels[r.PathValue("id")]
	if ch == nil {
		return http.StatusNotFound
	}
	for _, t := range ch.timers {
		t.Stop()
	}
	delete(s.channels, ch.ID)
	s.emit("StasisEnd", map[string]any{"channel": s.channelJSON(ch)})
	s.stats["continued_"+r.URL.Query().Get("extension")]++
	return http.StatusNoContent
}

func (s *Sim) moh(on bool) func(w http.ResponseWriter, r *http.Request) int {
	return func(w http.ResponseWriter, r *http.Request) int {
		ch := s.channels[r.PathValue("id")]
		if ch == nil {
			return http.StatusNotFound
		}
		if on && !ch.MOH && ch.Role == "caller" && ch.Bridge == "" {
			// Waiting callers abandon once their patience runs out.
			s.after(ch, s.d(ch.Patience), func() {
				if ch.MOH && ch.Bridge == "" {
					s.hangup(ch, 16, "abandon")
				}
			})
		}
		ch.MOH = on
		return http.StatusNoContent
	}
}
