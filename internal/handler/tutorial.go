package handler

import (
	"fmt"
	"html/template"
	"net/http"
	"net/url"

	"github.com/rmewborne/jellygate/internal/domain"
	"github.com/rmewborne/jellygate/web"
)

// TutorialHandler serves the onboarding tutorial for new users.
type TutorialHandler struct {
	settings domain.SettingsStore
	tmpl     *template.Template
}

// NewTutorialHandler constructs a TutorialHandler.
func NewTutorialHandler(settings domain.SettingsStore) (*TutorialHandler, error) {
	tmpl, err := template.ParseFS(web.FS, "templates/base.html", "templates/tutorial.html")
	if err != nil {
		return nil, fmt.Errorf("handler.NewTutorialHandler: parse template: %w", err)
	}
	return &TutorialHandler{settings: settings, tmpl: tmpl}, nil
}

// validateAbsoluteURL returns an error if rawURL is not an absolute http/https URL with a non-empty host.
func validateAbsoluteURL(rawURL string) error {
	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("not a valid URL: %w", err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("URL scheme must be http or https, got %q", u.Scheme)
	}
	if u.Host == "" {
		return fmt.Errorf("URL must have a non-empty host")
	}
	return nil
}

type tutorialPageData struct {
	MediaURL string
	SeerrURL string
	Flash    string
	FlashType string
}

// HandleTutorial renders GET /tutorial.
func (h *TutorialHandler) HandleTutorial(w http.ResponseWriter, r *http.Request) {
	mediaURL, seerrURL := h.urls(r)
	h.render(w, tutorialPageData{MediaURL: mediaURL, SeerrURL: seerrURL})
}

// HandleTutorialSkip handles GET /tutorial/skip.
func (h *TutorialHandler) HandleTutorialSkip(w http.ResponseWriter, r *http.Request) {
	mediaURL, _ := h.urls(r)
	http.Redirect(w, r, mediaURL, http.StatusSeeOther)
}

// HandleTutorialComplete handles GET /tutorial/complete.
func (h *TutorialHandler) HandleTutorialComplete(w http.ResponseWriter, r *http.Request) {
	mediaURL, _ := h.urls(r)
	http.Redirect(w, r, mediaURL, http.StatusSeeOther)
}

func (h *TutorialHandler) urls(r *http.Request) (mediaURL, seerrURL string) {
	mediaURL, _ = h.settings.GetJellyfinURL(r.Context())
	seerrURL, _ = h.settings.GetSeerrURL(r.Context())
	return
}

// --- helpers ---

func (h *TutorialHandler) render(w http.ResponseWriter, data tutorialPageData) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.tmpl.ExecuteTemplate(w, "base", data); err != nil {
		http.Error(w, "render error", http.StatusInternalServerError)
	}
}
