package core

import (
	"sort"
	"strconv"
	"strings"

	"callcenter-middleware/internal/domain"
)

// DefaultTenant receives calls to numbers that are not assigned to a tenant.
const DefaultTenant int64 = 1

// tenantFor resolves which tenant owns a dialed number.
func (cc *configCache) tenantFor(dnis string) (int64, *domain.PhoneNumber) {
	if n, ok := cc.numbers[NormalizePhone(dnis)]; ok && dnis != "" {
		return n.TenantID, n
	}
	return DefaultTenant, nil
}

// flowFor finds the tenant's flow for a dialed number: the number's own
// flow, a flow bound to that DNIS, or the tenant's catch-all flow.
func (cc *configCache) flowFor(tenant int64, dnis string, number *domain.PhoneNumber) *domain.IVRFlow {
	if number != nil && number.FlowID > 0 {
		if f := cc.flowByID(tenant, number.FlowID); f != nil && f.Enabled {
			return f
		}
	}
	var fallback *domain.IVRFlow
	for i := range cc.flows {
		f := &cc.flows[i]
		if !f.Enabled || f.TenantID != tenant {
			continue
		}
		if f.DNIS == dnis && dnis != "" {
			return f
		}
		if (f.DNIS == "*" || f.DNIS == "") && fallback == nil {
			fallback = f
		}
	}
	return fallback
}

func (cc *configCache) flowByID(tenant, id int64) *domain.IVRFlow {
	for i := range cc.flows {
		if cc.flows[i].ID == id && cc.flows[i].TenantID == tenant {
			return &cc.flows[i]
		}
	}
	return nil
}

func (cc *configCache) profile(tenant, id int64) *domain.AIProfile {
	if p, ok := cc.profiles[id]; ok && p.TenantID == tenant {
		return p
	}
	return nil
}

func (cc *configCache) user(tenant, id int64) *domain.User {
	if u, ok := cc.users[id]; ok && u.TenantID == tenant {
		return u
	}
	return nil
}

// findQueue resolves a tenant's queue by id, name or extension.
func (cc *configCache) findQueue(tenant int64, ref string) *domain.Queue {
	ref = strings.TrimSpace(ref)
	if id, err := strconv.ParseInt(ref, 10, 64); err == nil {
		if q, ok := cc.queues[id]; ok && q.TenantID == tenant {
			return q
		}
	}
	for _, q := range cc.sortedQueues(tenant) {
		if strings.EqualFold(q.Name, ref) || (q.Extension != "" && q.Extension == ref) {
			return q
		}
	}
	return nil
}

// sortedQueues returns a tenant's queues (all tenants when tenant is 0)
// by priority, department priority included.
func (cc *configCache) sortedQueues(tenant int64) []*domain.Queue {
	out := make([]*domain.Queue, 0, len(cc.queues))
	for _, q := range cc.queues {
		if tenant == 0 || q.TenantID == tenant {
			out = append(out, q)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		pi, pj := cc.queuePriority(out[i]), cc.queuePriority(out[j])
		if pi != pj {
			return pi > pj
		}
		return out[i].ID < out[j].ID
	})
	return out
}

// queuePriority combines department and queue priority.
func (cc *configCache) queuePriority(q *domain.Queue) int {
	p := q.Priority
	if d, ok := cc.departments[q.DepartmentID]; ok {
		p += d.Priority * 100
	}
	return p
}

func (cc *configCache) penalty(queueID, userID int64) (int, bool) {
	for _, m := range cc.members[queueID] {
		if m.UserID == userID {
			return m.Penalty, true
		}
	}
	return 0, false
}

func (cc *configCache) queuesOf(userID int64) []int64 {
	ids := []int64{}
	for qid, ms := range cc.members {
		for _, m := range ms {
			if m.UserID == userID {
				ids = append(ids, qid)
			}
		}
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids
}
