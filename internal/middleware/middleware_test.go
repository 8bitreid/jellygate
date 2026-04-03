package middleware_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/rmewborne/jellygate/internal/auth"
	"github.com/rmewborne/jellygate/internal/domain"
	"github.com/rmewborne/jellygate/internal/middleware"
)

// --- stub session store (reused from auth tests pattern) ---

type stubStore struct {
	sessions map[string]domain.Session
}

func newStub() *stubStore { return &stubStore{sessions: map[string]domain.Session{}} }

func (s *stubStore) Create(_ context.Context, sess domain.Session) error {
	s.sessions[sess.Token] = sess
	return nil
}
func (s *stubStore) Get(_ context.Context, token string) (domain.Session, error) {
	sess, ok := s.sessions[token]
	if !ok {
		return domain.Session{}, domain.ErrSessionNotFound
	}
	if sess.IsExpired() {
		return domain.Session{}, domain.ErrSessionExpired
	}
	return sess, nil
}
func (s *stubStore) Delete(_ context.Context, token string) error { delete(s.sessions, token); return nil }
func (s *stubStore) Purge(_ context.Context) error                { return nil }

// --- RequireSession ---

func TestRequireSession_Authenticated(t *testing.T) {
	store := newStub()
	store.sessions["valid-tok"] = domain.Session{
		Token:     "valid-tok",
		Username:  "admin",
		ExpiresAt: time.Now().Add(time.Hour),
	}

	mgr := auth.NewManager(store, false)
	var capturedUser string
	handler := middleware.RequireSession(mgr)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if sess, ok := auth.FromContext(r.Context()); ok {
			capturedUser = sess.Username
		}
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	req.AddCookie(&http.Cookie{Name: auth.SessionCookieName, Value: "valid-tok"})
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("want 200, got %d", rec.Code)
	}
	if capturedUser != "admin" {
		t.Errorf("want username in context %q, got %q", "admin", capturedUser)
	}
}

func TestRequireSession_Unauthenticated_Redirects(t *testing.T) {
	mgr := auth.NewManager(newStub(), false)
	handler := middleware.RequireSession(mgr)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Errorf("want redirect 303, got %d", rec.Code)
	}
	if loc := rec.Header().Get("Location"); loc != "/admin/login" {
		t.Errorf("want redirect to /admin/login, got %q", loc)
	}
}

// --- RateLimit ---

func TestRateLimit_AllowsUnderLimit(t *testing.T) {
	handler := middleware.RateLimit(3, time.Minute, false)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	for i := 0; i < 3; i++ {
		req := httptest.NewRequest(http.MethodPost, "/invite/tok", nil)
		req.RemoteAddr = "1.2.3.4:5000"
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Errorf("request %d: want 200, got %d", i+1, rec.Code)
		}
	}
}

func TestRateLimit_BlocksOverLimit(t *testing.T) {
	handler := middleware.RateLimit(2, time.Minute, false)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	for i := 0; i < 3; i++ {
		req := httptest.NewRequest(http.MethodPost, "/invite/tok", nil)
		req.RemoteAddr = "5.6.7.8:1234"
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if i < 2 && rec.Code != http.StatusOK {
			t.Errorf("request %d: want 200, got %d", i+1, rec.Code)
		}
		if i == 2 && rec.Code != http.StatusTooManyRequests {
			t.Errorf("request %d: want 429, got %d", i+1, rec.Code)
		}
	}
}

// --- SecureHeaders ---

func TestSecureHeaders(t *testing.T) {
	handler := middleware.SecureHeaders(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	checks := map[string]string{
		"X-Frame-Options":        "DENY",
		"X-Content-Type-Options": "nosniff",
	}
	for header, want := range checks {
		if got := rec.Header().Get(header); got != want {
			t.Errorf("%s: want %q, got %q", header, want, got)
		}
	}
	if rec.Header().Get("Content-Security-Policy") == "" {
		t.Error("expected Content-Security-Policy header")
	}
}
