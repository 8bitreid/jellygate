package handler

import (
	"errors"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/rmewborne/jellygate/internal/domain"
	"github.com/rmewborne/jellygate/web"
)

// InviteHandler serves the public invite registration flow.
type InviteHandler struct {
	invites       domain.InviteStore
	registrations domain.RegistrationStore
	jellyfin      domain.JellyfinClient
	notifier      domain.Notifier
	settings      domain.SettingsStore
	tmpl          *template.Template
}

// NewInviteHandler constructs an InviteHandler.
// The Jellyfin admin token is fetched from settings at request time, so the
// handler works correctly even if the token changes after a re-login.
func NewInviteHandler(
	invites domain.InviteStore,
	registrations domain.RegistrationStore,
	jellyfin domain.JellyfinClient,
	notifier domain.Notifier,
	settings domain.SettingsStore,
) (*InviteHandler, error) {
	tmpl, err := template.ParseFS(web.FS, "templates/base.html", "templates/invite.html")
	if err != nil {
		return nil, fmt.Errorf("handler.NewInviteHandler: parse template: %w", err)
	}
	return &InviteHandler{
		invites:       invites,
		registrations: registrations,
		jellyfin:      jellyfin,
		notifier:      notifier,
		settings:      settings,
		tmpl:          tmpl,
	}, nil
}

type invitePageData struct {
	Status    string // "form" | "success" | "error"
	Token     string
	Label     string // invite label shown to the registrant
	Error     string // inline form validation error
	Message   string // error page explanation
	Flash     string // required by base.html layout
	FlashType string // required by base.html layout
}

// HandleInviteForm renders GET /invite/{token}.
func (h *InviteHandler) HandleInviteForm(w http.ResponseWriter, r *http.Request) {
	token := r.PathValue("token")
	inv, err := h.invites.GetByToken(r.Context(), token)
	if err != nil {
		h.render(w, invitePageData{Status: "error", Message: "this invite link is not valid"})
		return
	}
	if err := inv.IsValid(); err != nil {
		h.render(w, invitePageData{Status: "error", Message: inviteErrMessage(err)})
		return
	}
	h.render(w, invitePageData{Status: "form", Token: token, Label: inv.Label})
}

// HandleInviteSubmit processes POST /invite/{token}.
func (h *InviteHandler) HandleInviteSubmit(w http.ResponseWriter, r *http.Request) {
	token := r.PathValue("token")

	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	inv, err := h.invites.GetByToken(r.Context(), token)
	if err != nil {
		h.render(w, invitePageData{Status: "error", Message: "this invite link is not valid"})
		return
	}
	if err := inv.IsValid(); err != nil {
		h.render(w, invitePageData{Status: "error", Message: inviteErrMessage(err)})
		return
	}

	username := r.FormValue("username")
	password := r.FormValue("password")
	confirm := r.FormValue("confirm")

	if username == "" || password == "" {
		h.render(w, invitePageData{
			Status: "form", Token: token, Label: inv.Label,
			Error: "username and password are required",
		})
		return
	}
	if password != confirm {
		h.render(w, invitePageData{
			Status: "form", Token: token, Label: inv.Label,
			Error: "passwords do not match",
		})
		return
	}

	// Fetch the Jellyfin admin token stored at last admin login.
	adminToken, err := h.settings.GetJellyfinAdminToken(r.Context())
	if errors.Is(err, domain.ErrSettingNotFound) {
		h.render(w, invitePageData{
			Status:  "error",
			Message: "jellygate is not configured yet — ask your admin to sign in first",
		})
		return
	}
	if err != nil {
		slog.Error("handler.InviteHandler.HandleInviteSubmit: get jellyfin admin token", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	// Create the Jellyfin user.
	jellyfinUID, err := h.jellyfin.CreateUser(r.Context(), adminToken, username, password)
	if err != nil {
		h.render(w, invitePageData{
			Status: "form", Token: token, Label: inv.Label,
			Error: "could not create account — the username may already be taken",
		})
		return
	}

	// Apply library access policy (best-effort; user is already created).
	_ = h.jellyfin.SetLibraryAccess(r.Context(), adminToken, jellyfinUID, inv.LibraryIDs)

	// Record the registration (best-effort).
	reg := domain.Registration{
		ID:           uuid.NewString(),
		InviteID:     inv.ID,
		JellyfinUID:  jellyfinUID,
		Username:     username,
		RegisteredAt: time.Now(),
	}
	_ = h.registrations.Create(r.Context(), reg)

	// Increment invite use count (best-effort).
	_ = h.invites.IncrementUse(r.Context(), inv.ID)
	inv.UseCount++ // reflect the increment so the notification shows the correct count

	// Notify (best-effort).
	_ = h.notifier.InviteUsed(r.Context(), inv, username)

	// Redirect to tutorial/onboarding.
	http.Redirect(w, r, "/tutorial", http.StatusFound)
}

// --- helpers ---

func (h *InviteHandler) render(w http.ResponseWriter, data invitePageData) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.tmpl.ExecuteTemplate(w, "base", data); err != nil {
		http.Error(w, "render error", http.StatusInternalServerError)
	}
}

func inviteErrMessage(err error) string {
	switch {
	case errors.Is(err, domain.ErrInviteRevoked):
		return "this invite has been revoked"
	case errors.Is(err, domain.ErrInviteExpired):
		return "this invite has expired"
	case errors.Is(err, domain.ErrInviteExhausted):
		return "this invite has reached its maximum number of uses"
	default:
		return "this invite is no longer available"
	}
}
