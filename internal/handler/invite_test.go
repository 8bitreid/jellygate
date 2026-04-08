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
	cfg := &stubSettingsStore{token: "admin-tok"}
	h, err := handler.NewInviteHandler(is, rs, jf, &notifications.NoopNotifier{}, cfg)
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
	if !strings.Contains(rec.Body.String(), "create my account") {
		t.Error("expected registration form")
	}
}

func TestHandleInviteForm_InvalidStates(t *testing.T) {
	cases := []struct {
		name         string
		setupInvite  func(is *stubInviteStore, id, token string)
		token        string
		wantContains string
	}{
		{
			name:         "invalid token",
			setupInvite:  func(is *stubInviteStore, id, token string) {}, // no invite created
			token:        "no-such-token",
			wantContains: "invite unavailable",
		},
		{
			name: "revoked",
			setupInvite: func(is *stubInviteStore, id, token string) {
				inv := activeInvite(id, token)
				inv.Revoked = true
				is.invites[id] = inv
			},
			token:        "tok-rev",
			wantContains: "revoked",
		},
		{
			name: "expired",
			setupInvite: func(is *stubInviteStore, id, token string) {
				inv := activeInvite(id, token)
				past := time.Now().Add(-time.Hour)
				inv.ExpiresAt = &past
				is.invites[id] = inv
			},
			token:        "tok-exp",
			wantContains: "expired",
		},
		{
			name: "exhausted",
			setupInvite: func(is *stubInviteStore, id, token string) {
				inv := activeInvite(id, token)
				max := 1
				inv.MaxUses = &max
				inv.UseCount = 1
				is.invites[id] = inv
			},
			token:        "tok-ex",
			wantContains: "maximum",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			is := newInviteStore()
			id := uuid.NewString()
			tc.setupInvite(is, id, tc.token)

			h, _ := newInviteHandler(t, is, &stubJellyfinClient{token: "tok"})
			req := httptest.NewRequest(http.MethodGet, "/invite/"+tc.token, nil)
			req.SetPathValue("token", tc.token)
			rec := httptest.NewRecorder()

			h.HandleInviteForm(rec, req)

			if rec.Code != http.StatusOK {
				t.Errorf("want 200, got %d", rec.Code)
			}
			if !strings.Contains(rec.Body.String(), tc.wantContains) {
				t.Errorf("expected response to contain %q", tc.wantContains)
			}
		})
	}
}

func TestHandleInviteSubmit_Success(t *testing.T) {
	is := newInviteStore()
	id := uuid.NewString()
	is.invites[id] = activeInvite(id, "tok-ok")

	h, rs := newInviteHandler(t, is, &stubJellyfinClient{token: "tok"})

	form := url.Values{"username": {"newuser"}, "password": {"Secret1!"}, "confirm": {"Secret1!"}}
	req := httptest.NewRequest(http.MethodPost, "/invite/tok-ok", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetPathValue("token", "tok-ok")
	rec := httptest.NewRecorder()

	h.HandleInviteSubmit(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Errorf("want 303, got %d", rec.Code)
	}
	if loc := rec.Header().Get("Location"); loc != "/tutorial" {
		t.Errorf("expected redirect to /tutorial, got %q", loc)
	}
	if len(rs.regs) == 0 {
		t.Error("expected registration to be recorded")
	}
	if is.invites[id].UseCount != 1 {
		t.Error("expected use count to be incremented")
	}
}

func TestHandleInviteSubmit_ValidationErrors(t *testing.T) {
	cases := []struct {
		name              string
		setupInvite       func(is *stubInviteStore, id, token string)
		username          string
		password          string
		confirm           string
		wantContains      string
		skipUseCountCheck bool
	}{
		{
			name:         "password mismatch",
			setupInvite:  func(is *stubInviteStore, id, token string) { is.invites[id] = activeInvite(id, token) },
			username:     "newuser",
			password:     "Secret1!",
			confirm:      "different",
			wantContains: "do not match",
		},
		{
			name: "already exhausted",
			setupInvite: func(is *stubInviteStore, id, token string) {
				inv := activeInvite(id, token)
				max := 1
				inv.MaxUses = &max
				inv.UseCount = 1
				is.invites[id] = inv
			},
			username:          "user",
			password:          "Secret1!",
			confirm:           "Secret1!",
			wantContains:      "maximum",
			skipUseCountCheck: true, // invite was already exhausted
		},
		{
			name:         "weak password",
			setupInvite:  func(is *stubInviteStore, id, token string) { is.invites[id] = activeInvite(id, token) },
			username:     "user",
			password:     "weak",
			confirm:      "weak",
			wantContains: "password must be at least 8 characters",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			is := newInviteStore()
			id := uuid.NewString()
			token := "tok-val"
			tc.setupInvite(is, id, token)

			h, _ := newInviteHandler(t, is, &stubJellyfinClient{token: "tok"})

			form := url.Values{"username": {tc.username}, "password": {tc.password}, "confirm": {tc.confirm}}
			req := httptest.NewRequest(http.MethodPost, "/invite/"+token, strings.NewReader(form.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			req.SetPathValue("token", token)
			rec := httptest.NewRecorder()

			h.HandleInviteSubmit(rec, req)

			if !strings.Contains(rec.Body.String(), tc.wantContains) {
				t.Errorf("expected response to contain %q, got: %s", tc.wantContains, rec.Body.String())
			}
			if !tc.skipUseCountCheck && is.invites[id].UseCount != 0 {
				t.Error("use count should not increment on validation error")
			}
		})
	}
}

func TestHandleInviteSubmit_JellyfinError(t *testing.T) {
	is := newInviteStore()
	id := uuid.NewString()
	is.invites[id] = activeInvite(id, "tok-jf")

	h, _ := newInviteHandler(t, is, &stubJellyfinClientErr{err: errors.New("username taken")})

	form := url.Values{"username": {"taken"}, "password": {"Secret1!"}, "confirm": {"Secret1!"}}
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
