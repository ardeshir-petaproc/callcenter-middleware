package server

import (
	"net/http"

	"callcenter-middleware/internal/domain"
	"callcenter-middleware/internal/retention"

	"github.com/go-chi/chi/v5"
)

func (s *Server) retentionRoutes(r chi.Router) {
	settings := requirePerm(domain.PermSettingsManage)
	r.With(settings).Get("/retention", handler(s.getRetention).ServeHTTP)
	r.With(settings).Put("/retention", handler(s.putRetention).ServeHTTP)
	r.With(settings).Post("/retention/run", handler(s.runRetention).ServeHTTP)
	r.With(settings).Post("/integrations/inbound-key", handler(s.rotateInboundKey).ServeHTTP)
	r.With(settings, requirePerm(domain.PermAuditView)).Get("/export/organization.zip", handler(s.exportOrganization).ServeHTTP)
}

func (s *Server) getRetention(w http.ResponseWriter, r *http.Request) error {
	writeJSON(w, http.StatusOK, retention.Load(r.Context(), s.d.Store))
	return nil
}

func (s *Server) putRetention(w http.ResponseWriter, r *http.Request) error {
	var p retention.Policy
	if err := decode(r, &p); err != nil {
		return err
	}
	if err := retention.Save(r.Context(), s.d.Store, p); err != nil {
		return bad(err.Error())
	}
	s.audit(r, "retention.policy", "tenant", tenantOf(r), "")
	writeJSON(w, http.StatusOK, p)
	return nil
}

// runRetention applies the policy now instead of waiting for the hourly run.
func (s *Server) runRetention(w http.ResponseWriter, r *http.Request) error {
	if s.d.Retention == nil {
		return errStatus(http.StatusServiceUnavailable, "retention not configured")
	}
	res, err := s.d.Retention.Apply(r.Context())
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, res)
	return nil
}
