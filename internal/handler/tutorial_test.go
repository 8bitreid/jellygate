package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/rmewborne/jellygate/internal/domain"
)

const (
	testMediaURL = "http://jellyfin.example.com"
	testSeerrURL = "http://seerr.example.com"
)

// tutorialStubSettings returns fixed URLs for tutorial tests.
type tutorialStubSettings struct {
	jellyfinURL string
	seerrURL    string
}

func (s *tutorialStubSettings) GetJellyfinURL(_ context.Context) (string, error) {
	return s.jellyfinURL, nil
}
func (s *tutorialStubSettings) SetJellyfinURL(_ context.Context, _ string) error { return nil }
func (s *tutorialStubSettings) GetSeerrURL(_ context.Context) (string, error) {
	if s.seerrURL == "" {
		return "", domain.ErrSettingNotFound
	}
	return s.seerrURL, nil
}
func (s *tutorialStubSettings) SetSeerrURL(_ context.Context, _ string) error { return nil }
func (s *tutorialStubSettings) GetJellyfinAdminToken(_ context.Context) (string, error) {
	return "", domain.ErrSettingNotFound
}
func (s *tutorialStubSettings) SetJellyfinAdminToken(_ context.Context, _ string) error { return nil }
func (s *tutorialStubSettings) GetDiscordWebhookURL(_ context.Context) (string, error) {
	return "", domain.ErrSettingNotFound
}
func (s *tutorialStubSettings) SetDiscordWebhookURL(_ context.Context, _ string) error { return nil }

func newTutorialHandler(t *testing.T) *TutorialHandler {
	t.Helper()
	h, err := NewTutorialHandler(&tutorialStubSettings{jellyfinURL: testMediaURL, seerrURL: testSeerrURL})
	if err != nil {
		t.Fatalf("NewTutorialHandler() error = %v", err)
	}
	return h
}

func TestTutorialHandler_HandleTutorial(t *testing.T) {
	h := newTutorialHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/tutorial", nil)
	w := httptest.NewRecorder()

	h.HandleTutorial(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if ct := w.Header().Get("Content-Type"); ct != "text/html; charset=utf-8" {
		t.Errorf("Content-Type = %q, want text/html; charset=utf-8", ct)
	}

	body := w.Body.String()
	for _, want := range []string{"watch on your devices", testMediaURL, testSeerrURL} {
		if !strings.Contains(body, want) {
			t.Errorf("body missing %q", want)
		}
	}
}

func TestTutorialHandler_HandleTutorial_NoSeerr(t *testing.T) {
	h, err := NewTutorialHandler(&tutorialStubSettings{jellyfinURL: testMediaURL})
	if err != nil {
		t.Fatalf("NewTutorialHandler() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/tutorial", nil)
	w := httptest.NewRecorder()
	h.HandleTutorial(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
	body := w.Body.String()
	if strings.Contains(body, "request & report") {
		t.Error("Seerr step should be hidden when SeerrURL is empty")
	}
}

func TestTutorialHandler_HandleTutorialSkip(t *testing.T) {
	h := newTutorialHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/tutorial/skip", nil)
	w := httptest.NewRecorder()
	h.HandleTutorialSkip(w, req)

	if w.Code != http.StatusSeeOther {
		t.Errorf("status = %d, want %d", w.Code, http.StatusSeeOther)
	}
	if loc := w.Header().Get("Location"); loc != testMediaURL {
		t.Errorf("Location = %q, want %q", loc, testMediaURL)
	}
}

func TestTutorialHandler_HandleTutorialComplete(t *testing.T) {
	h := newTutorialHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/tutorial/complete", nil)
	w := httptest.NewRecorder()
	h.HandleTutorialComplete(w, req)

	if w.Code != http.StatusSeeOther {
		t.Errorf("status = %d, want %d", w.Code, http.StatusSeeOther)
	}
	if loc := w.Header().Get("Location"); loc != testMediaURL {
		t.Errorf("Location = %q, want %q", loc, testMediaURL)
	}
}
