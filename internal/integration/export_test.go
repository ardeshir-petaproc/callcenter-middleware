package integration

import (
	"archive/zip"
	"bytes"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestExport_OrganizationZipIsScopedAndRedacted(t *testing.T) {
	s := newStack(t, 40)
	admin := s.login("admin", "admin123")
	s.call("POST", "/webhooks", admin, map[string]any{"url": "http://example.test/hook", "secret": "s3cret",
		"events": []string{"call.ended"}, "enabled": true}, nil)
	get := func(tok string) (*http.Response, []byte) {
		req, _ := http.NewRequest("GET", s.api.URL+"/api/v1/export/organization.zip", nil)
		req.Header.Set("Authorization", "Bearer "+tok)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		b, _ := io.ReadAll(resp.Body)
		return resp, b
	}
	if resp, _ := get(s.login("alice", "agent123")); resp.StatusCode != 403 {
		t.Fatalf("agent export: %d", resp.StatusCode)
	}
	resp, body := get(admin)
	zr, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	if resp.StatusCode != 200 || err != nil {
		t.Fatalf("export: %d %v", resp.StatusCode, err)
	}
	files := map[string]string{}
	for _, f := range zr.File {
		rc, _ := f.Open()
		b, _ := io.ReadAll(rc)
		rc.Close()
		files[f.Name] = string(b)
	}
	for _, name := range []string{"users.json", "calls.json", "tickets.json", "audit.json", "config/queue.json", "config/webhook.json"} {
		if _, ok := files[name]; !ok {
			t.Fatalf("missing %s in %v", name, len(files))
		}
	}
	if strings.Contains(files["config/webhook.json"], "s3cret") || strings.Contains(files["users.json"], "password") {
		t.Fatal("secrets leaked into the export")
	}
	if strings.Contains(files["users.json"], "reza") || !strings.Contains(files["users.json"], "alice") {
		t.Fatal("export must contain only this organization's people")
	}
	if !strings.Contains(files["audit.json"], "organization.export") {
		t.Fatal("export not audited")
	}
}
