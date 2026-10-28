package server

import (
	"fmt"
	"io"
	"net/http"
	"strings"

	"callcenter-middleware/internal/domain"
)

// proxyRecording streams a stored Asterisk recording to the browser.
func (s *Server) proxyRecording(w http.ResponseWriter, r *http.Request, name string, attachment bool) error {
	if name == "" {
		return errStatus(http.StatusNotFound, "no recording")
	}
	if s.d.Fetcher == nil {
		return errStatus(http.StatusServiceUnavailable, "recording storage unavailable")
	}
	resp, err := s.d.Fetcher.FetchRecording(r.Context(), name)
	if err != nil {
		if strings.Contains(err.Error(), "status 404") {
			return errStatus(http.StatusNotFound, "recording not found")
		}
		return errStatus(http.StatusBadGateway, "fetch recording: "+err.Error())
	}
	defer resp.Body.Close()
	ct := resp.Header.Get("Content-Type")
	if ct == "" || ct == "application/octet-stream" {
		ct = "audio/wav"
	}
	w.Header().Set("Content-Type", ct)
	disposition := "inline"
	if attachment {
		disposition = "attachment"
	}
	w.Header().Set("Content-Disposition", fmt.Sprintf("%s; filename=%q", disposition, name+".wav"))
	if cl := resp.Header.Get("Content-Length"); cl != "" {
		w.Header().Set("Content-Length", cl)
	}
	_, err = io.Copy(w, resp.Body)
	return err
}

// callRecording plays (recordings.play, or an agent's own call when the
// organization allows it) or downloads (?download=1, recordings.export) a
// call recording. Every access is audited.
func (s *Server) callRecording(w http.ResponseWriter, r *http.Request) error {
	call, err := s.loadCall(r)
	if err != nil {
		return err
	}
	download := r.URL.Query().Get("download") == "1"
	own := call.AgentID == claimsOf(r).UserID && s.d.Store.GetSetting(r.Context(), "recordings.agent_playback", "1") == "1"
	switch {
	case download && !can(r, domain.PermRecordingsExport):
		return errForbidden
	case !download && !can(r, domain.PermRecordingsPlay) && !own:
		return errForbidden
	}
	action := "recording.play"
	if download {
		action = "recording.download"
	}
	s.audit(r, action, "call", call.ID, call.Recording)
	return s.proxyRecording(w, r, call.Recording, download)
}

func (s *Server) voicemailAudio(w http.ResponseWriter, r *http.Request) error {
	id, err := pathID(r)
	if err != nil {
		return err
	}
	vm, err := s.d.Store.GetVoicemail(r.Context(), id)
	if err != nil {
		return err
	}
	s.audit(r, "voicemail.play", "voicemail", vm.ID, vm.Recording)
	return s.proxyRecording(w, r, vm.Recording, false)
}
