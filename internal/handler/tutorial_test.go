package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

const (
	testMediaURL = "http://jellyfin.example.com"
	testSeerrURL = "http://seerr.example.com"
)

func newTutorialHandler(t *testing.T) *TutorialHandler {
	t.Helper()
	h, err := NewTutorialHandler(testMediaURL, testSeerrURL)
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
	for _, want := range []string{"suggested apps", testMediaURL, testSeerrURL} {
		if !findSubstring(body, want) {
			t.Errorf("body missing %q", want)
		}
	}
}

func TestTutorialHandler_HandleTutorial_NoSeerr(t *testing.T) {
	h, err := NewTutorialHandler(testMediaURL, "")
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
	if findSubstring(body, "request & report") {
		t.Error("Seerr step should be hidden when SeerrURL is empty")
	}
}

func TestTutorialHandler_HandleTutorialSkip(t *testing.T) {
	h := newTutorialHandler(t)

	req := httptest.NewRequest(http.MethodPost, "/tutorial/skip", nil)
	w := httptest.NewRecorder()
	h.HandleTutorialSkip(w, req)

	if w.Code != http.StatusFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusFound)
	}
	if loc := w.Header().Get("Location"); loc != testMediaURL {
		t.Errorf("Location = %q, want %q", loc, testMediaURL)
	}
}

func TestTutorialHandler_HandleTutorialComplete(t *testing.T) {
	h := newTutorialHandler(t)

	req := httptest.NewRequest(http.MethodPost, "/tutorial/complete", nil)
	w := httptest.NewRecorder()
	h.HandleTutorialComplete(w, req)

	if w.Code != http.StatusFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusFound)
	}
	if loc := w.Header().Get("Location"); loc != testMediaURL {
		t.Errorf("Location = %q, want %q", loc, testMediaURL)
	}
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
