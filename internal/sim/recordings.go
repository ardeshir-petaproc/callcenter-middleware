package sim

import (
	"bytes"
	"encoding/binary"
	"math"
	"net/http"
	"strconv"
	"time"
)

func (s *Sim) recordChannel(w http.ResponseWriter, r *http.Request) int {
	ch := s.channels[r.PathValue("id")]
	if ch == nil {
		return http.StatusNotFound
	}
	name := r.URL.Query().Get("name")
	maxSec, _ := strconv.Atoi(r.URL.Query().Get("maxDurationSeconds"))
	s.recordings[name] = &recording{Name: name, Channel: ch.ID, Started: time.Now(), MaxSec: maxSec}
	ch.Recording = name
	// The caller leaves a short message and hangs up.
	s.after(ch, s.between(4, 12), func() { s.hangup(ch, 16, "voicemail") })
	return writeObj(w, map[string]any{"name": name, "state": "recording", "format": "wav"})
}

func (s *Sim) recordBridge(w http.ResponseWriter, r *http.Request) int {
	id := r.PathValue("id")
	if _, ok := s.bridges[id]; !ok {
		return http.StatusNotFound
	}
	name := r.URL.Query().Get("name")
	s.recordings[name] = &recording{Name: name, Channel: "bridge:" + id, Started: time.Now()}
	return writeObj(w, map[string]any{"name": name, "state": "recording", "format": "wav"})
}

func (s *Sim) stopRecording(w http.ResponseWriter, r *http.Request) int {
	name := r.PathValue("name")
	if _, ok := s.recordings[name]; !ok {
		return http.StatusNotFound
	}
	s.finishRecording(name)
	return http.StatusNoContent
}

func (s *Sim) finishBridgeRecordings(bridgeID string) {
	for name, rec := range s.recordings {
		if rec.Channel == "bridge:"+bridgeID {
			s.finishRecording(name)
		}
	}
}

// finishRecording stores the recording and emits RecordingFinished.
func (s *Sim) finishRecording(name string) {
	rec := s.recordings[name]
	if rec == nil || rec.done {
		return
	}
	rec.done = true
	delete(s.recordings, name)
	dur := int(time.Since(rec.Started).Seconds() * s.opts.Speed)
	if rec.MaxSec > 0 && dur > rec.MaxSec {
		dur = rec.MaxSec
	}
	s.stored[name] = dur
	s.emit("RecordingFinished", map[string]any{"recording": map[string]any{
		"name": name, "state": "done", "format": "wav", "duration": dur, "target_uri": rec.Channel}})
}

// recordingFile serves a synthetic WAV (a soft two-tone chime) for a stored recording.
func (s *Sim) recordingFile(w http.ResponseWriter, r *http.Request) {
	if !s.authorized(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	s.mu.Lock()
	dur, ok := s.stored[r.PathValue("name")]
	s.mu.Unlock()
	if !ok {
		http.Error(w, `{"message":"Recording not found"}`, http.StatusNotFound)
		return
	}
	dur = max(1, min(dur, 20))
	w.Header().Set("Content-Type", "audio/wav")
	_, _ = w.Write(toneWAV(dur))
}

func toneWAV(seconds int) []byte {
	const rate = 8000
	n := rate * seconds
	var b bytes.Buffer
	b.WriteString("RIFF")
	_ = binary.Write(&b, binary.LittleEndian, uint32(36+n*2))
	b.WriteString("WAVEfmt ")
	_ = binary.Write(&b, binary.LittleEndian, []any{uint32(16), uint16(1), uint16(1), uint32(rate),
		uint32(rate * 2), uint16(2), uint16(16)})
	b.WriteString("data")
	_ = binary.Write(&b, binary.LittleEndian, uint32(n*2))
	for i := 0; i < n; i++ {
		t := float64(i) / rate
		freq := 440.0
		if int(t*2)%2 == 1 {
			freq = 660
		}
		v := int16(3000 * math.Sin(2*math.Pi*freq*t))
		_ = binary.Write(&b, binary.LittleEndian, v)
	}
	return b.Bytes()
}

func (s *Sim) deleteRecording(w http.ResponseWriter, r *http.Request) {
	if !s.authorized(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	s.mu.Lock()
	_, ok := s.stored[r.PathValue("name")]
	delete(s.stored, r.PathValue("name"))
	s.mu.Unlock()
	if !ok {
		http.Error(w, `{"message":"Recording not found"}`, http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
