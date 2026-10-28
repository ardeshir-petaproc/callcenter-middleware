package kb

import (
	"context"
	"fmt"
	"time"

	"callcenter-middleware/internal/domain"
)

// Allowed decides whether a searcher may use a source. The AI only sees
// active, unexpired, AI-enabled sources inside its profile's scope;
// employees see what their role, permissions and departments allow.
func Allowed(src *domain.KnowledgeSource, a domain.KBAccess, now time.Time) bool {
	if a.Manager && !a.AI {
		return true
	}
	if src.Status != domain.KBActive || src.Expired(now) {
		return false
	}
	if a.AI {
		if !src.AIEnabled {
			return false
		}
		if len(a.Categories) > 0 && !containsStr(a.Categories, src.Category) {
			return false
		}
		return len(a.DepartmentIDs) == 0 || len(src.DepartmentIDs) == 0 || intersects(src.DepartmentIDs, a.DepartmentIDs)
	}
	if src.Visibility == domain.KBVisibilityRestricted && !a.CanRestricted && !roleAllowed(src, a) {
		return false
	}
	return len(src.DepartmentIDs) == 0 || intersects(src.DepartmentIDs, a.DepartmentIDs)
}

func roleAllowed(src *domain.KnowledgeSource, a domain.KBAccess) bool {
	return containsStr(src.AllowedRoles, string(a.Role)) ||
		(a.CustomRoleID > 0 && containsStr(src.AllowedRoles, fmt.Sprintf("role:%d", a.CustomRoleID)))
}

func containsStr(list []string, v string) bool {
	for _, s := range list {
		if s == v {
			return true
		}
	}
	return false
}

func intersects(a, b []int64) bool {
	for _, x := range a {
		for _, y := range b {
			if x == y {
				return true
			}
		}
	}
	return false
}

// Search finds passages the searcher is allowed to read, best first.
func (s *Service) Search(ctx context.Context, tenant int64, query string, a domain.KBAccess, limit int) ([]domain.KBHit, error) {
	if limit <= 0 || limit > 20 {
		limit = 5
	}
	hits, err := s.Store.SearchKB(ctx, tenant, query, limit*5)
	if err != nil {
		return nil, err
	}
	sources := map[int64]*domain.KnowledgeSource{}
	all, err := s.Store.KB.List(ctx)
	if err != nil {
		return nil, err
	}
	for i := range all {
		if all[i].TenantID == tenant {
			sources[all[i].ID] = &all[i]
		}
	}
	now := time.Now()
	out := []domain.KBHit{}
	for _, h := range hits {
		src := sources[h.SourceID]
		if src == nil || !Allowed(src, a, now) {
			continue
		}
		h.Title, h.Category = src.Title, src.Category
		if !a.AI {
			h.Text = "" // employees open the source; the AI gets the passage
		}
		out = append(out, h)
		if len(out) == limit {
			break
		}
	}
	return out, nil
}
