package core

import (
	"math/rand/v2"
	"sort"

	"callcenter-middleware/internal/domain"
)

type candidate struct {
	agent   *agentRT
	penalty int
}

// selectAgents picks which agent(s) to ring for a waiting call according to
// the queue strategy. Lower penalty members are always preferred.
func (c *Core) selectAgents(call *liveCall) []*agentRT {
	if call.targetAgent != 0 {
		if call.tried[call.targetAgent] {
			call.targetAgent = 0 // directed ring failed: fall back to the queue
		} else if a := c.agents[call.targetAgent]; a != nil && c.ringable(a) {
			return []*agentRT{a}
		} else {
			return nil
		}
	}
	q := call.queue
	if q == nil || q.ID == 0 {
		return nil
	}
	cands := c.candidates(call, q, false)
	if len(cands) == 0 && len(call.tried) > 0 {
		// Everyone was tried once: start another round.
		if retry := c.candidates(call, q, true); len(retry) > 0 {
			call.tried = map[int64]bool{}
			cands = retry
		}
	}
	if len(cands) == 0 {
		return nil
	}
	best := cands[0].penalty
	for _, cd := range cands {
		if cd.penalty < best {
			best = cd.penalty
		}
	}
	tier := make([]*agentRT, 0, len(cands))
	for _, cd := range cands {
		if cd.penalty == best {
			tier = append(tier, cd.agent)
		}
	}
	switch q.Strategy {
	case domain.StrategyRingAll:
		return tier
	case domain.StrategyRandom:
		return []*agentRT{tier[rand.IntN(len(tier))]}
	case domain.StrategyFewestCalls:
		sort.SliceStable(tier, func(i, j int) bool {
			if tier[i].CallsToday != tier[j].CallsToday {
				return tier[i].CallsToday < tier[j].CallsToday
			}
			return tier[i].idleSince.Before(tier[j].idleSince)
		})
	case domain.StrategyRoundRobin:
		sort.Slice(tier, func(i, j int) bool { return tier[i].UserID < tier[j].UserID })
		last := c.rr[q.ID]
		pick := tier[0]
		for _, a := range tier {
			if int64(last) < a.UserID {
				pick = a
				break
			}
		}
		c.rr[q.ID] = int(pick.UserID)
		return []*agentRT{pick}
	default: // longest_idle
		sort.SliceStable(tier, func(i, j int) bool { return tier[i].idleSince.Before(tier[j].idleSince) })
	}
	return tier[:1]
}

func (c *Core) candidates(call *liveCall, q *domain.Queue, ignoreTried bool) []candidate {
	var out []candidate
	for _, m := range c.cache.members[q.ID] {
		a := c.agents[m.UserID]
		if a == nil || !c.ringable(a) {
			continue
		}
		if !ignoreTried && call.tried[a.UserID] {
			continue
		}
		if !a.user.HasSkills(q.RequiredSkills) {
			continue
		}
		out = append(out, candidate{agent: a, penalty: m.Penalty})
	}
	return out
}

func (c *Core) ringable(a *agentRT) bool {
	return a.Status == domain.AgentAvailable && a.user.Enabled && a.user.DialEndpoint() != "" &&
		!c.now().Before(a.retryAfter)
}

// loggedIn counts members of q that are not offline.
func (c *Core) loggedIn(q *domain.Queue) int {
	n := 0
	for _, m := range c.cache.members[q.ID] {
		if a := c.agents[m.UserID]; a != nil && a.Status != domain.AgentOffline {
			n++
		}
	}
	return n
}
