package routing

import (
	"fmt"
	"strings"

	"callcenter-middleware/internal/domain"
)

// classify finds the request's topic: the one named by the AI, or the one
// whose keywords best match the request text.
func (o *org) classify(req domain.RouteRequest, text string, d *domain.RoutingDecision) *domain.Topic {
	if req.Topic != "" {
		want := strings.TrimSpace(Normalize(req.Topic))
		for i := range o.topics {
			if strings.TrimSpace(Normalize(o.topics[i].Name)) == want {
				d.Topic = o.topics[i].Name
				d.Steps = append(d.Steps, fmt.Sprintf("Topic %q identified by the assistant.", d.Topic))
				return &o.topics[i]
			}
		}
	}
	var best *domain.Topic
	var bestHits []string
	for i := range o.topics {
		if hits := matches(text, append([]string{o.topics[i].Name}, o.topics[i].Keywords...)); len(hits) > len(bestHits) {
			best, bestHits = &o.topics[i], hits
		}
	}
	if best == nil {
		d.Steps = append(d.Steps, "No known topic recognized in the request.")
		return nil
	}
	d.Topic = best.Name
	d.Steps = append(d.Steps, fmt.Sprintf("Topic %q detected from: %s.", best.Name, strings.Join(bestHits, ", ")))
	return best
}

// score checks a rule's conditions against the request.
func score(r *domain.RoutingRule, topic *domain.Topic, text string, req domain.RouteRequest) (domain.RouteCandidate, bool) {
	c := domain.RouteCandidate{RuleID: r.ID, RuleName: r.Name, Matched: []string{}}
	if len(r.Languages) > 0 && req.Language != "" && !containsFold(r.Languages, req.Language) {
		return c, false
	}
	if r.VIPOnly && !req.VIP {
		return c, false
	}
	if topic != nil && containsFold(r.Topics, topic.Name) {
		c.Score += 10
		c.Matched = append(c.Matched, "topic "+topic.Name)
	}
	for _, k := range matches(text, r.Keywords) {
		c.Score += 3
		c.Matched = append(c.Matched, "“"+k+"”")
	}
	if len(r.Topics) == 0 && len(r.Keywords) == 0 {
		c.Score = 1
		c.Matched = append(c.Matched, "catch-all rule")
	}
	if r.VIPOnly {
		c.Matched = append(c.Matched, "VIP caller")
	}
	return c, c.Score > 0
}

func containsFold(list []string, v string) bool {
	for _, s := range list {
		if strings.EqualFold(strings.TrimSpace(s), strings.TrimSpace(v)) {
			return true
		}
	}
	return false
}
