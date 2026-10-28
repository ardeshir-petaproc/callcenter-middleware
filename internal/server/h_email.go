package server

import (
	"net/http"
	"time"

	"callcenter-middleware/internal/domain"

	"github.com/go-chi/chi/v5"
)

func (s *Server) emailRoutes(r chi.Router) {
	r.Group(func(r chi.Router) {
		r.Use(requirePerm(domain.PermSettingsManage))
		r.Get("/email-connectors", handler(s.listMailboxes).ServeHTTP)
		r.Post("/email-connectors", handler(s.saveMailbox).ServeHTTP)
		r.Post("/email-connectors/folders", handler(s.mailboxFolders).ServeHTTP)
		r.Put("/email-connectors/{id}", handler(s.saveMailbox).ServeHTTP)
		r.Delete("/email-connectors/{id}", handler(s.disconnectMailbox).ServeHTTP)
		r.Post("/email-connectors/{id}/sync", handler(s.syncMailbox).ServeHTTP)
		r.Post("/email-connectors/{id}/ai", handler(s.mailboxAI).ServeHTTP)
		r.Get("/email-connectors/{id}/messages", handler(s.mailboxMessages).ServeHTTP)
	})
}

func publicMailbox(c domain.EmailConnector) domain.EmailConnector {
	c.HasPassword, c.Password = c.Password != "", ""
	return c
}

func (s *Server) listMailboxes(w http.ResponseWriter, r *http.Request) error {
	all, err := s.d.Store.Mailboxes.List(r.Context())
	if err != nil {
		return err
	}
	out := make([]domain.EmailConnector, 0, len(all))
	for _, c := range all {
		out = append(out, publicMailbox(c))
	}
	writeJSON(w, http.StatusOK, out)
	return nil
}

// mailboxFolders lists folders for the settings being edited, so the
// administrator chooses exactly which folder is shared.
func (s *Server) mailboxFolders(w http.ResponseWriter, r *http.Request) error {
	var req struct {
		domain.EmailConnector
		ID int64 `json:"id"`
	}
	if err := decode(r, &req); err != nil {
		return err
	}
	c, pw := req.EmailConnector, req.Password
	if pw == "" && req.ID > 0 {
		prev, err := s.d.Store.Mailboxes.Get(r.Context(), req.ID)
		if err != nil {
			return err
		}
		if pw, err = s.d.Mail.Password(prev); err != nil {
			return err
		}
	}
	if err := normalizeMailbox(&c); err != nil {
		return err
	}
	folders, err := s.d.Mail.Mailbox.Folders(r.Context(), &c, pw)
	if err != nil {
		return bad("mailbox: " + err.Error())
	}
	writeJSON(w, http.StatusOK, folders)
	return nil
}

func (s *Server) syncMailbox(w http.ResponseWriter, r *http.Request) error {
	id, err := pathID(r)
	if err != nil {
		return err
	}
	n, err := s.d.Mail.Sync(r.Context(), id)
	if err != nil {
		if _, gerr := s.d.Store.Mailboxes.Get(r.Context(), id); gerr != nil {
			return gerr
		}
		return bad("sync failed: " + err.Error())
	}
	s.audit(r, "email.sync", "email_connector", id, "")
	writeJSON(w, http.StatusOK, map[string]int{"imported": n})
	return nil
}

func (s *Server) mailboxAI(w http.ResponseWriter, r *http.Request) error {
	id, err := pathID(r)
	if err != nil {
		return err
	}
	var req struct {
		Enabled bool `json:"enabled"`
	}
	if err := decode(r, &req); err != nil {
		return err
	}
	n, err := s.d.Mail.SetAIAccess(r.Context(), id, req.Enabled)
	if err != nil {
		return err
	}
	action := "email.ai_excluded"
	if req.Enabled {
		action = "email.ai_allowed"
	}
	s.audit(r, action, "email_connector", id, "")
	writeJSON(w, http.StatusOK, map[string]int{"sources_changed": n})
	return nil
}

// disconnectMailbox revokes access: credentials are deleted; with
// ?purge=1 imported messages and knowledge are removed as well.
func (s *Server) disconnectMailbox(w http.ResponseWriter, r *http.Request) error {
	id, err := pathID(r)
	if err != nil {
		return err
	}
	purge := r.URL.Query().Get("purge") == "1"
	if err := s.d.Mail.Disconnect(r.Context(), id, purge); err != nil {
		return err
	}
	detail := "credentials removed"
	if purge {
		detail += "; imported mail and knowledge purged"
	}
	s.audit(r, "email.disconnect", "email_connector", id, detail)
	ok(w)
	return nil
}

func (s *Server) mailboxMessages(w http.ResponseWriter, r *http.Request) error {
	id, err := pathID(r)
	if err != nil {
		return err
	}
	if _, err := s.d.Store.Mailboxes.Get(r.Context(), id); err != nil {
		return err
	}
	msgs, err := s.d.Store.ListEmails(r.Context(), id, qInt(r, "limit", 100))
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, msgs)
	return nil
}

func nowRFC() string { return time.Now().UTC().Format(time.RFC3339) }
