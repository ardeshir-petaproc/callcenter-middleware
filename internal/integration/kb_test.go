package integration

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"testing"

	"callcenter-middleware/internal/domain"
)

func (s *stack) upload(token, name string, meta any, data []byte) (int, domain.KnowledgeSource) {
	s.t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	m, _ := json.Marshal(meta)
	_ = mw.WriteField("meta", string(m))
	fw, _ := mw.CreateFormFile("file", name)
	_, _ = fw.Write(data)
	_ = mw.Close()
	req, _ := http.NewRequest("POST", s.api.URL+"/api/v1/kb/sources", &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		s.t.Fatal(err)
	}
	defer resp.Body.Close()
	var src domain.KnowledgeSource
	_ = json.NewDecoder(resp.Body).Decode(&src)
	return resp.StatusCode, src
}

func TestKnowledge_ScopedSearchAndAudit(t *testing.T) {
	s := newStack(t, 40)
	admin := s.login("admin", "admin123")
	var depts []domain.Department
	s.call("GET", "/departments", admin, nil, &depts)
	finance := depts[2]

	code, refund := s.upload(admin, "refunds.md", map[string]any{"title": "Refund policy", "category": "policy",
		"ai_enabled": true, "tags": []string{"refund"}}, []byte("# Refunds\nبازپرداخت وجه ظرف هفت روز کاری انجام می‌شود."))
	if code != 200 || refund.ID == 0 || refund.StorageKey != "" {
		t.Fatalf("upload: %d %+v", code, refund)
	}
	var secret domain.KnowledgeSource
	s.call("POST", "/kb/sources", admin, map[string]any{"title": "Payroll escalation", "content": "Payroll bonus refunds go to the CFO.",
		"visibility": "restricted", "department_ids": []int64{finance.ID}}, &secret)
	var stale domain.KnowledgeSource
	s.call("POST", "/kb/sources", admin, map[string]any{"title": "Old refund promo", "content": "Refunds within 30 days during promo.",
		"ai_enabled": true, "valid_until": "2020-01-01"}, &stale)

	// Persian full-text search finds the uploaded file.
	alice := s.login("alice", "agent123")
	var hits []domain.KBHit
	s.call("GET", "/kb/search?q=بازپرداخت", alice, nil, &hits)
	if len(hits) != 1 || hits[0].SourceID != refund.ID {
		t.Fatalf("persian search: %+v", hits)
	}
	// Restricted finance material and expired guidance stay hidden from agents.
	s.call("GET", "/kb/search?q=refunds", alice, nil, &hits)
	for _, h := range hits {
		if h.SourceID == secret.ID || h.SourceID == stale.ID {
			t.Fatalf("alice must not see %+v", h)
		}
	}
	if code := s.call("GET", "/kb/sources/"+itoa(secret.ID), alice, nil, nil); code != 403 {
		t.Fatalf("restricted source: %d", code)
	}
	// The AI never retrieves expired guidance and gets passage text.
	s.call("POST", "/kb/search/ai", admin, map[string]any{"q": "refunds"}, &hits)
	for _, h := range hits {
		if h.SourceID == stale.ID || h.SourceID == secret.ID {
			t.Fatalf("AI must not see %+v", h)
		}
	}
	if len(hits) == 0 || hits[0].Text == "" {
		t.Fatalf("AI search should return passages: %+v", hits)
	}
	// Agents cannot manage sources; downloads decrypt and are audited.
	if code := s.call("POST", "/kb/sources", alice, map[string]any{"title": "x", "content": "y"}, nil); code != 403 {
		t.Fatalf("agent create: %d", code)
	}
	req, _ := http.NewRequest("GET", s.api.URL+"/api/v1/kb/sources/"+itoa(refund.ID)+"/download", nil)
	req.Header.Set("Authorization", "Bearer "+alice)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if !bytes.Contains(body, []byte("بازپرداخت")) {
		t.Fatalf("download: %q", body)
	}
	var audit []domain.AuditEntry
	s.call("GET", "/audit?action=kb.download", admin, nil, &audit)
	if len(audit) != 1 || audit[0].Username != "alice" {
		t.Fatalf("download audit: %+v", audit)
	}
	// Another tenant can neither read nor find this organization's knowledge.
	other := s.loginTenant("admin", "admin123", "parsian")
	if code := s.call("GET", "/kb/sources/"+itoa(refund.ID), other, nil, nil); code != 404 {
		t.Fatalf("cross-tenant read: %d", code)
	}
	s.call("GET", "/kb/search?q=بازپرداخت", other, nil, &hits)
	if len(hits) != 0 {
		t.Fatalf("cross-tenant search: %+v", hits)
	}
	// Deleting removes the source from search.
	s.call("DELETE", "/kb/sources/"+itoa(refund.ID), admin, nil, nil)
	s.call("GET", "/kb/search?q=بازپرداخت", admin, nil, &hits)
	if len(hits) != 0 {
		t.Fatalf("deleted source still found: %+v", hits)
	}
}
