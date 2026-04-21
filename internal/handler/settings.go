package handler

import (
	"fmt"
	"html/template"
	"net/http"

	"github.com/rmewborne/jellygate/internal/auth"
	"github.com/rmewborne/jellygate/internal/domain"
	"github.com/rmewborne/jellygate/web"
)

// SettingsHandler serves GET/POST /admin/settings.
type SettingsHandler struct {
	settings domain.SettingsStore
	secure   bool
	tmpl     *template.Template
}

// NewSettingsHandler constructs a SettingsHandler.
func NewSettingsHandler(settings domain.SettingsStore, secure bool) (*SettingsHandler, error) {
	tmpl, err := template.ParseFS(web.FS, "templates/base.html", "templates/settings.html")
	if err != nil {
		return nil, fmt.Errorf("handler.NewSettingsHandler: parse template: %w", err)
	}
	return &SettingsHandler{settings: settings, secure: secure, tmpl: tmpl}, nil
}

type settingsData struct {
	CSRFToken  string
	SeerrURL   string
	DiscordURL string
	Flash      string
	FlashType  string
	Username   string
}

// HandleSettingsForm renders GET /admin/settings.
func (h *SettingsHandler) HandleSettingsForm(w http.ResponseWriter, r *http.Request) {
	sess, _ := auth.FromContext(r.Context())

	csrfToken, err := auth.SetCSRFCookie(w, h.secure)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	ctx := r.Context()
	seerrURL, _ := h.settings.GetSeerrURL(ctx)
	discordURL, _ := h.settings.GetDiscordWebhookURL(ctx)

	data := settingsData{
		CSRFToken:  csrfToken,
		SeerrURL:   seerrURL,
		DiscordURL: discordURL,
		Username:   sess.Username,
	}
	if msg := r.URL.Query().Get("flash"); msg != "" {
		data.Flash = msg
		data.FlashType = r.URL.Query().Get("type")
		if data.FlashType == "" {
			data.FlashType = "success"
		}
	}
	h.render(w, data)
}

// HandleSettingsSubmit processes POST /admin/settings.
func (h *SettingsHandler) HandleSettingsSubmit(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if !auth.ValidateCSRF(r) {
		http.Error(w, "invalid CSRF token", http.StatusForbidden)
		return
	}

	seerrURL := r.FormValue("seerr_url")
	discordURL := r.FormValue("discord_url")

	if seerrURL != "" {
		if err := validateAbsoluteURL(seerrURL); err != nil {
			sess, _ := auth.FromContext(r.Context())
			csrfToken, _ := auth.SetCSRFCookie(w, h.secure)
			h.render(w, settingsData{
				CSRFToken:  csrfToken,
				SeerrURL:   seerrURL,
				DiscordURL: discordURL,
				Username:   sess.Username,
				Flash:      "Seerr URL must be a valid http/https URL",
				FlashType:  "error",
			})
			return
		}
	}
	if discordURL != "" {
		if err := validateAbsoluteURL(discordURL); err != nil {
			sess, _ := auth.FromContext(r.Context())
			csrfToken, _ := auth.SetCSRFCookie(w, h.secure)
			h.render(w, settingsData{
				CSRFToken:  csrfToken,
				SeerrURL:   seerrURL,
				DiscordURL: discordURL,
				Username:   sess.Username,
				Flash:      "Discord webhook URL must be a valid http/https URL",
				FlashType:  "error",
			})
			return
		}
	}

	ctx := r.Context()
	if err := h.settings.SetSeerrURL(ctx, seerrURL); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if err := h.settings.SetDiscordWebhookURL(ctx, discordURL); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/admin/settings?flash=settings+saved", http.StatusSeeOther)
}

func (h *SettingsHandler) render(w http.ResponseWriter, data settingsData) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.tmpl.ExecuteTemplate(w, "base", data); err != nil {
		http.Error(w, "render error", http.StatusInternalServerError)
	}
}
