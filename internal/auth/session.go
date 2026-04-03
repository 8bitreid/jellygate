package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/http"
	"time"

	"github.com/rmewborne/jellygate/internal/domain"
)

const (
	SessionCookieName = "jg_session"
	sessionTTL        = 24 * time.Hour
)

// contextKey is an unexported type for context values set by this package.
type contextKey int

const sessionKey contextKey = 0

// Manager handles session creation, retrieval, and deletion.
type Manager struct {
	store  domain.SessionStore
	secure bool // set Secure flag on cookies (false only in tests)
}

// NewManager returns a session Manager backed by the given store.
// secure should be true in production (served over HTTPS via Traefik).
func NewManager(store domain.SessionStore, secure bool) *Manager {
	return &Manager{store: store, secure: secure}
}

// Create generates a new session for username, persists it, and writes
// the session cookie to w.
func (m *Manager) Create(ctx context.Context, w http.ResponseWriter, username string) error {
	token, err := generateToken()
	if err != nil {
		return fmt.Errorf("auth.Manager.Create: %w", err)
	}

	sess := domain.Session{
		Token:     token,
		Username:  username,
		ExpiresAt: time.Now().Add(sessionTTL),
	}
	if err := m.store.Create(ctx, sess); err != nil {
		return fmt.Errorf("auth.Manager.Create: %w", err)
	}

	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    token,
		Path:     "/",
		Expires:  sess.ExpiresAt,
		HttpOnly: true,
		Secure:   m.secure,
		SameSite: http.SameSiteStrictMode,
	})
	return nil
}

// Get retrieves the session from the request cookie.
// Returns domain.ErrSessionNotFound or domain.ErrSessionExpired if invalid.
func (m *Manager) Get(ctx context.Context, r *http.Request) (domain.Session, error) {
	cookie, err := r.Cookie(SessionCookieName)
	if err != nil {
		return domain.Session{}, domain.ErrSessionNotFound
	}
	return m.store.Get(ctx, cookie.Value)
}

// Delete removes the session and clears the cookie.
func (m *Manager) Delete(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	cookie, err := r.Cookie(SessionCookieName)
	if err != nil {
		return nil // already gone
	}
	if err := m.store.Delete(ctx, cookie.Value); err != nil {
		return fmt.Errorf("auth.Manager.Delete: %w", err)
	}
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   m.secure,
		SameSite: http.SameSiteStrictMode,
	})
	return nil
}

// FromContext retrieves the session stored by RequireSession middleware.
func FromContext(ctx context.Context) (domain.Session, bool) {
	s, ok := ctx.Value(sessionKey).(domain.Session)
	return s, ok
}

// WithSessionCtx returns a new context carrying the session.
// Called by middleware after successful session validation.
func WithSessionCtx(ctx context.Context, s domain.Session) context.Context {
	return context.WithValue(ctx, sessionKey, s)
}

// generateToken returns a cryptographically random 32-byte base64url token.
func generateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generateToken: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
