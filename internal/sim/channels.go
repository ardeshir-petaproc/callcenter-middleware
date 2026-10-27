package sim

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
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

// originate places a call to an agent phone or a trunk number.
func (s *Sim) originate(w http.ResponseWriter, r *http.Request) int {
	q := r.URL.Query()
	endpoint := q.Get("endpoint")
	if endpoint == "" {
		return http.StatusBadRequest
	}
	id := q.Get("channelId")
	if id == "" {
		id = s.nextID("out")
	}
	timeout, _ := strconv.Atoi(q.Get("timeout"))
	if timeout <= 0 {
		timeout = 30
	}
	var args []string
	if a := q.Get("appArgs"); a != "" {
		args = strings.Split(a, ",")
	}
	ch := &channel{ID: id, Name: endpoint + "-" + id, State: "Down", Role: endpointKind(endpoint), Args: args,
		Exten: endpoint, Number: strings.TrimPrefix(endpoint, "PJSIP/"), created: time.Now()}
	s.channels[id] = ch
	s.stats["originate"]++

	answers := s.rng.Float64() < s.opts.AnswerProb
	delay := s.between(1, 4)
	if ch.Role == "trunk" {
		answers, delay = s.rng.Float64() < s.opts.TrunkAnswer, s.between(2, 6)
	}
	if len(args) > 0 && args[0] == "supervisor" {
		answers = true
	}
	if answers && delay < s.d(float64(timeout)) {
		s.after(ch, delay, func() { s.enterStasis(ch) })
	} else {
		cause, fail := 19, s.d(float64(timeout))
		if ch.Role == "trunk" {
			cause, fail = 17, s.between(2, 4)
		} else if s.rng.Float64() < 0.3 {
			cause, fail = 21, s.between(2, 5)
		}
		s.after(ch, fail, func() { s.hangup(ch, cause, "noanswer") })
	}
	return writeObj(w, s.channelJSON(ch))
}

func (s *Sim) enterStasis(ch *channel) {
	ch.State, ch.InStasis = "Up", true
	s.emit("StasisStart", map[string]any{"channel": s.channelJSON(ch), "args": ch.Args})
}

func (s *Sim) externalMedia(w http.ResponseWriter, r *http.Request) int {
	q := r.URL.Query()
	id := q.Get("channelId")
	if id == "" {
		id = s.nextID("media")
	}
	ch := &channel{ID: id, Name: "UnicastRTP/" + q.Get("external_host") + "-" + id, State: "Up",
		Role: "media", InStasis: true, created: time.Now()}
	s.channels[id] = ch
	s.emit("StasisStart", map[string]any{"channel": s.channelJSON(ch), "args": []string{}})
	return writeObj(w, s.channelJSON(ch))
}

func (s *Sim) snoop(w http.ResponseWriter, r *http.Request) int {
	if s.channels[r.PathValue("id")] == nil {
		return http.StatusNotFound
	}
	q := r.URL.Query()
	id := q.Get("snoopId")
	if id == "" {
		id = s.nextID("snoop")
	}
	ch := &channel{ID: id, Name: "Snoop/" + r.PathValue("id"), State: "Up", Role: "snoop", InStasis: true,
		Args: strings.Split(q.Get("appArgs"), ","), created: time.Now()}
	s.channels[id] = ch
	s.emit("StasisStart", map[string]any{"channel": s.channelJSON(ch), "args": ch.Args})
	return writeObj(w, s.channelJSON(ch))
}
