package server

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"callcenter-middleware/internal/domain"
	"callcenter-middleware/internal/kb"
)

// saveKB accepts JSON ({...source, "content": "text"}) or multipart form
// data with a "meta" JSON field and a "file" part.
func (s *Server) saveKB(w http.ResponseWriter, r *http.Request) error {
	var src domain.KnowledgeSource
	var fileName string
	var data []byte
	if strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/") {
		if err := r.ParseMultipartForm(kb.MaxFileSize + 1<<20); err != nil {
			return bad("invalid upload: " + err.Error())
		}
		if err := json.Unmarshal([]byte(r.FormValue("meta")), &src); err != nil {
			return bad("meta must be JSON")
		}
		if f, hdr, err := r.FormFile("file"); err == nil {
			defer f.Close()
			if data, err = io.ReadAll(io.LimitReader(f, kb.MaxFileSize+1)); err != nil {
				return err
			}
			fileName = hdr.Filename
		}
	} else {
		var req struct {
			domain.KnowledgeSource
			Content *string `json:"content"`
		}
		if err := decode(r, &req); err != nil {
			return err
		}
		src = req.KnowledgeSource
		if req.Content != nil {
			data = []byte(*req.Content)
		}
	}
	src.ID, src.TenantID = 0, 0
	if id, err := pathID(r); err == nil {
		src.ID = id
	}
	if src.OwnerID == 0 {
		src.OwnerID = claimsOf(r).UserID
	}
	if err := s.checkKBRefs(r, &src); err != nil {
		return err
	}
	if err := s.d.KB.Save(r.Context(), &src, fileName, data); err != nil {
		if strings.Contains(err.Error(), "not found") {
			return err
		}
		return bad(err.Error())
	}
	s.audit(r, "kb.save", "kb_source", src.ID, src.Title)
	writeJSON(w, http.StatusOK, publicKB(src))
	return nil
}

func (s *Server) checkKBRefs(r *http.Request, src *domain.KnowledgeSource) error {
	switch src.Visibility {
	case "", domain.KBVisibilityAll, domain.KBVisibilityRestricted:
	default:
		return bad("visibility must be all or restricted")
	}
	switch src.Status {
	case "", domain.KBActive, domain.KBDraft, domain.KBArchived:
	default:
		return bad("status must be active, draft or archived")
	}
	if src.ValidUntil != "" {
		if _, err := time.Parse("2006-01-02", src.ValidUntil); err != nil {
			return bad("valid_until must be YYYY-MM-DD")
		}
	}
	for _, id := range src.DepartmentIDs {
		if err := s.exists(r, "department", id); err != nil {
			return err
		}
	}
	return nil
}
