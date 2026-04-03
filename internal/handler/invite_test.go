package handler_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/rmewborne/jellygate/internal/domain"
	"github.com/rmewborne/jellygate/internal/handler"
	"github.com/rmewborne/jellygate/internal/notifications"
)

// --- stubs ---

type stubRegistrationStore struct{ regs []domain.Registration }

func (s *stubRegistrationStore) Create(_ context.Context, reg domain.Registration) error {
	s.regs = append(s.regs, reg)
	return nil
}

type stubJellyfinClientErr struct{ err error }

func (j *stubJellyfinClientErr) Authenticate(_ context.Context, _, _ string) (string, error) {
	return "", j.err
}
func (j *stubJellyfinClientErr) ListLibraries(_ context.Context, _ string) ([]domain.Library, error) {
	return nil, nil
}
func (j *stubJellyfinClientErr) CreateUser(_ context.Context, _, _, _ string) (string, error) {
	return "", j.err
}
func (j *stubJellyfinClientErr) SetLibraryAccess(_ context.Context, _, _ string, _ []string) error {
	return nil
}

// --- helpers ---

func newInviteHandler(t *testing.T, is domain.InviteStore, jf domain.JellyfinClient) (*handler.InviteHandler, *stubRegistrationStore) {
	t.Helper()
	rs := &stubRegistrationStore{}
	h, err := handler.NewInviteHandler(is, rs, jf, &notifications.NoopNotifier{}, "admin-tok")
	if err != nil {
		t.Fatalf("NewInviteHandler: %v", err)
	}
	return h, rs
}

func activeInvite(id, token string) domain.Invite {
	return domain.Invite{
		ID:        id,
		Token:     token,
		Label:     "test invite",
		CreatedBy: "admin",
		ExpiresAt: nil,
		MaxUses:   nil,
	}
}

// --- tests ---

