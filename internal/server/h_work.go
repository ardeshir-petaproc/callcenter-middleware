package server

import (
	"context"
	"net/http"
	"strings"

	"callcenter-middleware/internal/core"
	"callcenter-middleware/internal/domain"
)

func (s *Server) ctx() context.Context { return context.Background() }

func (s *Server) listCallbacks(w http.ResponseWriter, r *http.Request) error {
	cbs, err := s.d.Store.ListCallbacks(r.Context(), r.URL.Query().Get("status"), qInt(r, "limit", 200))
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, cbs)
	return nil
}

func (s *Server) createCallback(w http.ResponseWriter, r *http.Request) error {
	var cb domain.Callback
	if err := decode(r, &cb); err != nil {
		return err
	}
	if core.NormalizePhone(cb.Number) == "" {
		return bad("number is required")
	}
	cb.ID, cb.Status, cb.Attempts = 0, domain.CallbackPending, 0
	if err := s.d.Store.CreateCallback(r.Context(), &cb); err != nil {
		return err
	}
	s.audit(r, "create", "callback", cb.ID, cb.Number)
	s.d.Hub.Publish(domain.Event{Type: domain.EvCallbackCreated, Data: cb})
	writeJSON(w, http.StatusCreated, cb)
	return nil
}

func (s *Server) updateCallback(w http.ResponseWriter, r *http.Request) error {
	id, err := pathID(r)
	if err != nil {
		return err
	}
	existing, err := s.d.Store.GetCallback(r.Context(), id)
	if err != nil {
		return err
	}
	cb := *existing
	if err := decode(r, &cb); err != nil {
		return err
	}
	cb.ID, cb.CallID, cb.CreatedAt = id, existing.CallID, existing.CreatedAt
	switch cb.Status {
	case domain.CallbackPending, domain.CallbackInProgress, domain.CallbackDone, domain.CallbackFailed, domain.CallbackCancelled:
	default:
		return bad("invalid status")
	}
	if err := s.d.Store.UpdateCallback(r.Context(), &cb); err != nil {
		return err
	}
	s.d.Hub.Publish(domain.Event{Type: domain.EvCallbackCreated, Data: cb})
	writeJSON(w, http.StatusOK, cb)
	return nil
}

func (s *Server) deleteCallback(w http.ResponseWriter, r *http.Request) error {
	id, err := pathID(r)
	if err != nil {
		return err
	}
	if err := s.d.Store.DeleteCallback(r.Context(), id); err != nil {
		return err
	}
	s.audit(r, "delete", "callback", id, "")
	ok(w)
	return nil
}

// dialCallback rings the requesting user and then the callback number.
func (s *Server) dialCallback(w http.ResponseWriter, r *http.Request) error {
	id, err := pathID(r)
	if err != nil {
		return err
	}
	cb, err := s.d.Store.GetCallback(r.Context(), id)
	if err != nil {
		return err
	}
	if cb.Status == domain.CallbackDone || cb.Status == domain.CallbackCancelled {
		return errStatus(http.StatusConflict, "callback already closed")
	}
	uid := claimsOf(r).UserID
	callID, err := s.d.Core.Dial(uid, cb.Number, cb.ID)
	if err != nil {
		return dialErr(err)
	}
	cb.Status, cb.AssignedTo = domain.CallbackInProgress, uid
	cb.Attempts++
	_ = s.d.Store.UpdateCallback(r.Context(), cb)
	s.audit(r, "dial", "callback", id, cb.Number)
	writeJSON(w, http.StatusOK, map[string]string{"call_id": callID})
	return nil
}

func (s *Server) listVoicemails(w http.ResponseWriter, r *http.Request) error {
	vms, err := s.d.Store.ListVoicemails(r.Context(), qInt(r, "limit", 200))
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, vms)
	return nil
}

func (s *Server) patchVoicemail(w http.ResponseWriter, r *http.Request) error {
	id, err := pathID(r)
	if err != nil {
		return err
	}
	vm, err := s.d.Store.GetVoicemail(r.Context(), id)
	if err != nil {
		return err
	}
	var req struct {
		Heard *bool `json:"heard"`
	}
	if err := decode(r, &req); err != nil {
		return err
	}
	if req.Heard != nil {
		vm.Heard = *req.Heard
	}
	if err := s.d.Store.UpdateVoicemail(r.Context(), vm); err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, vm)
	return nil
}

func (s *Server) deleteVoicemail(w http.ResponseWriter, r *http.Request) error {
	id, err := pathID(r)
	if err != nil {
		return err
	}
	if err := s.d.Store.DeleteVoicemail(r.Context(), id); err != nil {
		return err
	}
	s.audit(r, "delete", "voicemail", id, "")
	ok(w)
	return nil
}

func (s *Server) lookupContact(w http.ResponseWriter, r *http.Request) error {
	phone := strings.TrimSpace(r.URL.Query().Get("phone"))
	if phone == "" {
		return bad("phone is required")
	}
	resp := map[string]any{"contact": nil}
	if c := core.FindContact(r.Context(), s.d.Store, phone); c != nil {
		resp["contact"] = c
	}
	calls, err := s.d.Store.RecentCallsByNumber(r.Context(), core.NormalizePhone(phone), 10)
	if err != nil {
		return err
	}
	resp["recent_calls"] = calls
	writeJSON(w, http.StatusOK, resp)
	return nil
}
