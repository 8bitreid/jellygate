package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rmewborne/jellygate/internal/domain"
)

// SessionStore is a Postgres-backed implementation of domain.SessionStore.
type SessionStore struct {
	pool *pgxpool.Pool
}

// NewSessionStore returns a SessionStore backed by the given pool.
func NewSessionStore(pool *pgxpool.Pool) *SessionStore {
	return &SessionStore{pool: pool}
}

// Create persists a new session.
func (s *SessionStore) Create(ctx context.Context, sess domain.Session) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO sessions (token, username, jellyfin_token, expires_at)
		VALUES ($1, $2, $3, $4)`,
		sess.Token, sess.Username, sess.JellyfinToken, sess.ExpiresAt,
	)
	if err != nil {
		return fmt.Errorf("store.SessionStore.Create: %w", err)
	}
	return nil
}

// Get returns the session for the given token.
// Returns domain.ErrSessionNotFound if no session matches,
// domain.ErrSessionExpired if the session exists but is past its expiry.
func (s *SessionStore) Get(ctx context.Context, token string) (domain.Session, error) {
	var sess domain.Session
	err := s.pool.QueryRow(ctx, `
		SELECT token, username, jellyfin_token, expires_at FROM sessions WHERE token = $1`, token).
		Scan(&sess.Token, &sess.Username, &sess.JellyfinToken, &sess.ExpiresAt)

	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Session{}, domain.ErrSessionNotFound
	}
	if err != nil {
		return domain.Session{}, fmt.Errorf("store.SessionStore.Get: %w", err)
	}
	if sess.IsExpired() {
		return domain.Session{}, domain.ErrSessionExpired
	}
	return sess, nil
}

// Delete removes a session by token.
func (s *SessionStore) Delete(ctx context.Context, token string) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM sessions WHERE token = $1`, token)
	if err != nil {
		return fmt.Errorf("store.SessionStore.Delete: %w", err)
	}
	return nil
}

// Purge removes all expired sessions.
func (s *SessionStore) Purge(ctx context.Context) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM sessions WHERE expires_at < now()`)
	if err != nil {
		return fmt.Errorf("store.SessionStore.Purge: %w", err)
	}
	return nil
}
