package sim

import (
	"net/http"
	"strconv"
	"strings"
	"time"
)

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
