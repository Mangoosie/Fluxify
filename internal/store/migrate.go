package store

import (
	"context"
	_ "embed"
	"fmt"
)

//go:embed schema.sql
var schema string

func (s *Store) migrate(ctx context.Context) error {
	if _, err := s.db.ExecContext(ctx, schema); err != nil {
		return fmt.Errorf("apply schema: %w", err)
	}
	return nil
}
