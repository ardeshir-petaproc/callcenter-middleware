package server

import (
	"io/fs"
	"net/http"
	"path"
	"strings"

	"github.com/go-chi/chi/v5"
)

var publicSettings = map[string]bool{"company_name": true, "record_outbound": true, "wallboard_message": true}

func (s *Server) getSettings(w http.ResponseWriter, r *http.Request) error {
	all, err := s.d.Store.AllSettings(r.Context())
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, all)
	return nil
}

func (s *Server) putSettings(w http.ResponseWriter, r *http.Request) error {
	var req map[string]string
	if err := decode(r, &req); err != nil {
		return err
	}
	for k, v := range req {
		if !publicSettings[k] && !strings.HasPrefix(k, "custom_") {
			return bad("unknown setting " + k + " (use custom_ prefix for your own keys)")
		}
		if err := s.d.Store.SetSetting(r.Context(), k, v); err != nil {
			return err
		}
	}
	s.audit(r, "update", "settings", "", strings.Join(keys(req), ","))
	return s.getSettings(w, r)
}

func keys(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

func (s *Server) listAudit(w http.ResponseWriter, r *http.Request) error {
	entries, err := s.d.Store.ListAudit(r.Context(), qInt(r, "limit", 100), qInt(r, "offset", 0))
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, entries)
	return nil
}

// mountStatic serves the embedded SPA with index.html fallback for client routes.
func (s *Server) mountStatic(r chi.Router) {
	if s.d.StaticFS == nil {
		return
	}
	files := http.FileServer(s.d.StaticFS)
	r.Get("/*", func(w http.ResponseWriter, req *http.Request) {
		p := path.Clean("/" + chi.URLParam(req, "*"))
		if strings.HasPrefix(p, "/api/") {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
			return
		}
		if f, err := s.d.StaticFS.Open(p); err == nil {
			st, _ := f.Stat()
			f.Close()
			if st != nil && !st.IsDir() {
				if strings.HasPrefix(p, "/assets/") {
					w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
				}
				files.ServeHTTP(w, req)
				return
			}
		}
		w.Header().Set("Cache-Control", "no-cache")
		req.URL.Path = "/"
		files.ServeHTTP(w, req)
	})
}

// StaticFS wraps an fs.FS for http serving.
func StaticFS(f fs.FS) http.FileSystem { return http.FS(f) }
