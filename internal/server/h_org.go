package server

import (
	"net/http"
	"strings"
	"time"

	"callcenter-middleware/internal/core"
	"callcenter-middleware/internal/domain"
	"callcenter-middleware/internal/store"

	"github.com/go-chi/chi/v5"
)

func (s *Server) orgRoutes(r chi.Router) {
	mountCRUD(r, "/departments", s, s.d.Store.Departments, crudOpts[domain.Department]{entity: "department",
		check: s.checkDepartment, reload: true, writePerm: domain.PermOrgManage, history: true})
	mountCRUD(r, "/job-profiles", s, s.d.Store.JobProfiles, crudOpts[domain.JobProfile]{entity: "job_profile",
		check: s.checkJobProfile, reload: true, writePerm: domain.PermOrgManage, history: true})
	mountCRUD(r, "/numbers", s, s.d.Store.Numbers, crudOpts[domain.PhoneNumber]{entity: "phone_number",
		check: s.checkNumber, reload: true, writePerm: domain.PermFlowsManage, history: true})
	mountCRUD(r, "/roles", s, s.d.Store.Roles, crudOpts[domain.CustomRole]{entity: "custom_role",
		prepare: prepareRole, writePerm: domain.PermUsersManage, history: true})
	r.Get("/revisions", handler(s.listRevisions).ServeHTTP)
	r.Get("/job-profile-templates", handler(s.jobTemplates).ServeHTTP)
	s.inviteRoutes(r)
}

func (s *Server) exists(r *http.Request, kind string, id int64) error {
	if id <= 0 {
		return nil
	}
	var err error
	ctx := r.Context()
	switch kind {
	case "queue":
		_, err = s.d.Store.Queues.Get(ctx, id)
	case "department":
		_, err = s.d.Store.Departments.Get(ctx, id)
	case "schedule":
		_, err = s.d.Store.Schedules.Get(ctx, id)
	case "user":
		_, err = s.d.Store.GetUser(ctx, id)
	case "flow":
		_, err = s.d.Store.Flows.Get(ctx, id)
	case "job_profile":
		_, err = s.d.Store.JobProfiles.Get(ctx, id)
	case "custom_role":
		_, err = s.d.Store.Roles.Get(ctx, id)
	}
	if err != nil {
		return bad(kind + " " + itoa(id) + " does not exist in this organization")
	}
	return nil
}

func (s *Server) checkQueue(r *http.Request, q *domain.Queue) error {
	return s.exists(r, "department", q.DepartmentID)
}

func (s *Server) checkDepartment(r *http.Request, d *domain.Department) error {
	if d.Name = strings.TrimSpace(d.Name); d.Name == "" {
		return bad("name is required")
	}
	if d.Capacity < 0 {
		return bad("capacity must be positive")
	}
	if d.Languages == nil {
		d.Languages = []string{}
	}
	if err := validAction(d.Fallback); err != nil {
		return err
	}
	if err := s.exists(r, "user", d.ManagerID); err != nil {
		return err
	}
	return s.exists(r, "schedule", d.ScheduleID)
}

func (s *Server) checkJobProfile(r *http.Request, j *domain.JobProfile) error {
	if j.Title = strings.TrimSpace(j.Title); j.Title == "" {
		return bad("title is required")
	}
	if j.Responsibilities == nil {
		j.Responsibilities = []string{}
	}
	if j.RequestTypes == nil {
		j.RequestTypes = []string{}
	}
	j.Version = 1
	if j.ID > 0 {
		if prev, err := s.d.Store.JobProfiles.Get(r.Context(), j.ID); err == nil {
			j.Version = prev.Version + 1
		}
	}
	j.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	if err := s.exists(r, "department", j.DepartmentID); err != nil {
		return err
	}
	return s.exists(r, "user", j.EscalateToUserID)
}

// checkNumber keeps every dialed number unique across all organizations,
// since a number decides which tenant receives the call.
func (s *Server) checkNumber(r *http.Request, n *domain.PhoneNumber) error {
	n.Number = strings.TrimSpace(n.Number)
	key := core.NormalizePhone(n.Number)
	if key == "" {
		return bad("number is required")
	}
	all, err := s.d.Store.Numbers.List(store.System(r.Context()))
	if err != nil {
		return err
	}
	for _, other := range all {
		if core.NormalizePhone(other.Number) == key && other.ID != n.ID {
			return errStatus(http.StatusConflict, "this number is already assigned")
		}
	}
	return s.exists(r, "flow", n.FlowID)
}

func prepareRole(c *domain.CustomRole) error {
	if c.Name = strings.TrimSpace(c.Name); c.Name == "" {
		return bad("name is required")
	}
	if c.Permissions == nil {
		c.Permissions = []string{}
	}
	for _, p := range c.Permissions {
		if !domain.ValidPermission(p) {
			return bad("unknown permission " + p)
		}
	}
	return nil
}

// revisionPerms: who may read the history of each kind of object (the
// people who manage it).
var revisionPerms = map[string][]domain.Permission{
	"department": {domain.PermOrgManage}, "job_profile": {domain.PermOrgManage},
	"user": {domain.PermUsersManage}, "custom_role": {domain.PermUsersManage},
	"phone_number": {domain.PermFlowsManage},
	"topic":        {domain.PermRoutingManage, domain.PermRoutingApprove},
	"routing_rule": {domain.PermRoutingManage, domain.PermRoutingApprove},
	"qa_scorecard": {domain.PermQAEvaluate},
}

func (s *Server) listRevisions(w http.ResponseWriter, r *http.Request) error {
	allowed := can(r, domain.PermOrgManage)
	for _, p := range revisionPerms[r.URL.Query().Get("entity")] {
		allowed = allowed || can(r, p)
	}
	if !allowed {
		return errForbidden
	}
	revs, err := s.d.Store.Revisions(r.Context(), r.URL.Query().Get("entity"), qInt64(r, "entity_id"))
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, revs)
	return nil
}
