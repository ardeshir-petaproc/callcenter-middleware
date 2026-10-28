package routing

import (
	"fmt"
	"sort"
	"strconv"
	"time"

	"callcenter-middleware/internal/domain"
)

var none = domain.Action{}

func agent(u *domain.User) domain.Action {
	return domain.Action{Type: domain.ActionAgent, Target: strconv.FormatInt(u.ID, 10)}
}

func (e *Engine) now() time.Time {
	if e.Now != nil {
		return e.Now()
	}
	return time.Now()
}

// status returns an employee's live status ("offline" when unknown).
func (e *Engine) status(u *domain.User) domain.AgentStatus {
	if e.Live != nil {
		if st, ok := e.Live.Agent(u.ID); ok && st.TenantID == u.TenantID {
			return st.Status
		}
	}
	return domain.AgentOffline
}

// resolve turns a target into an action. When the target cannot take the
// call now it tries, in order: the rule's explicit fallback, the job
// profile's escalation contact, then the department's queue.
func (e *Engine) resolve(o *org, typ string, id int64, fallback domain.Action) (domain.Action, string, string) {
	var notes []string
	steps := []func() (domain.Action, string, string){}
	switch typ {
	case domain.TargetUser:
		steps = append(steps, func() (domain.Action, string, string) { return e.user(o, id) })
	case domain.TargetJobProfile:
		steps = append(steps, func() (domain.Action, string, string) { return e.jobProfile(o, id) })
	case domain.TargetQueue:
		steps = append(steps, func() (domain.Action, string, string) { return e.queue(o, id) })
	case domain.TargetDepartment:
		steps = append(steps, func() (domain.Action, string, string) { return e.department(o, id) })
	}
	if fallback.Type != domain.ActionNone {
		steps = append(steps, func() (domain.Action, string, string) {
			name := string(fallback.Type) + " " + fallback.Target
			return fallback, name, "using the rule's fallback " + name
		})
	}
	if typ == domain.TargetUser || typ == domain.TargetJobProfile {
		steps = append(steps, func() (domain.Action, string, string) {
			jobID, deptID := id, int64(0)
			if j := o.jobs[id]; typ == domain.TargetJobProfile && j != nil {
				deptID = j.DepartmentID
			}
			if u := o.users[id]; typ == domain.TargetUser && u != nil {
				jobID, deptID = u.JobProfileID, firstID(u.DepartmentIDs)
			}
			return e.escalate(o, jobID, deptID)
		})
	}
	for _, step := range steps {
		a, name, note := step()
		notes = append(notes, note)
		if a.Type != domain.ActionNone {
			return a, name, joinNotes(notes)
		}
	}
	return none, "", joinNotes(notes)
}

func (e *Engine) user(o *org, id int64) (domain.Action, string, string) {
	u := o.users[id]
	if u == nil || u.Status == domain.UserDisabled {
		return none, "", "employee no longer exists"
	}
	if st := e.status(u); st != domain.AgentAvailable {
		return none, "", fmt.Sprintf("%s is %s", u.DisplayName, st)
	}
	return agent(u), u.DisplayName, u.DisplayName + " is available"
}

// jobProfile picks the available holder of a job profile who has been
// idle longest.
func (e *Engine) jobProfile(o *org, id int64) (domain.Action, string, string) {
	var free []*domain.User
	for _, u := range o.users {
		if u.JobProfileID == id && u.Status != domain.UserDisabled && e.status(u) == domain.AgentAvailable {
			free = append(free, u)
		}
	}
	title := "job profile"
	if j := o.jobs[id]; j != nil {
		title = j.Title
	}
	if len(free) == 0 {
		return none, "", "nobody holding " + title + " is available"
	}
	sort.Slice(free, func(i, j int) bool {
		a, b := e.lastCall(free[i]), e.lastCall(free[j])
		if !a.Equal(b) {
			return a.Before(b)
		}
		return free[i].ID < free[j].ID
	})
	return agent(free[0]), free[0].DisplayName, fmt.Sprintf("%s (%s) is available", free[0].DisplayName, title)
}

func (e *Engine) lastCall(u *domain.User) time.Time {
	st, _ := e.Live.Agent(u.ID)
	return st.LastCallEnd
}

// escalate follows a job profile's escalation contact, then the department.
func (e *Engine) escalate(o *org, jobID, deptID int64) (domain.Action, string, string) {
	if j := o.jobs[jobID]; j != nil && j.EscalateToUserID > 0 {
		if a, n, note := e.user(o, j.EscalateToUserID); a.Type != domain.ActionNone {
			return a, n, "escalation contact: " + note
		}
	}
	if deptID > 0 {
		return e.department(o, deptID)
	}
	return none, "", "no escalation path"
}
