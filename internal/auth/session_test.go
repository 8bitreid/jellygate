package auth_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/rmewborne/jellygate/internal/auth"
	"github.com/rmewborne/jellygate/internal/domain"
)

// stubSessionStore is an in-memory SessionStore for tests.
type stubSessionStore struct {
	sessions map[string]domain.Session
}

func newStubStore() *stubSessionStore {
	return &stubSessionStore{sessions: map[string]domain.Session{}}
}

func (s *stubSessionStore) Create(_ context.Context, sess domain.Session) error {
	s.sessions[sess.Token] = sess
	return nil
}

func (s *stubSessionStore) Get(_ context.Context, token string) (domain.Session, error) {
	sess, ok := s.sessions[token]
	if !ok {
		return domain.Session{}, domain.ErrSessionNotFound
	}
	if sess.IsExpired() {
		return domain.Session{}, domain.ErrSessionExpired
	}
	return sess, nil
}

func (s *stubSessionStore) Delete(_ context.Context, token string) error {
	delete(s.sessions, token)
	return nil
}

func (s *stubSessionStore) Purge(_ context.Context) error {
	for k, v := range s.sessions {
		if v.IsExpired() {
			delete(s.sessions, k)
		}
	}
	return nil
}

func TestManager_CreateAndGet(t *testing.T) {
	mgr := auth.NewManager(newStubStore(), false)
	w := httptest.NewRecorder()

	if err := mgr.Create(context.Background(), w, "admin", "jf-tok"); err != nil {
		t.Fatalf("Create: %v", err)
	}

	// Extract the session cookie from the response.
	resp := w.Result()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	for _, c := range resp.Cookies() {
		req.AddCookie(c)
	}

	sess, err := mgr.Get(context.Background(), req)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if sess.Username != "admin" {
		t.Errorf("want username %q, got %q", "admin", sess.Username)
	}
}

func TestManager_Get_NoCookie(t *testing.T) {
	mgr := auth.NewManager(newStubStore(), false)
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	_, err := mgr.Get(context.Background(), req)
	if !errors.Is(err, domain.ErrSessionNotFound) {
		t.Errorf("want ErrSessionNotFound, got %v", err)
	}
}

func TestManager_Delete(t *testing.T) {
	store := newStubStore()
	mgr := auth.NewManager(store, false)

	w := httptest.NewRecorder()
	mgr.Create(context.Background(), w, "admin", "jf-tok")

	resp := w.Result()
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	for _, c := range resp.Cookies() {
		req.AddCookie(c)
	}

	w2 := httptest.NewRecorder()
	if err := mgr.Delete(context.Background(), w2, req); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	// Cookie should be cleared (MaxAge=-1).
	var found bool
	for _, c := range w2.Result().Cookies() {
		if c.Name == auth.SessionCookieName && c.MaxAge == -1 {
			found = true
		}
	}
	if !found {
		t.Error("expected cleared session cookie in response")
	}
}

func TestManager_ExpiredSession(t *testing.T) {
	store := newStubStore()
	// Manually insert an expired session.
	store.sessions["expired-tok"] = domain.Session{
		Token:     "expired-tok",
		Username:  "admin",
		ExpiresAt: time.Now().Add(-time.Minute),
	}

	mgr := auth.NewManager(store, false)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: auth.SessionCookieName, Value: "expired-tok"})

	_, err := mgr.Get(context.Background(), req)
	if !errors.Is(err, domain.ErrSessionExpired) {
		t.Errorf("want ErrSessionExpired, got %v", err)
	}
}

func TestFromContext_RoundTrip(t *testing.T) {
	sess := domain.Session{Token: "t", Username: "admin", ExpiresAt: time.Now().Add(time.Hour)}
	ctx := auth.WithSessionCtx(context.Background(), sess)

	got, ok := auth.FromContext(ctx)
	if !ok {
		t.Fatal("expected session in context")
	}
	if got.Username != "admin" {
		t.Errorf("want username %q, got %q", "admin", got.Username)
	}
}
