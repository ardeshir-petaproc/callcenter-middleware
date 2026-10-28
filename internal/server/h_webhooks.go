package server

import (
	"net/http"
	"strings"

	"callcenter-middleware/internal/domain"
)

// sealWebhook encrypts a new signing secret; an empty secret on update
// keeps the stored one.
func (s *Server) sealWebhook(r *http.Request, h *domain.Webhook) error {
	h.HasSecret = false
	if h.Secret == "" {
		if id, err := pathID(r); err == nil {
			if prev, err := s.d.Store.Webhooks.Get(r.Context(), id); err == nil {
				h.Secret = prev.Secret
			}
		}
		return nil
	}
	if strings.HasPrefix(h.Secret, "enc:") || s.d.Mail == nil || s.d.Mail.Box == nil {
		return nil
	}
	sealed, err := s.d.Mail.Box.SealString(tenantOf(r), h.Secret)
	if err != nil {
		return err
	}
	h.Secret = sealed
	return nil
}

func hideWebhookSecret(h *domain.Webhook) {
	h.HasSecret, h.Secret = h.Secret != "", ""
}
