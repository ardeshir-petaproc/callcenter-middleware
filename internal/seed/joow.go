package seed

import (
	"context"

	"callcenter-middleware/internal/joow"
)

// DemoJoowAccounts registers the demo employees' Joow accounts with the
// local Joow provider, plus one person who is not an employee yet.
func DemoJoowAccounts(ctx context.Context, upsert func(context.Context, joow.Account) error) error {
	accounts := []joow.Account{
		{ID: "j-alice", Name: "Alice Moreno", Email: "alice@example.com", Phone: "+15550201"},
		{ID: "j-bob", Name: "Bob Chen", Email: "bob@example.com", Phone: "+15550202"},
		{ID: "j-carol", Name: "Carol Haddad", Email: "carol@example.com", Phone: "+15550203"},
		{ID: "j-dave", Name: "Dave Okafor", Email: "dave@example.com", Phone: "+15550204"},
		{ID: "j-sara", Name: "Sara Lindqvist", Email: "sara@example.com", Phone: "+15550205"},
		{ID: "j-reza", Name: "رضا احمدی", Email: "reza@example.ir", Phone: "+989120000001"},
		{ID: "j-nima", Name: "نیما رستمی", Email: "nima@example.ir", Phone: "+989120000002"},
	}
	for _, a := range accounts {
		if err := upsert(ctx, a); err != nil {
			return err
		}
	}
	return nil
}
