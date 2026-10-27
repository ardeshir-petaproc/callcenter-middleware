package sim

import (
	"net/http"
	"strings"
	"time"
)

func mediaSeconds(uri string) float64 {
	switch {
	case strings.Contains(uri, "beep"), strings.Contains(uri, "invalid"):
		return 1.5
	case strings.Contains(uri, "vm"), strings.Contains(uri, "callback"), strings.Contains(uri, "closed"):
		return 3
	}
	return 5
}

func (s *Sim) play(w http.ResponseWriter, r *http.Request) int {
	ch := s.channels[r.PathValue("id")]
	if ch == nil {
		return http.StatusNotFound
	}
	pid := r.PathValue("pid")
	media := r.URL.Query().Get("media")
	pb := map[string]any{"id": pid, "media_uri": media, "target_uri": "channel:" + ch.ID, "state": "playing"}
	s.playbacks[pid] = time.AfterFunc(s.d(mediaSeconds(media)), func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		if _, ok := s.playbacks[pid]; !ok {
			return
		}
		delete(s.playbacks, pid)
		pb["state"] = "done"
		s.emit("PlaybackFinished", map[string]any{"playback": pb})
	})
	s.maybePress(ch, media)
	return writeObj(w, pb)
}

// maybePress makes a menu-listening caller press their next planned digit,
// sometimes barging in during the prompt.
func (s *Sim) maybePress(ch *channel, media string) {
	if ch.Role != "caller" || ch.MOH || ch.Bridge != "" || ch.Recording != "" {
		return
	}
	if strings.Contains(media, "vm") || strings.Contains(media, "callback") || strings.Contains(media, "invalid") {
		return
	}
	if ch.Pressed >= len(ch.Digits) {
		return
	}
	digit := string(ch.Digits[ch.Pressed])
	ch.Pressed++
	s.after(ch, s.between(1.5, 7), func() {
		s.emit("ChannelDtmfReceived", map[string]any{"channel": s.channelJSON(ch), "digit": digit, "duration_ms": 120})
	})
}

func (s *Sim) stopPlayback(w http.ResponseWriter, r *http.Request) int {
	pid := r.PathValue("id")
	t, ok := s.playbacks[pid]
	if !ok {
		return http.StatusNotFound
	}
	t.Stop()
	delete(s.playbacks, pid)
	s.emit("PlaybackFinished", map[string]any{"playback": map[string]any{"id": pid, "state": "done"}})
	return http.StatusNoContent
}

func (s *Sim) createBridge(w http.ResponseWriter, r *http.Request) int {
	id := r.PathValue("id")
	if _, ok := s.bridges[id]; !ok {
		s.bridges[id] = []string{}
	}
	return writeObj(w, map[string]any{"id": id, "bridge_type": "mixing", "channels": s.bridges[id]})
}

func (s *Sim) destroyBridge(w http.ResponseWriter, r *http.Request) int {
	id := r.PathValue("id")
	members, ok := s.bridges[id]
	if !ok {
		return http.StatusNotFound
	}
	for _, cid := range members {
		if ch := s.channels[cid]; ch != nil {
			ch.Bridge = ""
		}
	}
	delete(s.bridges, id)
	s.finishBridgeRecordings(id)
	return http.StatusNoContent
}

func (s *Sim) addChannel(w http.ResponseWriter, r *http.Request) int {
	id := r.PathValue("id")
	if _, ok := s.bridges[id]; !ok {
		return http.StatusNotFound
	}
	for _, cid := range strings.Split(r.URL.Query().Get("channel"), ",") {
		ch := s.channels[cid]
		if ch == nil {
			return http.StatusUnprocessableEntity
		}
		if ch.Bridge != "" && ch.Bridge != id {
			s.leaveBridge(ch.Bridge, cid)
		}
		ch.Bridge, ch.MOH = id, false
		s.bridges[id] = append(s.bridges[id], cid)
	}
	s.scheduleTalk(id)
	return http.StatusNoContent
}

// scheduleTalk ends a conversation after a while: the external party (caller
// or dialed customer) hangs up once it is bridged with someone.
func (s *Sim) scheduleTalk(bridgeID string) {
	var party, other *channel
	for _, cid := range s.bridges[bridgeID] {
		ch := s.channels[cid]
		if ch == nil {
			continue
		}
		switch ch.Role {
		case "caller", "trunk":
			party = ch
		default:
			other = ch
		}
	}
	if party == nil || other == nil {
		return
	}
	talk := party.Talk
	if talk <= 0 {
		talk = 20 + s.rng.Float64()*80
	}
	if other.Role == "media" {
		talk = 15 + s.rng.Float64()*30
	}
	bridge := bridgeID
	s.after(party, s.d(talk), func() {
		if party.Bridge == bridge {
			s.hangup(party, 16, "talk_done")
		}
	})
}

func (s *Sim) leaveBridge(bridgeID, chanID string) {
	members := s.bridges[bridgeID]
	for i, cid := range members {
		if cid == chanID {
			s.bridges[bridgeID] = append(members[:i:i], members[i+1:]...)
			break
		}
	}
	if ch := s.channels[chanID]; ch != nil {
		ch.Bridge = ""
	}
}

func (s *Sim) removeChannel(w http.ResponseWriter, r *http.Request) int {
	id := r.PathValue("id")
	if _, ok := s.bridges[id]; !ok {
		return http.StatusNotFound
	}
	s.leaveBridge(id, r.URL.Query().Get("channel"))
	return http.StatusNoContent
}
