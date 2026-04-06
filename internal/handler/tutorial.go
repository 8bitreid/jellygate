package handler

import (
	"fmt"
	"html/template"
	"net/http"
	"net/url"

	"github.com/rmewborne/jellygate/web"
)

// TutorialHandler serves the onboarding tutorial for new users.
type TutorialHandler struct {
	mediaURL string
	seerrURL string // optional — Seerr step is hidden when empty
	tmpl     *template.Template
}

// NewTutorialHandler constructs a TutorialHandler.
// mediaURL is the public-facing Jellyfin URL users will be redirected to.
// seerrURL is optional; the request/report step is omitted when blank.
func NewTutorialHandler(mediaURL, seerrURL string) (*TutorialHandler, error) {
	if err := validateAbsoluteURL(mediaURL); err != nil {
		return nil, fmt.Errorf("handler.NewTutorialHandler: invalid MEDIA_URL: %w", err)
	}
	if seerrURL != "" {
		if err := validateAbsoluteURL(seerrURL); err != nil {
			return nil, fmt.Errorf("handler.NewTutorialHandler: invalid SEERR_URL: %w", err)
		}
	}
	tmpl, err := template.ParseFS(web.FS, "templates/base.html", "templates/tutorial.html")
	if err != nil {
		return nil, fmt.Errorf("handler.NewTutorialHandler: parse template: %w", err)
	}
	return &TutorialHandler{mediaURL: mediaURL, seerrURL: seerrURL, tmpl: tmpl}, nil
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
	h.render(w, tutorialPageData{MediaURL: h.mediaURL, SeerrURL: h.seerrURL})
}

// HandleTutorialSkip handles GET /tutorial/skip.
// Redirects the user directly to the media server.
func (h *TutorialHandler) HandleTutorialSkip(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, h.mediaURL, http.StatusSeeOther)
}

// HandleTutorialComplete handles GET /tutorial/complete.
// Redirects the user to the media server after completing the tutorial.
func (h *TutorialHandler) HandleTutorialComplete(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, h.mediaURL, http.StatusSeeOther)
}

// --- helpers ---

func (h *TutorialHandler) render(w http.ResponseWriter, data tutorialPageData) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.tmpl.ExecuteTemplate(w, "base", data); err != nil {
		http.Error(w, "render error", http.StatusInternalServerError)
	}
}
