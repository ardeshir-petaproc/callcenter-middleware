package domain

import "time"

// Routing rule lifecycle and target kinds.
const (
	RuleDraft    = "draft"    // proposed (by a person or the AI); not used for routing
	RuleApproved = "approved" // active after approval by someone with routing.approve
	RuleRejected = "rejected"
	RuleRetired  = "retired"

	TargetUser       = "user"
	TargetJobProfile = "job_profile" // any available employee holding the job profile
	TargetQueue      = "queue"
	TargetDepartment = "department" // the department's queue

	ToolRouteCall       = "route_call"
	ToolSearchKnowledge = "search_knowledge"
	ToolCreateTicket    = "create_ticket"
)

// Topic is a kind of request callers make, with words that signal it.
type Topic struct {
	ID           int64    `json:"id"`
	TenantID     int64    `json:"tenant_id"`
	Name         string   `json:"name"`
	Description  string   `json:"description"`
	Keywords     []string `json:"keywords"`
	DepartmentID int64    `json:"department_id"` // default owner when no rule matches
}

// RoutingRule sends matching requests to a target. Rules are explainable:
// every match records which conditions held.
type RoutingRule struct {
	ID           int64    `json:"id"`
	TenantID     int64    `json:"tenant_id"`
	Name         string   `json:"name"`
	Topics       []string `json:"topics"`    // topic names; any matches
	Keywords     []string `json:"keywords"`  // any keyword in the request matches
	Languages    []string `json:"languages"` // empty = any
	VIPOnly      bool     `json:"vip_only"`
	TargetType   string   `json:"target_type"`
	TargetID     int64    `json:"target_id"`
	Fallback     Action   `json:"fallback"` // when the target cannot take the call
	Priority     int      `json:"priority"` // higher wins
	Rationale    string   `json:"rationale"`
	Status       string   `json:"status"`
	Source       string   `json:"source"` // manual or ai
	JobProfileID int64    `json:"job_profile_id"`
	Version      int      `json:"version"`
	CreatedBy    int64    `json:"created_by"`
	ApprovedBy   int64    `json:"approved_by"`
	ApprovedAt   string   `json:"approved_at"`
	UpdatedAt    string   `json:"updated_at"`
}

// RouteRequest is what the caller needs, as understood by the AI or IVR.
type RouteRequest struct {
	Text         string `json:"text"`
	Topic        string `json:"topic"`
	Language     string `json:"language"`
	CallerNumber string `json:"caller_number"`
	VIP          bool   `json:"vip"`
	CallID       string `json:"call_id,omitempty"`
}

// RouteCandidate is a rule considered during a decision.
type RouteCandidate struct {
	RuleID   int64    `json:"rule_id"`
	RuleName string   `json:"rule_name"`
	Score    int      `json:"score"`
	Matched  []string `json:"matched"`
	Result   string   `json:"result"` // selected, unavailable, skipped
}

// RoutingDecision explains where a request went and why.
type RoutingDecision struct {
	ID         int64            `json:"id"`
	TenantID   int64            `json:"tenant_id"`
	CallID     string           `json:"call_id"`
	At         time.Time        `json:"at"`
	Request    RouteRequest     `json:"request"`
	Topic      string           `json:"topic"`
	RuleID     int64            `json:"rule_id"`
	Action     Action           `json:"action"`
	TargetName string           `json:"target_name"`
	Reason     string           `json:"reason"`
	Steps      []string         `json:"steps"`
	Candidates []RouteCandidate `json:"candidates"`
}
