package store

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rmewborne/jellygate/internal/domain"
)

// RegistrationStore is a Postgres-backed domain.RegistrationStore.
type RegistrationStore struct {
	pool *pgxpool.Pool
}

// NewRegistrationStore returns a RegistrationStore backed by the given pool.
func NewRegistrationStore(pool *pgxpool.Pool) *RegistrationStore {
	return &RegistrationStore{pool: pool}
}

// Create persists a registration record.
func (s *RegistrationStore) Create(ctx context.Context, reg domain.Registration) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO registrations (id, invite_id, jf_user_id, username, registered_at)
		 VALUES ($1, $2, $3, $4, $5)`,
		reg.ID, reg.InviteID, reg.JellyfinUID, reg.Username, reg.RegisteredAt,
	)
	if err != nil {
		return fmt.Errorf("store.RegistrationStore.Create: %w", err)
	}
	return nil
}
