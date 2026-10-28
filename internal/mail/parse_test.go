package mail

import (
	"strings"
	"testing"

	"callcenter-middleware/internal/domain"
)

func TestParse_MultipartHTMLAndReferences(t *testing.T) {
	raw := strings.ReplaceAll(`From: "Sara" <Sara@Shop.IR>
Subject: =?UTF-8?B?2LPZgdin2LHYtA==?=
Message-ID: <b@shop.ir>
In-Reply-To: <a@shop.ir>
References: <root@shop.ir> <a@shop.ir>
MIME-Version: 1.0
Content-Type: multipart/mixed; boundary=XX

--XX
Content-Type: text/html; charset=utf-8

<p>Order <b>42</b> is late</p><blockquote>old</blockquote>
--XX
Content-Type: application/pdf
Content-Disposition: attachment; filename="invoice.pdf"

%PDF-1.4
--XX--
`, "\n", "\r\n")
	p, err := Parse([]byte(raw))
	if err != nil {
		t.Fatal(err)
	}
	if p.FromAddr != "sara@shop.ir" || p.Subject != "سفارش" || p.MessageID != "b@shop.ir" {
		t.Fatalf("headers: %+v", p)
	}
	if len(p.References) != 3 || p.References[0] != "a@shop.ir" {
		t.Fatalf("references: %v", p.References)
	}
	if !strings.Contains(p.Text, "Order 42 is late") || len(p.Attachments) != 1 || p.Attachments[0] != "invoice.pdf" {
		t.Fatalf("body: %q %v", p.Text, p.Attachments)
	}
}

func TestMatches_Filters(t *testing.T) {
	c := &domain.EmailConnector{FromFilter: []string{"@customer.com", "vip@gmail.com"}, SubjectFilter: "order"}
	cases := map[string]bool{"a@customer.com|Order 1": true, "vip@gmail.com|my order": true,
		"x@gmail.com|order": false, "a@customer.com|hello": false, "a@evilcustomer.com|order": false}
	for in, want := range cases {
		parts := strings.SplitN(in, "|", 2)
		if got := Matches(c, &Parsed{FromAddr: parts[0], Subject: parts[1]}); got != want {
			t.Errorf("%s: got %v", in, got)
		}
	}
}
