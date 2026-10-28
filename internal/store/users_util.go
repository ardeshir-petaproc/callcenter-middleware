package store

import (
	"callcenter-middleware/internal/domain"
	"context"
	"fmt"
	"strings"
)

func first(users []domain.User, err error) (*domain.User, error) {
	if err != nil {
		return nil, err
	}
	if len(users) == 0 {
		return nil, ErrNotFound
	}
	return &users[0], nil
}

// CountUsers returns the number of users in the scope.
func (s *Store) CountUsers(ctx context.Context) (int, error) {
	users, err := s.ListUsers(ctx)
	return len(users), err
}

func wrapConflict(err error) error {
	if strings.Contains(err.Error(), "UNIQUE") {
		return fmt.Errorf("%w: %v", ErrConflict, err)
	}
	return err
}

func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func nonNil(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}
