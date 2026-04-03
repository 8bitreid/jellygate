package handler_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/rmewborne/jellygate/internal/auth"
	"github.com/rmewborne/jellygate/internal/domain"
	"github.com/rmewborne/jellygate/internal/handler"
	"github.com/rmewborne/jellygate/internal/middleware"
)

// --- stubs ---

type stubInviteStore struct {
	invites map[string]domain.Invite
}

func newInviteStore() *stubInviteStore {
	return &stubInviteStore{invites: map[string]domain.Invite{}}
}
func (s *stubInviteStore) Create(_ context.Context, inv domain.Invite) error {
	s.invites[inv.ID] = inv; return nil
}
func (s *stubInviteStore) GetByToken(_ context.Context, token string) (domain.Invite, error) {
	for _, inv := range s.invites {
		if inv.Token == token {
			return inv, nil
		}
	}
	return domain.Invite{}, domain.ErrInviteNotFound
}
func (s *stubInviteStore) List(_ context.Context) ([]domain.Invite, error) {
	out := make([]domain.Invite, 0, len(s.invites))
	for _, inv := range s.invites {
		out = append(out, inv)
	}
	return out, nil
}
func (s *stubInviteStore) Revoke(_ context.Context, id string) error {
	inv, ok := s.invites[id]
	if !ok {
		return domain.ErrInviteNotFound
	}
	inv.Revoked = true
	s.invites[id] = inv
	return nil
}
func (s *stubInviteStore) IncrementUse(_ context.Context, id string) error {
	inv, ok := s.invites[id]
	if !ok {
		return domain.ErrInviteNotFound
	}
	inv.UseCount++
	s.invites[id] = inv
	return nil
}

type stubSessionStore struct{ sessions map[string]domain.Session }

func newSessionStore() *stubSessionStore {
	return &stubSessionStore{sessions: map[string]domain.Session{}}
}
func (s *stubSessionStore) Create(_ context.Context, sess domain.Session) error {
	s.sessions[sess.Token] = sess; return nil
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
	delete(s.sessions, token); return nil
}
func (s *stubSessionStore) Purge(_ context.Context) error { return nil }

type stubJellyfinClient struct{ token string; err error }

func (j *stubJellyfinClient) Authenticate(_ context.Context, _, _ string) (string, error) {
	return j.token, j.err
}
func (j *stubJellyfinClient) ListLibraries(_ context.Context, _ string) ([]domain.Library, error) {
	return []domain.Library{{ID: "lib-1", Name: "Movies"}}, nil
}
func (j *stubJellyfinClient) CreateUser(_ context.Context, _, _, _ string) (string, error) {
	return "new-user-id", nil
}
func (j *stubJellyfinClient) SetLibraryAccess(_ context.Context, _, _ string, _ []string) error {
	return nil
}

// --- helpers ---

func newAdmin(t *testing.T, jf domain.JellyfinClient) (*handler.Admin, *stubSessionStore, *stubInviteStore) {
	t.Helper()
	ss := newSessionStore()
	is := newInviteStore()
	mgr := auth.NewManager(ss, false)
	adm, err := handler.NewAdmin(mgr, is, jf, "http://localhost:8080", false)
	if err != nil {
		t.Fatalf("NewAdmin: %v", err)
	}
	return adm, ss, is
}

// authenticatedRequest creates a request with a valid session cookie.
func authenticatedRequest(t *testing.T, ss *stubSessionStore, method, path string, body *strings.Reader) *http.Request {
	t.Helper()
	tok := "test-session-tok"
	ss.sessions[tok] = domain.Session{
		Token:         tok,
		Username:      "admin",
		JellyfinToken: "jf-tok",
		ExpiresAt:     time.Now().Add(time.Hour),
	}
	var req *http.Request
	if body != nil {
		req = httptest.NewRequest(method, path, body)
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	} else {
		req = httptest.NewRequest(method, path, nil)
	}
	req.AddCookie(&http.Cookie{Name: auth.SessionCookieName, Value: tok})
	return req
}

// withSession injects a session into the request context (simulates RequireSession middleware).
func withSession(req *http.Request, sess domain.Session) *http.Request {
	return req.WithContext(auth.WithSessionCtx(req.Context(), sess))
}

// --- tests ---

