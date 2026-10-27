package server

import (
	"net/http"
	"strings"

	"callcenter-middleware/internal/ai"
	"callcenter-middleware/internal/domain"
)

var realtimeVoices = []string{"marin", "cedar", "alloy", "ash", "ballad", "coral", "echo", "sage", "shimmer", "verse"}

func (s *Server) prepareAIProfile(p *domain.AIProfile) error {
	p.Name = strings.TrimSpace(p.Name)
	if p.Name == "" {
		return bad("name is required")
	}
	if p.Tools == nil {
		p.Tools = []string{domain.ToolTransferToQueue, domain.ToolEndCall}
	}
	for _, t := range p.Tools {
		valid := false
		for _, known := range ai.AllTools {
			if t == known {
				valid = true
			}
		}
		if !valid {
			return bad("unknown tool " + t)
		}
	}
	if p.MaxDurationSec < 0 {
		return bad("max_duration_sec must be positive")
	}
	if p.HandoffQueueID > 0 {
		if _, err := s.d.Store.Queues.Get(s.ctx(), p.HandoffQueueID); err != nil {
			return bad("handoff queue does not exist")
		}
	}
	return nil
}

func (s *Server) aiStatus(w http.ResponseWriter, r *http.Request) error {
	cfg := s.d.Config
	writeJSON(w, http.StatusOK, map[string]any{
		"enabled": cfg.AIEnabled(), "model": cfg.OpenAIRealtimeModel, "voice": cfg.OpenAIVoice,
		"voices": realtimeVoices, "tools": ai.AllTools,
		"summary_model": cfg.OpenAISummaryModel, "transcribe_model": cfg.OpenAITranscribeModel,
	})
	return nil
}

// aiClientSecret mints an ephemeral key so a supervisor can talk to an AI
// profile directly from the browser over WebRTC.
func (s *Server) aiClientSecret(w http.ResponseWriter, r *http.Request) error {
	if !s.d.Config.AIEnabled() || s.d.OpenAI == nil {
		return errStatus(http.StatusServiceUnavailable, "OPENAI_API_KEY is not configured")
	}
	id, err := pathID(r)
	if err != nil {
		return err
	}
	p, err := s.d.Store.AIProfiles.Get(r.Context(), id)
	if err != nil {
		return err
	}
	dir := &ai.Directory{Store: s.d.Store}
	sc := ai.SessionContext{CallerNumber: "browser-test", Queues: dir.QueueNames(r.Context())}
	session := ai.BuildSession(s.d.Config, *p, sc)
	out, err := s.d.OpenAI.ClientSecret(r.Context(), session)
	if err != nil {
		return errStatus(http.StatusBadGateway, err.Error())
	}
	s.audit(r, "ai_test", "ai_profile", id, "")
	writeJSON(w, http.StatusOK, map[string]any{
		"value": out["value"], "expires_at": out["expires_at"], "model": session["model"],
	})
	return nil
}
