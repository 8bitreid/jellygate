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

const (
	keyJellyfinAdminToken = "jellyfin_admin_token"
	keyJellyfinURL        = "jellyfin_url"
	keyDiscordWebhookURL  = "discord_webhook_url"
	keySeerrURL           = "seerr_url"
)

// GetJellyfinAdminToken returns the stored Jellyfin admin access token.
// Returns domain.ErrSettingNotFound if the admin has not logged in yet.
func (s *SettingsStore) GetJellyfinAdminToken(ctx context.Context) (string, error) {
	return s.get(ctx, keyJellyfinAdminToken, "GetJellyfinAdminToken")
}

// SetJellyfinAdminToken upserts the Jellyfin admin access token.
func (s *SettingsStore) SetJellyfinAdminToken(ctx context.Context, token string) error {
	return s.set(ctx, keyJellyfinAdminToken, token, "SetJellyfinAdminToken")
}

func (s *SettingsStore) GetJellyfinURL(ctx context.Context) (string, error) {
	return s.get(ctx, keyJellyfinURL, "GetJellyfinURL")
}

func (s *SettingsStore) SetJellyfinURL(ctx context.Context, url string) error {
	return s.set(ctx, keyJellyfinURL, url, "SetJellyfinURL")
}

func (s *SettingsStore) GetDiscordWebhookURL(ctx context.Context) (string, error) {
	return s.get(ctx, keyDiscordWebhookURL, "GetDiscordWebhookURL")
}

func (s *SettingsStore) SetDiscordWebhookURL(ctx context.Context, url string) error {
	return s.set(ctx, keyDiscordWebhookURL, url, "SetDiscordWebhookURL")
}

func (s *SettingsStore) GetSeerrURL(ctx context.Context) (string, error) {
	return s.get(ctx, keySeerrURL, "GetSeerrURL")
}

func (s *SettingsStore) SetSeerrURL(ctx context.Context, url string) error {
	return s.set(ctx, keySeerrURL, url, "SetSeerrURL")
}

func (s *SettingsStore) get(ctx context.Context, key, caller string) (string, error) {
	var value string
	err := s.pool.QueryRow(ctx, `SELECT value FROM settings WHERE key = $1`, key).Scan(&value)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", domain.ErrSettingNotFound
	}
	if err != nil {
		return "", fmt.Errorf("store.SettingsStore.%s: %w", caller, err)
	}
	return value, nil
}

func (s *SettingsStore) set(ctx context.Context, key, value, caller string) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO settings (key, value) VALUES ($1, $2)
		 ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value`,
		key, value,
	)
	if err != nil {
		return fmt.Errorf("store.SettingsStore.%s: %w", caller, err)
	}
	return nil
}
