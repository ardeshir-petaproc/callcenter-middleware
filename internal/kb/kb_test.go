package kb

import (
	"archive/zip"
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"callcenter-middleware/internal/domain"
	"callcenter-middleware/internal/secure"
	"callcenter-middleware/internal/store"
)

func docx(t *testing.T, text string) []byte {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, _ := zw.Create("word/document.xml")
	_, _ = w.Write([]byte(`<w:document><w:body><w:p><w:r><w:t>` + text + `</w:t></w:r></w:p></w:body></w:document>`))
	zw.Close()
	return buf.Bytes()
}

func TestSaveSearchAndPermissions(t *testing.T) {
	st, _ := store.Open(":memory:")
	defer st.Close()
	dir := t.TempDir()
	svc := &Service{Store: st, Box: secure.New("k"), Dir: dir}
	ctx := store.ForTenant(context.Background(), 1)

	policy := &domain.KnowledgeSource{Title: "سیاست بازپرداخت", Category: "billing", AIEnabled: true}
	if err := svc.Save(ctx, policy, "refund.docx", docx(t, "بازپرداخت وجه تا ۳۰ روز پس از خرید ممکن است. Refunds within 30 days.")); err != nil {
		t.Fatal(err)
	}
	hr := &domain.KnowledgeSource{Title: "Salary bands", Category: "hr", Visibility: domain.KBVisibilityRestricted, AllowedRoles: []string{"admin"}}
	if err := svc.Save(ctx, hr, "", []byte("Salary bands for 2026: refunds are not related.")); err != nil {
		t.Fatal(err)
	}
	old := &domain.KnowledgeSource{Title: "Old refund policy", AIEnabled: true, ValidUntil: "2020-01-01"}
	if err := svc.Save(ctx, old, "old.txt", []byte("Refunds within 7 days only.")); err != nil {
		t.Fatal(err)
	}
	// Files are encrypted on disk.
	raw, _ := os.ReadFile(filepath.Join(dir, "kb", "1", policy.StorageKey+".bin"))
	if bytes.Contains(raw, []byte("Refunds")) {
		t.Fatal("knowledge file stored in plaintext")
	}
	agent := domain.KBAccess{Role: domain.RoleAgent}
	hits, _ := svc.Search(ctx, 1, "بازپرداخت", agent, 5)
	if len(hits) != 1 || hits[0].SourceID != policy.ID {
		t.Fatalf("persian search failed: %+v", hits)
	}
	hits, _ = svc.Search(ctx, 1, "refunds", agent, 5)
	for _, h := range hits {
		if h.SourceID == hr.ID || h.SourceID == old.ID {
			t.Fatalf("agent must not see restricted or expired sources: %+v", hits)
		}
	}
	ai := domain.KBAccess{AI: true}
	hits, _ = svc.Search(ctx, 1, "refunds", ai, 5)
	if len(hits) != 1 || hits[0].SourceID != policy.ID || !strings.Contains(hits[0].Text, "30") {
		t.Fatalf("AI must only get the valid, AI-enabled policy: %+v", hits)
	}
	if hits, _ = svc.Search(ctx, 1, "refunds", domain.KBAccess{AI: true, Categories: []string{"sales"}}, 5); len(hits) != 0 {
		t.Fatal("AI category scope not applied")
	}
	if hits, _ = svc.Search(ctx, 2, "refunds", domain.KBAccess{Manager: true}, 5); len(hits) != 0 {
		t.Fatal("other tenants must not find these sources")
	}
	admin := domain.KBAccess{Role: domain.RoleAdmin}
	if hits, _ = svc.Search(ctx, 1, "salary", admin, 5); len(hits) != 1 {
		t.Fatal("allowed role should see restricted source")
	}
	if err := svc.Delete(ctx, policy.ID); err != nil {
		t.Fatal(err)
	}
	if hits, _ = svc.Search(ctx, 1, "بازپرداخت", agent, 5); len(hits) != 0 {
		t.Fatal("deleted source still indexed")
	}
}

func TestExtractFormats(t *testing.T) {
	if s, _ := Extract("a.html", []byte("<p>Hello&nbsp;<b>world</b></p><script>x()</script>")); !strings.Contains(s, "Hello") || strings.Contains(s, "x()") {
		t.Fatalf("html: %q", s)
	}
	if _, err := Extract("a.exe", []byte{0}); err != ErrUnsupported {
		t.Fatal("expected unsupported")
	}
	chunks := Chunk(strings.Repeat("word ", 1000), 300)
	if len(chunks) < 10 {
		t.Fatalf("chunking: %d", len(chunks))
	}
}
