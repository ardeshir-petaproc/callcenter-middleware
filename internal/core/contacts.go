package core

import (
	"context"
	"strings"

	"callcenter-middleware/internal/domain"
	"callcenter-middleware/internal/store"
)

// NormalizePhone strips formatting so numbers compare reliably.
func NormalizePhone(n string) string {
	var b strings.Builder
	for _, r := range n {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// phoneMatch compares numbers ignoring formatting and international prefixes
// by matching the trailing 9 digits when both numbers are long enough.
func phoneMatch(a, b string) bool {
	a, b = NormalizePhone(a), NormalizePhone(b)
	if a == "" || b == "" {
		return false
	}
	if a == b {
		return true
	}
	const tail = 9
	if len(a) >= tail && len(b) >= tail {
		return a[len(a)-tail:] == b[len(b)-tail:]
	}
	return false
}

// FindContact looks up a CRM contact by phone number.
func FindContact(ctx context.Context, st *store.Store, number string) *domain.Contact {
	contacts, err := st.Contacts.List(ctx)
	if err != nil {
		return nil
	}
	for i := range contacts {
		if phoneMatch(contacts[i].Phone, number) {
			return &contacts[i]
		}
	}
	return nil
}
