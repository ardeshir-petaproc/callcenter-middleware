package routing

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"callcenter-middleware/internal/domain"
	"callcenter-middleware/internal/store"
)

// Live reports agents' real-time state.
type Live interface {
	Agent(userID int64) (domain.AgentState, bool)
}

// Engine makes routing decisions.
type Engine struct {
	Store *store.Store
	Live  Live
	Now   func() time.Time
}

// org is the tenant data a decision needs.
type org struct {
	topics []domain.Topic
	rules  []domain.RoutingRule
	users  map[int64]*domain.User
	queues []domain.Queue
	depts  map[int64]*domain.Department
	jobs   map[int64]*domain.JobProfile
	sched  map[int64]*domain.Schedule
}

func (e *Engine) load(ctx context.Context) (*org, error) {
	o := &org{users: map[int64]*domain.User{}, depts: map[int64]*domain.Department{}, jobs: map[int64]*domain.JobProfile{},
		sched: map[int64]*domain.Schedule{}}
	var err error
	if o.topics, err = e.Store.Topics.List(ctx); err != nil {
		return nil, err
	}
	if o.rules, err = e.Store.Rules.List(ctx); err != nil {
		return nil, err
	}
	if o.queues, err = e.Store.Queues.List(ctx); err != nil {
		return nil, err
	}
	users, err := e.Store.ListUsers(ctx)
	if err != nil {
		return nil, err
	}
	for i := range users {
		o.users[users[i].ID] = &users[i]
	}
	depts, _ := e.Store.Departments.List(ctx)
	for i := range depts {
		o.depts[depts[i].ID] = &depts[i]
	}
	jobs, _ := e.Store.JobProfiles.List(ctx)
	for i := range jobs {
		o.jobs[jobs[i].ID] = &jobs[i]
	}
	scheds, _ := e.Store.Schedules.List(ctx)
	for i := range scheds {
		o.sched[scheds[i].ID] = &scheds[i]
	}
	return o, nil
}

// Decide picks a target for a request and explains why. An empty Action
// means no rule applied and the caller should use its default handoff.
func (e *Engine) Decide(ctx context.Context, req domain.RouteRequest) (*domain.RoutingDecision, error) {
	o, err := e.load(ctx)
	if err != nil {
		return nil, err
	}
	d := &domain.RoutingDecision{CallID: req.CallID, Request: req, Candidates: []domain.RouteCandidate{}}
	text := Normalize(req.Text + " " + req.Topic)
	topic := o.classify(req, text, d)
	type cand struct {
		rule *domain.RoutingRule
		c    domain.RouteCandidate
	}
	var cands []cand
	for i := range o.rules {
		r := &o.rules[i]
		if r.Status != domain.RuleApproved {
			continue
		}
		if c, ok := score(r, topic, text, req); ok {
			cands = append(cands, cand{r, c})
		}
	}
	sort.SliceStable(cands, func(i, j int) bool {
		if cands[i].rule.Priority != cands[j].rule.Priority {
			return cands[i].rule.Priority > cands[j].rule.Priority
		}
		return cands[i].c.Score > cands[j].c.Score
	})
	if len(cands) == 0 {
		d.Steps = append(d.Steps, "No approved rule matched the request.")
	}
	for _, c := range cands {
		if d.Action.Type != domain.ActionNone {
			c.c.Result = "skipped"
			d.Candidates = append(d.Candidates, c.c)
			continue
		}
		action, name, note := e.resolve(o, c.rule.TargetType, c.rule.TargetID, c.rule.Fallback)
		d.Steps = append(d.Steps, fmt.Sprintf("Rule %q (matched %s): %s", c.rule.Name, strings.Join(c.c.Matched, ", "), note))
		c.c.Result = "unavailable"
		if action.Type != domain.ActionNone {
			c.c.Result = "selected"
			d.RuleID, d.Action, d.TargetName = c.rule.ID, action, name
			d.Reason = fmt.Sprintf("%s → %s (rule %q", topicOr(topic, "request"), name, c.rule.Name)
			if c.rule.Rationale != "" {
				d.Reason += ": " + c.rule.Rationale
			}
			d.Reason += ")"
		}
		d.Candidates = append(d.Candidates, c.c)
	}
	if d.Action.Type == domain.ActionNone && topic != nil && topic.DepartmentID > 0 {
		action, name, note := e.resolve(o, domain.TargetDepartment, topic.DepartmentID, domain.Action{})
		d.Steps = append(d.Steps, fmt.Sprintf("Topic %q belongs to a department: %s", topic.Name, note))
		if action.Type != domain.ActionNone {
			d.Action, d.TargetName = action, name
			d.Reason = fmt.Sprintf("%s → %s (topic's department)", topic.Name, name)
		}
	}
	if d.Action.Type == domain.ActionNone {
		d.Reason = "No rule or topic applied; default handoff."
	}
	return d, nil
}

func topicOr(t *domain.Topic, def string) string {
	if t == nil {
		return def
	}
	return t.Name
}
