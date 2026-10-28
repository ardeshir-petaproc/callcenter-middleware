package core

import (
	"context"

	"callcenter-middleware/internal/domain"
	"callcenter-middleware/internal/store"
)

// configCache is an immutable snapshot of configuration for every tenant.
// Lookups are tenant-aware so a call can never be routed into another
// tenant's queues, flows, agents or AI profiles.
type configCache struct {
	queues      map[int64]*domain.Queue
	flows       []domain.IVRFlow
	schedules   map[int64]*domain.Schedule
	profiles    map[int64]*domain.AIProfile
	members     map[int64][]domain.QueueMember
	users       map[int64]*domain.User
	campaigns   []domain.Campaign
	departments map[int64]*domain.Department
	numbers     map[string]*domain.PhoneNumber
	jobs        map[int64]*domain.JobProfile
}

func index[T any](items []T, id func(*T) int64) map[int64]*T {
	out := make(map[int64]*T, len(items))
	for i := range items {
		out[id(&items[i])] = &items[i]
	}
	return out
}

func loadCache(ctx context.Context, d Deps) (*configCache, error) {
	st := d.Store
	ctx = store.System(ctx)
	cc := &configCache{members: map[int64][]domain.QueueMember{}, numbers: map[string]*domain.PhoneNumber{}}
	queues, err := st.Queues.List(ctx)
	if err != nil {
		return nil, err
	}
	cc.queues = index(queues, func(q *domain.Queue) int64 { return q.ID })
	if cc.flows, err = st.Flows.List(ctx); err != nil {
		return nil, err
	}
	schedules, err := st.Schedules.List(ctx)
	if err != nil {
		return nil, err
	}
	cc.schedules = index(schedules, func(s *domain.Schedule) int64 { return s.ID })
	profiles, err := st.AIProfiles.List(ctx)
	if err != nil {
		return nil, err
	}
	cc.profiles = index(profiles, func(p *domain.AIProfile) int64 { return p.ID })
	if cc.campaigns, err = st.Campaigns.List(ctx); err != nil {
		return nil, err
	}
	depts, err := st.Departments.List(ctx)
	if err != nil {
		return nil, err
	}
	cc.departments = index(depts, func(d *domain.Department) int64 { return d.ID })
	jobs, err := st.JobProfiles.List(ctx)
	if err != nil {
		return nil, err
	}
	cc.jobs = index(jobs, func(j *domain.JobProfile) int64 { return j.ID })
	numbers, err := st.Numbers.List(ctx)
	if err != nil {
		return nil, err
	}
	for i := range numbers {
		if n := &numbers[i]; n.Enabled {
			cc.numbers[NormalizePhone(n.Number)] = n
		}
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
	cc.users = index(users, func(u *domain.User) int64 { return u.ID })
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
