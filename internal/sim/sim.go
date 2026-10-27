// Package sim is a lightweight Asterisk ARI simulator. It implements the
// subset of the ARI REST and WebSocket API the middleware uses and drives
// synthetic callers and agent phones, so the whole platform can be demoed
// and tested end-to-end without a PBX.
package sim

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"net/http"
	"sync"
	"time"
)

// Options tune simulated behaviour.
type Options struct {
	App            string
	Username       string
	Password       string
	Speed          float64 // >1 runs faster than real time
	AnswerProb     float64 // probability an agent phone answers
	TrunkAnswer    float64 // probability a dialed outside number answers
	CallsPerMinute float64 // automatic inbound call rate (0 = manual only)
	DNIS           []string
	Digits         string // digits callers pick from ("" = never press)
}

type channel struct {
	ID, Name, State    string
	Number, CallerName string
	Exten              string
	Args               []string
	InStasis           bool
	Bridge             string
	Role               string // caller, phone, trunk, media, snoop
	MOH                bool
	Pressed            int
	Recording          string
	Digits             string
	Patience           float64
	Talk               float64
	created            time.Time
	timers             []*time.Timer
}

type recording struct {
	Name    string
	Channel string
	Started time.Time
	MaxSec  int
	done    bool
}

// Sim is the simulator state.
type Sim struct {
	opts Options
	rng  *rand.Rand

	mu         sync.Mutex
	clients    map[*client]bool
	channels   map[string]*channel
	bridges    map[string][]string
	playbacks  map[string]*time.Timer
	recordings map[string]*recording
	stored     map[string]int // recording name -> seconds
	seq        int
	stats      map[string]int
}

// New creates a simulator.
func New(o Options) *Sim {
	if o.Speed <= 0 {
		o.Speed = 1
	}
	if o.App == "" {
		o.App = "callcenter-app"
	}
	if o.TrunkAnswer <= 0 {
		o.TrunkAnswer = 0.85
	}
	if len(o.DNIS) == 0 {
		o.DNIS = []string{"100"}
	}
	return &Sim{
		opts: o, rng: rand.New(rand.NewPCG(uint64(time.Now().UnixNano()), 1)),
		clients: map[*client]bool{}, channels: map[string]*channel{}, bridges: map[string][]string{},
		playbacks: map[string]*time.Timer{}, recordings: map[string]*recording{}, stored: map[string]int{},
		stats: map[string]int{},
	}
}

// d scales a simulated duration by the speed factor.
func (s *Sim) d(sec float64) time.Duration {
	return time.Duration(sec / s.opts.Speed * float64(time.Second))
}

// between returns a random duration in [a,b] seconds, scaled.
func (s *Sim) between(a, b float64) time.Duration { return s.d(a + s.rng.Float64()*(b-a)) }

func (s *Sim) nextID(prefix string) string {
	s.seq++
	return fmt.Sprintf("%s-%d-%d", prefix, time.Now().Unix()%100000, s.seq)
}

// after schedules fn under the simulator lock, tied to a channel's lifetime.
func (s *Sim) after(ch *channel, d time.Duration, fn func()) {
	t := time.AfterFunc(d, func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		if ch != nil {
			if _, alive := s.channels[ch.ID]; !alive {
				return
			}
		}
		fn()
	})
	if ch != nil {
		ch.timers = append(ch.timers, t)
	}
}

func (s *Sim) channelJSON(ch *channel) map[string]any {
	return map[string]any{
		"id": ch.ID, "name": ch.Name, "state": ch.State,
		"caller":       map[string]string{"number": ch.Number, "name": ch.CallerName},
		"dialplan":     map[string]any{"context": "from-external", "exten": ch.Exten, "priority": 1},
		"creationtime": ch.created.Format(time.RFC3339),
	}
}

// emit sends an ARI event to every connected application client.
func (s *Sim) emit(typ string, fields map[string]any) {
	fields["type"] = typ
	fields["timestamp"] = time.Now().UTC().Format("2006-01-02T15:04:05.000-0700")
	fields["application"] = s.opts.App
	raw, err := json.Marshal(fields)
	if err != nil {
		return
	}
	for c := range s.clients {
		c.send(raw)
	}
	s.stats["events"]++
}

// Handler returns the HTTP handler serving /ari and /sim endpoints.
func (s *Sim) Handler() http.Handler {
	mux := http.NewServeMux()
	s.routes(mux)
	return mux
}

// Stats returns counters for observability.
func (s *Sim) Stats() map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := map[string]any{"channels": len(s.channels), "bridges": len(s.bridges), "clients": len(s.clients)}
	for k, v := range s.stats {
		out[k] = v
	}
	return out
}

func (s *Sim) logf(msg string, args ...any) { slog.Debug(msg, args...) }
