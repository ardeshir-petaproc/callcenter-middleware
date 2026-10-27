package server

import (
	"fmt"
	"io"
	"net/http"
	"strings"
)

// proxyRecording streams a stored Asterisk recording to the browser.
func (s *Server) proxyRecording(w http.ResponseWriter, r *http.Request, name string) error {
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
	w.Header().Set("Content-Disposition", fmt.Sprintf("inline; filename=%q", name+".wav"))
	if cl := resp.Header.Get("Content-Length"); cl != "" {
		w.Header().Set("Content-Length", cl)
	}
	_, err = io.Copy(w, resp.Body)
	return err
}

func (s *Server) callRecording(w http.ResponseWriter, r *http.Request) error {
	call, err := s.loadCall(r)
	if err != nil {
		return err
	}
	return s.proxyRecording(w, r, call.Recording)
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
	return s.proxyRecording(w, r, vm.Recording)
}
