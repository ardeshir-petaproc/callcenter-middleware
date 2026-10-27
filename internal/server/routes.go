package server

import (
	"time"

	"callcenter-middleware/internal/domain"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func (s *Server) registerRoutes(r chi.Router) {
	r.Get("/healthz", s.handleHealthz)
	r.Get("/readyz", s.handleReadyz)
	r.Handle("/metrics", promhttp.Handler())

	r.Route("/api/v1", func(r chi.Router) {
		r.Post("/auth/login", handler(s.login).ServeHTTP)
		r.Group(func(r chi.Router) {
			r.Use(s.authenticate)
			// Streaming endpoints run without a request timeout.
			r.Get("/ws", s.handleWS)
			r.Get("/calls/{id}/recording", handler(s.callRecording).ServeHTTP)
			r.Get("/voicemails/{id}/audio", handler(s.voicemailAudio).ServeHTTP)
			r.Get("/reports/export.csv", require(domain.RoleSupervisor)(handler(s.exportCSV)).ServeHTTP)
			r.Group(func(r chi.Router) {
				r.Use(middleware.Timeout(30 * time.Second))
				s.selfRoutes(r)
				s.callRoutes(r)
				s.configRoutes(r)
				s.workRoutes(r)
				s.reportRoutes(r)
			})
		})
	})
	s.mountStatic(r)
}

func (s *Server) selfRoutes(r chi.Router) {
	r.Get("/auth/me", handler(s.me).ServeHTTP)
	r.Post("/auth/password", handler(s.changePassword).ServeHTTP)
	r.Post("/me/status", handler(s.setMyStatus).ServeHTTP)
	r.Get("/me/stats", handler(s.myStats).ServeHTTP)
	r.Post("/me/dial", handler(s.myDial).ServeHTTP)
	r.Get("/live", handler(s.live).ServeHTTP)
	r.Get("/system/info", handler(s.systemInfo).ServeHTTP)
	r.Get("/settings", handler(s.getSettings).ServeHTTP)
	r.With(require(domain.RoleAdmin)).Put("/settings", handler(s.putSettings).ServeHTTP)
	r.With(require(domain.RoleAdmin)).Get("/audit", handler(s.listAudit).ServeHTTP)
}

func (s *Server) callRoutes(r chi.Router) {
	r.Get("/calls", handler(s.listCalls).ServeHTTP)
	r.Get("/calls/{id}", handler(s.getCall).ServeHTTP)
	r.Patch("/calls/{id}", handler(s.patchCall).ServeHTTP)
	r.Post("/calls/{id}/hangup", handler(s.callHangup).ServeHTTP)
	r.Post("/calls/{id}/transfer", handler(s.callTransfer).ServeHTTP)
	r.Post("/calls/{id}/hold", handler(s.callHold).ServeHTTP)
	r.Post("/calls/{id}/record", handler(s.callRecord).ServeHTTP)
	r.With(require(domain.RoleSupervisor)).Post("/calls/{id}/monitor", handler(s.callMonitor).ServeHTTP)
}

func (s *Server) configRoutes(r chi.Router) {
	admin := require(domain.RoleAdmin)
	r.Get("/users", handler(s.listUsers).ServeHTTP)
	r.Get("/users/{id}", handler(s.getUser).ServeHTTP)
	r.With(admin).Post("/users", handler(s.createUser).ServeHTTP)
	r.With(admin).Put("/users/{id}", handler(s.updateUser).ServeHTTP)
	r.With(admin).Delete("/users/{id}", handler(s.deleteUser).ServeHTTP)
	r.With(admin).Put("/users/{id}/queues", handler(s.putUserQueues).ServeHTTP)
	r.With(require(domain.RoleSupervisor)).Post("/users/{id}/status", handler(s.forceStatus).ServeHTTP)
	r.Get("/members", handler(s.listMembers).ServeHTTP)
	r.With(admin).Put("/queues/{id}/members", handler(s.putQueueMembers).ServeHTTP)

	mountCRUD(r, "/queues", s, s.d.Store.Queues, crudOpts[domain.Queue]{entity: "queue", prepare: prepareQueue, reload: true, onDelete: s.deleteQueueMembers})
	mountCRUD(r, "/flows", s, s.d.Store.Flows, crudOpts[domain.IVRFlow]{entity: "flow", prepare: prepareFlow, reload: true})
	mountCRUD(r, "/schedules", s, s.d.Store.Schedules, crudOpts[domain.Schedule]{entity: "schedule", prepare: prepareSchedule, reload: true})
	mountCRUD(r, "/ai/profiles", s, s.d.Store.AIProfiles, crudOpts[domain.AIProfile]{entity: "ai_profile", prepare: s.prepareAIProfile, reload: true})
	mountCRUD(r, "/dispositions", s, s.d.Store.Dispositions, crudOpts[domain.Disposition]{entity: "disposition", prepare: prepareDisposition})
	mountCRUD(r, "/pause-reasons", s, s.d.Store.PauseReasons, crudOpts[domain.PauseReason]{entity: "pause_reason", prepare: preparePauseReason})
	mountCRUD(r, "/webhooks", s, s.d.Store.Webhooks, crudOpts[domain.Webhook]{entity: "webhook", prepare: prepareWebhook, readRole: domain.RoleAdmin, reload: true})
	r.Get("/contacts/lookup", handler(s.lookupContact).ServeHTTP)
	mountCRUD(r, "/contacts", s, s.d.Store.Contacts, crudOpts[domain.Contact]{entity: "contact", prepare: prepareContact, writeRole: domain.RoleAgent, filter: contactFilter})
	r.Get("/ai/status", handler(s.aiStatus).ServeHTTP)
	r.With(require(domain.RoleSupervisor)).Post("/ai/profiles/{id}/client-secret", handler(s.aiClientSecret).ServeHTTP)
}

func (s *Server) workRoutes(r chi.Router) {
	sup := require(domain.RoleSupervisor)
	r.Get("/callbacks", handler(s.listCallbacks).ServeHTTP)
	r.Post("/callbacks", handler(s.createCallback).ServeHTTP)
	r.Put("/callbacks/{id}", handler(s.updateCallback).ServeHTTP)
	r.With(sup).Delete("/callbacks/{id}", handler(s.deleteCallback).ServeHTTP)
	r.Post("/callbacks/{id}/dial", handler(s.dialCallback).ServeHTTP)
	r.Get("/voicemails", handler(s.listVoicemails).ServeHTTP)
	r.Patch("/voicemails/{id}", handler(s.patchVoicemail).ServeHTTP)
	r.With(sup).Delete("/voicemails/{id}", handler(s.deleteVoicemail).ServeHTTP)
}

func (s *Server) reportRoutes(r chi.Router) {
	r.Group(func(r chi.Router) {
		r.Use(require(domain.RoleSupervisor))
		r.Get("/reports/summary", handler(s.reportSummary).ServeHTTP)
		r.Get("/reports/queues", handler(s.reportQueues).ServeHTTP)
		r.Get("/reports/agents", handler(s.reportAgents).ServeHTTP)
		r.Get("/reports/agent-time", handler(s.reportAgentTime).ServeHTTP)
		r.Get("/reports/timeseries", handler(s.reportTimeseries).ServeHTTP)
		r.Get("/reports/breakdown", handler(s.reportBreakdown).ServeHTTP)
	})
}
