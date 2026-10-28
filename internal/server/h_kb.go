package server

import (
	"net/http"
	"sort"
	"strings"
	"time"

	"callcenter-middleware/internal/domain"
	"callcenter-middleware/internal/kb"

	"github.com/go-chi/chi/v5"
)

func (s *Server) kbRoutes(r chi.Router) {
	manage := requirePerm(domain.PermKBManage)
	r.Get("/kb/sources", handler(s.listKB).ServeHTTP)
	r.Get("/kb/sources/{id}", handler(s.getKB).ServeHTTP)
	r.Get("/kb/sources/{id}/download", handler(s.downloadKB).ServeHTTP)
	r.Get("/kb/search", handler(s.searchKB).ServeHTTP)
	r.Get("/kb/categories", handler(s.kbCategories).ServeHTTP)
	r.With(manage).Post("/kb/sources", handler(s.saveKB).ServeHTTP)
	r.With(manage).Put("/kb/sources/{id}", handler(s.saveKB).ServeHTTP)
	r.With(manage).Delete("/kb/sources/{id}", handler(s.deleteKB).ServeHTTP)
	r.With(manage).Post("/kb/search/ai", handler(s.searchKBAsAI).ServeHTTP)
}

// kbAccess describes the caller for knowledge permission checks.
func kbAccess(r *http.Request) domain.KBAccess {
	p := principalOf(r)
	return domain.KBAccess{Role: p.user.Role, CustomRoleID: p.user.CustomRoleID, CanRestricted: can(r, domain.PermKBRestricted),
		Manager: can(r, domain.PermKBManage), DepartmentIDs: p.user.DepartmentIDs}
}

func publicKB(src domain.KnowledgeSource) domain.KnowledgeSource {
	src.StorageKey = ""
	return src
}

func (s *Server) listKB(w http.ResponseWriter, r *http.Request) error {
	all, err := s.d.Store.KB.List(r.Context())
	if err != nil {
		return err
	}
	q := r.URL.Query()
	a, now, out := kbAccess(r), time.Now(), []domain.KnowledgeSource{}
	for i := range all {
		src := &all[i]
		if !kb.Allowed(src, a, now) || (q.Get("category") != "" && src.Category != q.Get("category")) {
			continue
		}
		if t := strings.ToLower(q.Get("q")); t != "" && !strings.Contains(strings.ToLower(src.Title+" "+strings.Join(src.Tags, " ")), t) {
			continue
		}
		out = append(out, publicKB(*src))
	}
	writeJSON(w, http.StatusOK, out)
	return nil
}

func (s *Server) loadKB(r *http.Request) (*domain.KnowledgeSource, error) {
	id, err := pathID(r)
	if err != nil {
		return nil, err
	}
	src, err := s.d.Store.KB.Get(r.Context(), id)
	if err != nil {
		return nil, err
	}
	if !kb.Allowed(src, kbAccess(r), time.Now()) {
		return nil, errForbidden
	}
	return src, nil
}

func (s *Server) getKB(w http.ResponseWriter, r *http.Request) error {
	src, err := s.loadKB(r)
	if err != nil {
		return err
	}
	text, _ := s.d.KB.Text(r.Context(), src)
	if runes := []rune(text); len(runes) > 20000 {
		text = string(runes[:20000]) + "…"
	}
	writeJSON(w, http.StatusOK, map[string]any{"source": publicKB(*src), "text": text})
	return nil
}

func (s *Server) downloadKB(w http.ResponseWriter, r *http.Request) error {
	src, err := s.loadKB(r)
	if err != nil {
		return err
	}
	data, err := s.d.KB.Content(r.Context(), src)
	if err != nil {
		return err
	}
	s.audit(r, "kb.download", "kb_source", src.ID, src.Title)
	name := firstNonEmpty(src.FileName, src.Title+".txt")
	w.Header().Set("Content-Type", firstNonEmpty(src.MimeType, "application/octet-stream"))
	w.Header().Set("Content-Disposition", "attachment; filename=\""+strings.ReplaceAll(name, `"`, "")+"\"")
	_, err = w.Write(data)
	return err
}

func (s *Server) deleteKB(w http.ResponseWriter, r *http.Request) error {
	id, err := pathID(r)
	if err != nil {
		return err
	}
	if err := s.d.KB.Delete(r.Context(), id); err != nil {
		return err
	}
	s.audit(r, "kb.delete", "kb_source", id, "")
	ok(w)
	return nil
}

func (s *Server) searchKB(w http.ResponseWriter, r *http.Request) error {
	hits, err := s.d.KB.Search(r.Context(), tenantOf(r), r.URL.Query().Get("q"), kbAccess(r), qInt(r, "limit", 10))
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, hits)
	return nil
}

// searchKBAsAI shows exactly what an AI profile would retrieve.
func (s *Server) searchKBAsAI(w http.ResponseWriter, r *http.Request) error {
	var req struct {
		Query     string `json:"q"`
		ProfileID int64  `json:"profile_id"`
	}
	if err := decode(r, &req); err != nil {
		return err
	}
	a := domain.KBAccess{AI: true}
	if req.ProfileID > 0 {
		p, err := s.d.Store.AIProfiles.Get(r.Context(), req.ProfileID)
		if err != nil {
			return err
		}
		a.DepartmentIDs, a.Categories = p.KBDepartmentIDs, p.KBCategories
	}
	hits, err := s.d.KB.Search(r.Context(), tenantOf(r), req.Query, a, 5)
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, hits)
	return nil
}

func (s *Server) kbCategories(w http.ResponseWriter, r *http.Request) error {
	all, err := s.d.Store.KB.List(r.Context())
	if err != nil {
		return err
	}
	counts := map[string]int{}
	for _, src := range all {
		counts[src.Category]++
	}
	out := []map[string]any{}
	for c, n := range counts {
		out = append(out, map[string]any{"category": c, "count": n})
	}
	sort.Slice(out, func(i, j int) bool { return out[i]["category"].(string) < out[j]["category"].(string) })
	writeJSON(w, http.StatusOK, out)
	return nil
}
