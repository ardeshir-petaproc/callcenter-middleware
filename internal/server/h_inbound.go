package server

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"net/http"
	"strings"

	"callcenter-middleware/internal/domain"
	"callcenter-middleware/internal/messages"
	"callcenter-middleware/internal/store"
)

const inboundKeySetting = "inbound.key_hash"

func keyHash(k string) string {
	sum := sha256.Sum256([]byte(k))
	return hex.EncodeToString(sum[:])
}

// rotateInboundKey issues a new key for the inbound message endpoint; the
// key is shown once and only its hash is stored.
func (s *Server) rotateInboundKey(w http.ResponseWriter, r *http.Request) error {
	raw := make([]byte, 24)
	if _, err := rand.Read(raw); err != nil {
		return err
	}
	key := "ik_" + hex.EncodeToString(raw)
	if err := s.d.Store.SetSetting(r.Context(), inboundKeySetting, keyHash(key)); err != nil {
		return err
	}
	s.audit(r, "integration.inbound_key", "tenant", tenantOf(r), "rotated")
	slug := principalOf(r).tenant.Slug
	writeJSON(w, http.StatusOK, map[string]string{"key": key, "header": "X-Inbound-Key",
		"endpoint": strings.TrimRight(s.d.Config.PublicURL, "/") + "/api/v1/inbound/messages?org=" + slug})
	return nil
}

// inboundMessage accepts a customer chat message from Joow (or another
// channel) for organization ?org=<slug>, authenticated with X-Inbound-Key.
func (s *Server) inboundMessage(w http.ResponseWriter, r *http.Request) error {
	if s.d.Messages == nil {
		return errStatus(http.StatusServiceUnavailable, "messaging not configured")
	}
	t, err := s.d.Store.GetTenantBySlug(store.System(r.Context()), r.URL.Query().Get("org"))
	if err != nil || t.Status != domain.TenantActive {
		return errStatus(http.StatusUnauthorized, "invalid organization or key")
	}
	ctx := store.ForTenant(r.Context(), t.ID)
	want := s.d.Store.GetSetting(ctx, inboundKeySetting, "")
	got := keyHash(r.Header.Get("X-Inbound-Key"))
	if want == "" || subtle.ConstantTimeCompare([]byte(want), []byte(got)) != 1 {
		return errStatus(http.StatusUnauthorized, "invalid organization or key")
	}
	var m domain.InboundMessage
	if err := decode(r, &m); err != nil {
		return err
	}
	tk, created, err := s.d.Messages.Receive(ctx, m)
	if errors.Is(err, messages.ErrInvalid) {
		return bad(err.Error())
	} else if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, map[string]any{"ticket_id": tk.ID, "number": tk.Number, "created": created})
	return nil
}
