package handler

import (
	"fmt"
	"html/template"
	"net/http"

	"github.com/rmewborne/jellygate/internal/domain"
	"github.com/rmewborne/jellygate/web"
)

// SetupHandler serves the first-run setup wizard at /setup.
type SetupHandler struct {
	settings domain.SettingsStore
	tmpl     *template.Template
}

// NewSetupHandler constructs a SetupHandler.
func NewSetupHandler(settings domain.SettingsStore) (*SetupHandler, error) {
	tmpl, err := template.ParseFS(web.FS, "templates/base.html", "templates/setup.html")
	if err != nil {
		return nil, fmt.Errorf("handler.NewSetupHandler: parse template: %w", err)
	}
	return &SetupHandler{settings: settings, tmpl: tmpl}, nil
}

type setupData struct {
	Error       string
	JellyfinURL string
	SeerrURL    string
	DiscordURL  string
	// base template fields
	Username  string
	Flash     string
	FlashType string
}

// HandleSetupForm renders GET /setup.
func (h *SetupHandler) HandleSetupForm(w http.ResponseWriter, r *http.Request) {
	h.render(w, setupData{})
}

// HandleSetupSubmit processes POST /setup.
func (h *SetupHandler) HandleSetupSubmit(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	jellyfinURL := r.FormValue("jellyfin_url")
	seerrURL := r.FormValue("seerr_url")
	discordURL := r.FormValue("discord_url")

	if err := validateAbsoluteURL(jellyfinURL); err != nil {
		h.render(w, setupData{
			Error:       "Jellyfin URL must be a valid http/https URL (e.g. http://localhost:8096)",
			JellyfinURL: jellyfinURL,
			SeerrURL:    seerrURL,
			DiscordURL:  discordURL,
		})
		return
	}
	if seerrURL != "" {
		if err := validateAbsoluteURL(seerrURL); err != nil {
			h.render(w, setupData{
				Error:       "Seerr URL must be a valid http/https URL",
				JellyfinURL: jellyfinURL,
				SeerrURL:    seerrURL,
				DiscordURL:  discordURL,
			})
			return
		}
	}
	if discordURL != "" {
		if err := validateAbsoluteURL(discordURL); err != nil {
			h.render(w, setupData{
				Error:       "Discord webhook URL must be a valid http/https URL",
				JellyfinURL: jellyfinURL,
				SeerrURL:    seerrURL,
				DiscordURL:  discordURL,
			})
			return
		}
	}

	ctx := r.Context()
	if err := h.settings.SetJellyfinURL(ctx, jellyfinURL); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if seerrURL != "" {
		if err := h.settings.SetSeerrURL(ctx, seerrURL); err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
	}
	if discordURL != "" {
		if err := h.settings.SetDiscordWebhookURL(ctx, discordURL); err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
	}

	http.Redirect(w, r, "/admin/login", http.StatusSeeOther)
}

func (h *SetupHandler) render(w http.ResponseWriter, data setupData) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.tmpl.ExecuteTemplate(w, "base", data); err != nil {
		http.Error(w, "render error", http.StatusInternalServerError)
	}
}