func TestHandleLoginForm_Renders(t *testing.T) {
	adm, _, _ := newAdmin(t, &stubJellyfinClient{token: "tok"})
	req := httptest.NewRequest(http.MethodGet, "/admin/login", nil)
	rec := httptest.NewRecorder()
	adm.HandleLoginForm(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("want 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "sign in") {
		t.Error("expected login page content")
	}
}

func TestHandleLogin_Success(t *testing.T) {
	adm, ss, _ := newAdmin(t, &stubJellyfinClient{token: "jf-tok-abc"})

	// First get the CSRF token via GET.
	w0 := httptest.NewRecorder()
	adm.HandleLoginForm(w0, httptest.NewRequest(http.MethodGet, "/admin/login", nil))
	csrfCookie := w0.Result().Cookies()

	form := url.Values{"username": {"admin"}, "password": {"pass"}}
	for _, c := range csrfCookie {
		if c.Name == auth.CSRFCookieName {
			form.Set(auth.CSRFFieldName, c.Value)
		}
	}

	req := httptest.NewRequest(http.MethodPost, "/admin/login", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	for _, c := range csrfCookie {
		req.AddCookie(c)
	}

	rec := httptest.NewRecorder()
	adm.HandleLogin(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Errorf("want redirect 303, got %d", rec.Code)
	}
	if loc := rec.Header().Get("Location"); loc != "/admin" {
		t.Errorf("want redirect to /admin, got %q", loc)
	}
	if len(ss.sessions) == 0 {
		t.Error("expected session to be created")
	}
}

func TestHandleLogin_BadCredentials(t *testing.T) {
	adm, _, _ := newAdmin(t, &stubJellyfinClient{err: domain.ErrInviteNotFound}) // any error

	w0 := httptest.NewRecorder()
	adm.HandleLoginForm(w0, httptest.NewRequest(http.MethodGet, "/admin/login", nil))
	csrfCookie := w0.Result().Cookies()

	form := url.Values{"username": {"admin"}, "password": {"wrong"}}
	for _, c := range csrfCookie {
		if c.Name == auth.CSRFCookieName {
			form.Set(auth.CSRFFieldName, c.Value)
		}
	}

	req := httptest.NewRequest(http.MethodPost, "/admin/login", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	for _, c := range csrfCookie {
		req.AddCookie(c)
	}

	rec := httptest.NewRecorder()
	adm.HandleLogin(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Errorf("want redirect on bad credentials, got %d", rec.Code)
	}
	if !strings.Contains(rec.Header().Get("Location"), "error=") {
		t.Error("expected error param in redirect")
	}
}

func TestHandleLogin_CSRFRejected(t *testing.T) {
	adm, _, _ := newAdmin(t, &stubJellyfinClient{token: "tok"})

	form := url.Values{"username": {"admin"}, "password": {"pass"}, auth.CSRFFieldName: {"wrong"}}
	req := httptest.NewRequest(http.MethodPost, "/admin/login", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: auth.CSRFCookieName, Value: "real-token"})

	rec := httptest.NewRecorder()
	adm.HandleLogin(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("want 403 on CSRF failure, got %d", rec.Code)
	}
}

func TestHandleDashboard_ListsInvites(t *testing.T) {
	adm, ss, is := newAdmin(t, &stubJellyfinClient{token: "tok"})
	is.invites["inv-1"] = domain.Invite{ID: "inv-1", Token: "tok1", Label: "test invite", CreatedBy: "admin"}

	sess := domain.Session{Token: "s", Username: "admin", JellyfinToken: "jf", ExpiresAt: time.Now().Add(time.Hour)}
	ss.sessions["s"] = sess

	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	req.AddCookie(&http.Cookie{Name: auth.SessionCookieName, Value: "s"})
	req = withSession(req, sess)

	rec := httptest.NewRecorder()

	// Wire RequireSession middleware.
	mgr := auth.NewManager(ss, false)
	middleware.RequireSession(mgr)(http.HandlerFunc(adm.HandleDashboard)).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("want 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "test invite") {
		t.Error("expected invite label in dashboard HTML")
	}
}

func TestHandleCreateInvite(t *testing.T) {
	adm, ss, is := newAdmin(t, &stubJellyfinClient{token: "tok"})

	sess := domain.Session{Token: "s", Username: "admin", JellyfinToken: "jf", ExpiresAt: time.Now().Add(time.Hour)}
	ss.sessions["s"] = sess

	// Get a valid CSRF token.
	w0 := httptest.NewRecorder()
	req0 := httptest.NewRequest(http.MethodGet, "/admin", nil)
	req0 = withSession(req0, sess)
	adm.HandleDashboard(rec(w0), req0)
	csrfCookie := w0.Result().Cookies()

	form := url.Values{"label": {"friends"}}
	for _, c := range csrfCookie {
		if c.Name == auth.CSRFCookieName {
			form.Set(auth.CSRFFieldName, c.Value)
		}
	}

	req := httptest.NewRequest(http.MethodPost, "/admin/invites", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	for _, c := range csrfCookie {
		req.AddCookie(c)
	}
	req = withSession(req, sess)

	rec2 := httptest.NewRecorder()
	adm.HandleCreateInvite(rec2, req)

	if rec2.Code != http.StatusSeeOther {
		t.Errorf("want redirect 303, got %d", rec2.Code)
	}
	if len(is.invites) == 0 {
		t.Error("expected invite to be created")
	}
}

func TestHandleRevokeInvite(t *testing.T) {
	adm, ss, is := newAdmin(t, &stubJellyfinClient{token: "tok"})

	id := uuid.NewString()
	is.invites[id] = domain.Invite{ID: id, Token: "tok-r", Label: "r", CreatedBy: "admin"}
	sess := domain.Session{Token: "s", Username: "admin", JellyfinToken: "jf", ExpiresAt: time.Now().Add(time.Hour)}
	ss.sessions["s"] = sess

	// Get CSRF token.
	w0 := httptest.NewRecorder()
	req0 := httptest.NewRequest(http.MethodGet, "/admin", nil)
	req0 = withSession(req0, sess)
	adm.HandleDashboard(rec(w0), req0)

	form := url.Values{}
	for _, c := range w0.Result().Cookies() {
		if c.Name == auth.CSRFCookieName {
			form.Set(auth.CSRFFieldName, c.Value)
		}
	}

	req := httptest.NewRequest(http.MethodPost, "/admin/invites/"+id+"/revoke", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetPathValue("id", id)
	for _, c := range w0.Result().Cookies() {
		req.AddCookie(c)
	}
	req = withSession(req, sess)

	rec2 := httptest.NewRecorder()
	adm.HandleRevokeInvite(rec2, req)

	if rec2.Code != http.StatusSeeOther {
		t.Errorf("want redirect 303, got %d", rec2.Code)
	}
	if !is.invites[id].Revoked {
		t.Error("expected invite to be revoked")
	}
}

// rec is a helper alias to keep test lines short.
func rec(w *httptest.ResponseRecorder) http.ResponseWriter { return w }
