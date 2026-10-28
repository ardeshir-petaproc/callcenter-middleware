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
		r.Post("/auth/joow", handler(s.loginJoow).ServeHTTP)
		r.Post("/auth/accept-invite", handler(s.acceptInvite).ServeHTTP)
		r.Get("/auth/invite/{token}", handler(s.peekInvite).ServeHTTP)
		r.Post("/inbound/messages", handler(s.inboundMessage).ServeHTTP)
		if s.d.DevJoowTokens {
			r.Post("/dev/joow-token", handler(s.devJoowToken).ServeHTTP)
		}
		r.Group(func(r chi.Router) {
			r.Use(s.authenticate)
			// Streaming endpoints run without a request timeout.
			r.Get("/ws", s.handleWS)
			r.Get("/calls/{id}/recording", handler(s.callRecording).ServeHTTP)
			r.Get("/voicemails/{id}/audio", handler(s.voicemailAudio).ServeHTTP)
			r.Get("/reports/export.csv", requirePerm(domain.PermReportsView)(handler(s.exportCSV)).ServeHTTP)
			r.Group(func(r chi.Router) {
				r.Use(middleware.Timeout(30 * time.Second))
				s.selfRoutes(r)
				s.callRoutes(r)
				s.configRoutes(r)
				s.orgRoutes(r)
				s.workRoutes(r)
				s.reportRoutes(r)
				s.campaignRoutes(r)
				s.platformRoutes(r)
				s.ticketRoutes(r)
				s.kbRoutes(r)
				s.emailRoutes(r)
				s.routingRoutes(r)
				s.qaRoutes(r)
				s.retentionRoutes(r)
			})
		})
	})
	s.mountStatic(r)
}

func (s *Server) selfRoutes(r chi.Router) {
	r.Get("/auth/me", handler(s.me).ServeHTTP)
	r.Post("/auth/password", handler(s.changePassword).ServeHTTP)
	r.Get("/auth/tenants", handler(s.myTenants).ServeHTTP)
	r.Post("/auth/switch", handler(s.switchTenant).ServeHTTP)
	r.Post("/me/status", handler(s.setMyStatus).ServeHTTP)
	r.Get("/me/stats", handler(s.myStats).ServeHTTP)
	r.Post("/me/dial", handler(s.myDial).ServeHTTP)
	r.Get("/live", handler(s.live).ServeHTTP)
	r.Get("/system/info", handler(s.systemInfo).ServeHTTP)
	r.Get("/settings", handler(s.getSettings).ServeHTTP)
	r.With(requirePerm(domain.PermSettingsManage)).Put("/settings", handler(s.putSettings).ServeHTTP)
	r.With(requirePerm(domain.PermAuditView)).Get("/audit", handler(s.listAudit).ServeHTTP)
	r.Get("/permissions", handler(s.listPermissions).ServeHTTP)
}

func (s *Server) callRoutes(r chi.Router) {
	r.Get("/calls", handler(s.listCalls).ServeHTTP)
	r.Get("/calls/{id}", handler(s.getCall).ServeHTTP)
	r.Patch("/calls/{id}", handler(s.patchCall).ServeHTTP)
	r.Post("/calls/{id}/hangup", handler(s.callHangup).ServeHTTP)
	r.Post("/calls/{id}/transfer", handler(s.callTransfer).ServeHTTP)
	r.Post("/calls/{id}/hold", handler(s.callHold).ServeHTTP)
	s.consultRoutes(r)
	r.Post("/calls/{id}/record", handler(s.callRecord).ServeHTTP)
	r.With(requirePerm(domain.PermCallsMonitor)).Post("/calls/{id}/monitor", handler(s.callMonitor).ServeHTTP)
}