func TestHandleInviteForm_Valid(t *testing.T) {
	is := newInviteStore()
	id := uuid.NewString()
	is.invites[id] = activeInvite(id, "good-token")

	h, _ := newInviteHandler(t, is, &stubJellyfinClient{token: "tok"})
	req := httptest.NewRequest(http.MethodGet, "/invite/good-token", nil)
	req.SetPathValue("token", "good-token")
	rec := httptest.NewRecorder()

	h.HandleInviteForm(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("want 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "create account") {
		t.Error("expected registration form")
	}
}

func TestHandleInviteForm_InvalidToken(t *testing.T) {
	h, _ := newInviteHandler(t, newInviteStore(), &stubJellyfinClient{token: "tok"})
	req := httptest.NewRequest(http.MethodGet, "/invite/no-such-token", nil)
	req.SetPathValue("token", "no-such-token")
	rec := httptest.NewRecorder()

	h.HandleInviteForm(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("want 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "invite unavailable") {
		t.Error("expected error page")
	}
}

func TestHandleInviteForm_Revoked(t *testing.T) {
	is := newInviteStore()
	id := uuid.NewString()
	inv := activeInvite(id, "tok-rev")
	inv.Revoked = true
	is.invites[id] = inv

	h, _ := newInviteHandler(t, is, &stubJellyfinClient{token: "tok"})
	req := httptest.NewRequest(http.MethodGet, "/invite/tok-rev", nil)
	req.SetPathValue("token", "tok-rev")
	rec := httptest.NewRecorder()

	h.HandleInviteForm(rec, req)

	if !strings.Contains(rec.Body.String(), "revoked") {
		t.Error("expected revoked message")
	}
}

func TestHandleInviteForm_Expired(t *testing.T) {
	is := newInviteStore()
	id := uuid.NewString()
	inv := activeInvite(id, "tok-exp")
	past := time.Now().Add(-time.Hour)
	inv.ExpiresAt = &past
	is.invites[id] = inv

	h, _ := newInviteHandler(t, is, &stubJellyfinClient{token: "tok"})
	req := httptest.NewRequest(http.MethodGet, "/invite/tok-exp", nil)
	req.SetPathValue("token", "tok-exp")
	rec := httptest.NewRecorder()

	h.HandleInviteForm(rec, req)

	if !strings.Contains(rec.Body.String(), "expired") {
		t.Error("expected expired message")
	}
}

func TestHandleInviteForm_Exhausted(t *testing.T) {
	is := newInviteStore()
	id := uuid.NewString()
	inv := activeInvite(id, "tok-ex")
	max := 1
	inv.MaxUses = &max
	inv.UseCount = 1
	is.invites[id] = inv

	h, _ := newInviteHandler(t, is, &stubJellyfinClient{token: "tok"})
	req := httptest.NewRequest(http.MethodGet, "/invite/tok-ex", nil)
	req.SetPathValue("token", "tok-ex")
	rec := httptest.NewRecorder()

	h.HandleInviteForm(rec, req)

	if !strings.Contains(rec.Body.String(), "maximum") {
		t.Error("expected exhausted message")
	}
}

func TestHandleInviteSubmit_Success(t *testing.T) {
	is := newInviteStore()
	id := uuid.NewString()
	is.invites[id] = activeInvite(id, "tok-ok")

	h, rs := newInviteHandler(t, is, &stubJellyfinClient{token: "tok"})

	form := url.Values{"username": {"newuser"}, "password": {"secret1"}, "confirm": {"secret1"}}
	req := httptest.NewRequest(http.MethodPost, "/invite/tok-ok", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetPathValue("token", "tok-ok")
	rec := httptest.NewRecorder()

	h.HandleInviteSubmit(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("want 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "welcome") {
		t.Error("expected success page")
	}
	if len(rs.regs) == 0 {
		t.Error("expected registration to be recorded")
	}
	if is.invites[id].UseCount != 1 {
		t.Error("expected use count to be incremented")
	}
}

func TestHandleInviteSubmit_PasswordMismatch(t *testing.T) {
	is := newInviteStore()
	id := uuid.NewString()
	is.invites[id] = activeInvite(id, "tok-mm")

	h, _ := newInviteHandler(t, is, &stubJellyfinClient{token: "tok"})

	form := url.Values{"username": {"newuser"}, "password": {"secret1"}, "confirm": {"different"}}
	req := httptest.NewRequest(http.MethodPost, "/invite/tok-mm", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetPathValue("token", "tok-mm")
	rec := httptest.NewRecorder()

	h.HandleInviteSubmit(rec, req)

	if !strings.Contains(rec.Body.String(), "do not match") {
		t.Error("expected password mismatch error")
	}
	if is.invites[id].UseCount != 0 {
		t.Error("use count should not increment on validation failure")
	}
}

func TestHandleInviteSubmit_JellyfinError(t *testing.T) {
	is := newInviteStore()
	id := uuid.NewString()
	is.invites[id] = activeInvite(id, "tok-jf")

	h, _ := newInviteHandler(t, is, &stubJellyfinClientErr{err: errors.New("username taken")})

	form := url.Values{"username": {"taken"}, "password": {"secret1"}, "confirm": {"secret1"}}
	req := httptest.NewRequest(http.MethodPost, "/invite/tok-jf", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetPathValue("token", "tok-jf")
	rec := httptest.NewRecorder()

	h.HandleInviteSubmit(rec, req)

	if !strings.Contains(rec.Body.String(), "could not create account") {
		t.Error("expected jellyfin error message")
	}
	if is.invites[id].UseCount != 0 {
		t.Error("use count should not increment on jellyfin failure")
	}
}

func TestHandleInviteSubmit_AlreadyExhausted(t *testing.T) {
	is := newInviteStore()
	id := uuid.NewString()
	inv := activeInvite(id, "tok-exh")
	max := 1
	inv.MaxUses = &max
	inv.UseCount = 1
	is.invites[id] = inv

	h, _ := newInviteHandler(t, is, &stubJellyfinClient{token: "tok"})

	form := url.Values{"username": {"user"}, "password": {"secret1"}, "confirm": {"secret1"}}
	req := httptest.NewRequest(http.MethodPost, "/invite/tok-exh", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetPathValue("token", "tok-exh")
	rec := httptest.NewRecorder()

	h.HandleInviteSubmit(rec, req)

	if !strings.Contains(rec.Body.String(), "maximum") {
		t.Error("expected exhausted error page")
	}
}
