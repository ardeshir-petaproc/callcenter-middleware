package core

import (
	"context"
	"sort"
	"strconv"
	"strings"

	"callcenter-middleware/internal/domain"
)

// configCache is an immutable snapshot of configuration used by the loop.
type configCache struct {
	queues    map[int64]*domain.Queue
	flows     []domain.IVRFlow
	schedules map[int64]*domain.Schedule
	profiles  map[int64]*domain.AIProfile
	members   map[int64][]domain.QueueMember
	users     map[int64]*domain.User
	campaigns []domain.Campaign
}

func loadCache(ctx context.Context, d Deps) (*configCache, error) {
	st := d.Store
	cc := &configCache{
		queues: map[int64]*domain.Queue{}, schedules: map[int64]*domain.Schedule{},
		profiles: map[int64]*domain.AIProfile{}, members: map[int64][]domain.QueueMember{},
		users: map[int64]*domain.User{},
	}
	queues, err := st.Queues.List(ctx)
	if err != nil {
		return nil, err
	}
	for i := range queues {
		cc.queues[queues[i].ID] = &queues[i]
	}
	if cc.flows, err = st.Flows.List(ctx); err != nil {
		return nil, err
	}
	schedules, err := st.Schedules.List(ctx)
	if err != nil {
		return nil, err
	}
	for i := range schedules {
		cc.schedules[schedules[i].ID] = &schedules[i]
	}
	profiles, err := st.AIProfiles.List(ctx)
	if err != nil {
		return nil, err
	}
	for i := range profiles {
		cc.profiles[profiles[i].ID] = &profiles[i]
	}
	if cc.campaigns, err = st.Campaigns.List(ctx); err != nil {
		return nil, err
	}
	members, err := st.ListMembers(ctx)
	if err != nil {
		return nil, err
	}
	for _, m := range members {
		cc.members[m.QueueID] = append(cc.members[m.QueueID], m)
	}
	users, err := st.ListUsers(ctx)
	if err != nil {
		return nil, err
	}
	for i := range users {
		cc.users[users[i].ID] = &users[i]
	}
	return cc, nil
}

// Reload refreshes configuration from the store; live calls keep running.
func (c *Core) Reload(ctx context.Context) error {
	cc, err := loadCache(ctx, c.Deps)
	if err != nil {
		return err
	}
	return c.do(func() error {
		c.cache = cc
		c.syncAgents()
		c.dispatch()
		return nil
	})
}

// flowFor finds the enabled flow for a dialed number, falling back to a
// catch-all flow ("*" or empty DNIS).
func (cc *configCache) flowFor(dnis string) *domain.IVRFlow {
	var fallback *domain.IVRFlow
	for i := range cc.flows {
		f := &cc.flows[i]
		if !f.Enabled {
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

func (cc *configCache) flowByID(id int64) *domain.IVRFlow {
	for i := range cc.flows {
		if cc.flows[i].ID == id {
			return &cc.flows[i]
		}
	}
	return nil
}

// findQueue resolves a queue by id, name or extension.
func (cc *configCache) findQueue(ref string) *domain.Queue {
	ref = strings.TrimSpace(ref)
	if id, err := strconv.ParseInt(ref, 10, 64); err == nil {
		if q, ok := cc.queues[id]; ok {
			return q
		}
	}
	for _, q := range cc.sortedQueues() {
		if strings.EqualFold(q.Name, ref) || (q.Extension != "" && q.Extension == ref) {
			return q
		}
	}
	return nil
}

func (cc *configCache) sortedQueues() []*domain.Queue {
	out := make([]*domain.Queue, 0, len(cc.queues))
	for _, q := range cc.queues {
		out = append(out, q)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Priority != out[j].Priority {
			return out[i].Priority > out[j].Priority
		}
		return out[i].ID < out[j].ID
	})
	return out
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