func (s *Server) configRoutes(r chi.Router) {
	users := requirePerm(domain.PermUsersManage)
	r.Get("/users", handler(s.listUsers).ServeHTTP)
	r.Get("/users/{id}", handler(s.getUser).ServeHTTP)
	r.With(users).Post("/users", handler(s.createUser).ServeHTTP)
	r.With(users).Put("/users/{id}", handler(s.updateUser).ServeHTTP)
	r.With(users).Delete("/users/{id}", handler(s.deleteUser).ServeHTTP)
	r.With(requirePerm(domain.PermQueuesManage)).Put("/users/{id}/queues", handler(s.putUserQueues).ServeHTTP)
	r.With(requirePerm(domain.PermCallsControl)).Post("/users/{id}/status", handler(s.forceStatus).ServeHTTP)
	r.Get("/members", handler(s.listMembers).ServeHTTP)
	r.With(requirePerm(domain.PermQueuesManage)).Put("/queues/{id}/members", handler(s.putQueueMembers).ServeHTTP)

	mountCRUD(r, "/queues", s, s.d.Store.Queues, crudOpts[domain.Queue]{entity: "queue", prepare: prepareQueue, check: s.checkQueue,
		reload: true, writePerm: domain.PermQueuesManage, onDelete: s.deleteQueueMembers, history: true})
	mountCRUD(r, "/flows", s, s.d.Store.Flows, crudOpts[domain.IVRFlow]{entity: "flow", prepare: prepareFlow,
		reload: true, writePerm: domain.PermFlowsManage, history: true})
	mountCRUD(r, "/schedules", s, s.d.Store.Schedules, crudOpts[domain.Schedule]{entity: "schedule",
		prepare: prepareSchedule, reload: true, writePerm: domain.PermQueuesManage})
	mountCRUD(r, "/ai/profiles", s, s.d.Store.AIProfiles, crudOpts[domain.AIProfile]{entity: "ai_profile",
		check: s.prepareAIProfile, reload: true, writePerm: domain.PermAIManage, history: true})
	mountCRUD(r, "/dispositions", s, s.d.Store.Dispositions, crudOpts[domain.Disposition]{entity: "disposition",
		prepare: prepareDisposition})
	mountCRUD(r, "/pause-reasons", s, s.d.Store.PauseReasons, crudOpts[domain.PauseReason]{entity: "pause_reason",
		prepare: preparePauseReason})
	mountCRUD(r, "/webhooks", s, s.d.Store.Webhooks, crudOpts[domain.Webhook]{entity: "webhook", prepare: prepareWebhook,
		readPerm: domain.PermSettingsManage, reload: true})
	r.Get("/contacts/lookup", handler(s.lookupContact).ServeHTTP)
	mountCRUD(r, "/contacts", s, s.d.Store.Contacts, crudOpts[domain.Contact]{entity: "contact", prepare: prepareContact,
		writePerm: anyone, filter: contactFilter})
	r.Get("/ai/status", handler(s.aiStatus).ServeHTTP)
	r.With(requirePerm(domain.PermAIManage)).Post("/ai/profiles/{id}/client-secret", handler(s.aiClientSecret).ServeHTTP)
}

func (s *Server) workRoutes(r chi.Router) {
	control := requirePerm(domain.PermCallsControl)
	r.Get("/callbacks", handler(s.listCallbacks).ServeHTTP)
	r.Post("/callbacks", handler(s.createCallback).ServeHTTP)
	r.Put("/callbacks/{id}", handler(s.updateCallback).ServeHTTP)
	r.With(control).Delete("/callbacks/{id}", handler(s.deleteCallback).ServeHTTP)
	r.Post("/callbacks/{id}/dial", handler(s.dialCallback).ServeHTTP)
	r.Get("/voicemails", handler(s.listVoicemails).ServeHTTP)
	r.Patch("/voicemails/{id}", handler(s.patchVoicemail).ServeHTTP)
	r.With(control).Delete("/voicemails/{id}", handler(s.deleteVoicemail).ServeHTTP)
}

func (s *Server) reportRoutes(r chi.Router) {
	r.Group(func(r chi.Router) {
		r.Use(requirePerm(domain.PermReportsView))
		r.Get("/reports/summary", handler(s.reportSummary).ServeHTTP)
		r.Get("/reports/queues", handler(s.reportQueues).ServeHTTP)
		r.Get("/reports/agents", handler(s.reportAgents).ServeHTTP)
		r.Get("/reports/agent-time", handler(s.reportAgentTime).ServeHTTP)
		r.Get("/reports/timeseries", handler(s.reportTimeseries).ServeHTTP)
		r.Get("/reports/breakdown", handler(s.reportBreakdown).ServeHTTP)
	})
}
