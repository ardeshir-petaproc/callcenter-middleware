package routing

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"callcenter-middleware/internal/domain"
)

// Drafter is a chat model that answers with JSON (OpenAI in production).
type Drafter interface {
	ChatJSON(ctx context.Context, system, user string, out any) error
}

const draftPrompt = `You turn an employee job description into call routing rules for a call center.
Return JSON {"rules":[{"name":"...","topics":["..."],"keywords":["..."],"escalation":false,"priority":0,"rationale":"..."}]}.
- One rule per kind of request the role handles. Use only topic names from the provided list in "topics".
- "keywords": short words or phrases callers would say, in the description's language (and English if helpful).
- Requests beyond the role's authority or matching its escalation conditions get "escalation": true and priority 5.
- "rationale": one sentence quoting the part of the job description that justifies the rule.
Never invent responsibilities that are not in the description.`

type draft struct {
	Name       string   `json:"name"`
	Topics     []string `json:"topics"`
	Keywords   []string `json:"keywords"`
	Escalation bool     `json:"escalation"`
	Priority   int      `json:"priority"`
	Rationale  string   `json:"rationale"`
}

// Generate proposes draft rules from a job profile. The rules are never
// active until someone with routing.approve approves them. With no model
// (or on model failure) a deterministic draft is built from the request
// types and escalation conditions.
func Generate(ctx context.Context, m Drafter, job *domain.JobProfile, topics []domain.Topic) ([]domain.RoutingRule, string) {
	var drafts []draft
	method := "ai"
	if m != nil {
		var names []string
		for _, t := range topics {
			names = append(names, t.Name)
		}
		in, _ := json.Marshal(map[string]any{"title": job.Title, "description": job.Description,
			"responsibilities": job.Responsibilities, "request_types": job.RequestTypes,
			"authority_limits": job.AuthorityLimits, "escalation_conditions": job.EscalationConditions, "topics": names})
		var out struct {
			Rules []draft `json:"rules"`
		}
		if err := m.ChatJSON(ctx, draftPrompt, string(in), &out); err == nil {
			drafts = out.Rules
		}
	}
	if len(drafts) == 0 {
		drafts, method = heuristic(job, topics), "template"
	}
	var rules []domain.RoutingRule
	for _, d := range drafts {
		r := domain.RoutingRule{Name: d.Name, Topics: knownTopics(d.Topics, topics), Keywords: clean(d.Keywords),
			Priority: d.Priority, Rationale: d.Rationale, Status: domain.RuleDraft, Source: "ai", JobProfileID: job.ID,
			TargetType: domain.TargetJobProfile, TargetID: job.ID, Languages: []string{}}
		if d.Escalation && job.EscalateToUserID > 0 {
			r.TargetType, r.TargetID = domain.TargetUser, job.EscalateToUserID
		}
		if r.Name == "" || (len(r.Topics) == 0 && len(r.Keywords) == 0) {
			continue
		}
		rules = append(rules, r)
	}
	return rules, method
}

func heuristic(job *domain.JobProfile, topics []domain.Topic) []draft {
	var out []draft
	for _, rt := range job.RequestTypes {
		d := draft{Name: fmt.Sprintf("%s: %s", job.Title, rt), Keywords: []string{rt},
			Rationale: fmt.Sprintf("Job profile %q lists %q as a request type it handles.", job.Title, rt)}
		for _, t := range topics {
			if strings.EqualFold(t.Name, rt) {
				d.Topics, d.Keywords = []string{t.Name}, nil
			}
		}
		out = append(out, d)
	}
	for _, cond := range splitList(job.EscalationConditions) {
		out = append(out, draft{Name: fmt.Sprintf("%s escalation: %s", job.Title, cond), Keywords: []string{cond},
			Escalation: true, Priority: 5, Rationale: fmt.Sprintf("Escalation condition of %q: %s.", job.Title, cond)})
	}
	return out
}

// splitList splits free text on newlines, semicolons and Persian/Latin commas.
func splitList(s string) []string {
	f := strings.FieldsFunc(s, func(r rune) bool { return r == '\n' || r == ';' || r == '،' || r == ',' })
	return clean(f)
}

func clean(in []string) []string {
	out := []string{}
	for _, s := range in {
		if s = strings.Trim(strings.TrimSpace(s), ".-•*"); s != "" && len([]rune(s)) <= 60 {
			out = append(out, strings.TrimSpace(s))
		}
	}
	return out
}

func knownTopics(in []string, topics []domain.Topic) []string {
	out := []string{}
	for _, n := range in {
		for _, t := range topics {
			if strings.EqualFold(strings.TrimSpace(n), t.Name) {
				out = append(out, t.Name)
			}
		}
	}
	return out
}
