package sim

import (
	"encoding/json"
	"net/http"
	"strings"
)

func (s *Sim) routes(mux *http.ServeMux) {
	ari := func(pattern string, h func(w http.ResponseWriter, r *http.Request) int) {
		mux.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
			if !s.authorized(r) {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			s.mu.Lock()
			status := h(w, r)
			s.mu.Unlock()
			if status != 0 {
				w.WriteHeader(status)
			}
		})
	}
	mux.HandleFunc("GET /ari/events", func(w http.ResponseWriter, r *http.Request) {
		if !s.authorized(r) {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		s.handleEvents(w, r)
	})
	ari("POST /ari/channels", s.originate)
	ari("GET /ari/channels", s.listChannels)
	ari("POST /ari/channels/externalMedia", s.externalMedia)
	ari("POST /ari/channels/{id}/answer", s.answer)
	ari("POST /ari/channels/{id}/play/{pid}", s.play)
	ari("DELETE /ari/playbacks/{id}", s.stopPlayback)
	ari("POST /ari/channels/{id}/continue", s.continueDialplan)
	ari("DELETE /ari/channels/{id}", s.hangupREST)
	ari("POST /ari/channels/{id}/moh", s.moh(true))
	ari("DELETE /ari/channels/{id}/moh", s.moh(false))
	ari("POST /ari/channels/{id}/ring", s.noop)
	ari("DELETE /ari/channels/{id}/ring", s.noop)
	ari("POST /ari/channels/{id}/mute", s.noop)
	ari("DELETE /ari/channels/{id}/mute", s.noop)
	ari("POST /ari/channels/{id}/record", s.recordChannel)
	ari("POST /ari/channels/{id}/snoop", s.snoop)
	ari("POST /ari/bridges/{id}", s.createBridge)
	ari("DELETE /ari/bridges/{id}", s.destroyBridge)
	ari("POST /ari/bridges/{id}/addChannel", s.addChannel)
	ari("POST /ari/bridges/{id}/removeChannel", s.removeChannel)
	ari("POST /ari/bridges/{id}/record", s.recordBridge)
	ari("POST /ari/recordings/live/{name}/stop", s.stopRecording)
	mux.HandleFunc("GET /ari/recordings/stored/{name}/file", s.recordingFile)
	mux.HandleFunc("DELETE /ari/recordings/stored/{name}", s.deleteRecording)

	mux.HandleFunc("POST /sim/call", s.handleSimCall)
	mux.HandleFunc("GET /sim/state", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(s.Stats())
	})
}

func (s *Sim) listChannels(w http.ResponseWriter, r *http.Request) int {
	out := []map[string]any{}
	for _, ch := range s.channels {
		out = append(out, s.channelJSON(ch))
	}
	return writeObj(w, out)
}

func (s *Sim) noop(w http.ResponseWriter, r *http.Request) int {
	if s.channels[r.PathValue("id")] == nil {
		return http.StatusNotFound
	}
	return http.StatusNoContent
}

func writeObj(w http.ResponseWriter, v any) int {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(v)
	return 0
}

// SimCall describes a scripted inbound call for POST /sim/call.
type SimCall struct {
	Caller   string  `json:"caller"`
	Name     string  `json:"name"`
	DNIS     string  `json:"dnis"`
	Digits   string  `json:"digits"`
	Patience float64 `json:"patience_sec"`
	TalkSec  float64 `json:"talk_sec"`
}

func (s *Sim) handleSimCall(w http.ResponseWriter, r *http.Request) {
	var c SimCall
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&c)
	}
	writeObj(w, map[string]string{"channel_id": s.InjectCall(c)})
}

// InjectCall starts a scripted inbound call and returns its channel id.
func (s *Sim) InjectCall(c SimCall) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.Inbound(c)
}

func endpointKind(endpoint string) string {
	switch {
	case strings.Contains(endpoint, "@"):
		return "trunk"
	default:
		return "phone"
	}
}
