package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rmewborne/jellygate/internal/domain"
)

// SettingsStore is a Postgres-backed domain.SettingsStore.
type SettingsStore struct {
	pool *pgxpool.Pool
}

// NewSettingsStore returns a SettingsStore backed by the given pool.
func NewSettingsStore(pool *pgxpool.Pool) *SettingsStore {
	return &SettingsStore{pool: pool}
}

const keyJellyfinAdminToken = "jellyfin_admin_token"

// GetJellyfinAdminToken returns the stored Jellyfin admin access token.
// Returns domain.ErrSettingNotFound if the admin has not logged in yet.
func (s *SettingsStore) GetJellyfinAdminToken(ctx context.Context) (string, error) {
	var value string
	err := s.pool.QueryRow(ctx,
		`SELECT value FROM settings WHERE key = $1`, keyJellyfinAdminToken,
	).Scan(&value)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", domain.ErrSettingNotFound
	}
	if err != nil {
		return "", fmt.Errorf("store.SettingsStore.GetJellyfinAdminToken: %w", err)
	}
	return value, nil
}

// SetJellyfinAdminToken upserts the Jellyfin admin access token.
// Called after every successful admin login to keep the token current.
func (s *SettingsStore) SetJellyfinAdminToken(ctx context.Context, token string) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO settings (key, value) VALUES ($1, $2)
		 ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value`,
		keyJellyfinAdminToken, token,
	)
	if err != nil {
		return fmt.Errorf("store.SettingsStore.SetJellyfinAdminToken: %w", err)
	}
	return nil
}
