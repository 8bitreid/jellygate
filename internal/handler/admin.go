package handler

import (
	"context"
	"errors"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"

	"github.com/rmewborne/jellygate/internal/auth"
	"github.com/rmewborne/jellygate/internal/domain"
	"github.com/rmewborne/jellygate/web"
)

// Admin handles all /admin/* routes.
type Admin struct {
	sessions      *auth.Manager
	invites       domain.InviteStore
	registrations domain.RegistrationStore
	jellyfin      domain.JellyfinClient
	settings      domain.SettingsStore
	baseURL       string
	secure        bool
	loginTmpl     *template.Template
	dashboardTmpl *template.Template
}

// NewAdmin constructs an Admin handler.
// baseURL is used to build invite links (e.g. "https://invites.example.com").
func NewAdmin(
	sessions *auth.Manager,
	invites domain.InviteStore,
	registrations domain.RegistrationStore,
	jellyfin domain.JellyfinClient,
	settings domain.SettingsStore,
	baseURL string,
	secure bool,
) (*Admin, error) {
	loginTmpl, err := template.ParseFS(web.FS, "templates/base.html", "templates/login.html")
	if err != nil {
		return nil, fmt.Errorf("handler.NewAdmin: parse login template: %w", err)
	}
	dashboardTmpl, err := template.ParseFS(web.FS, "templates/base.html", "templates/dashboard.html")
	if err != nil {
		return nil, fmt.Errorf("handler.NewAdmin: parse dashboard template: %w", err)
	}
	return &Admin{
		sessions:      sessions,
		invites:       invites,
		registrations: registrations,
		jellyfin:      jellyfin,
		settings:      settings,
		baseURL:       baseURL,
		secure:        secure,
		loginTmpl:     loginTmpl,
		dashboardTmpl: dashboardTmpl,
	}, nil
}

// --- view models ---

type loginData struct {
	CSRFToken string
	Flash     string
	FlashType string
	IsSetup   bool // true on first-ever login — shows setup messaging
}

type inviteView struct {
	domain.Invite
	URL              string
	Status           string
	StatusClass      string
	CanRevoke        bool
	RegistrationCount int
}

type dashboardData struct {
	Username    string
	CSRFToken   string
	Flash       string
	FlashType   string
	Invites     []inviteView
	Libraries   []domain.Library
	ActiveTab   string
	ActiveCount int
	UsedCount   int
	ExpiredCount int
	RevokedCount int
}

// --- handlers ---

// HandleLoginForm renders GET /admin/login.
func (a *Admin) HandleLoginForm(w http.ResponseWriter, r *http.Request) {
	csrfToken, err := auth.SetCSRFCookie(w, a.secure)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	// Detect first-run: no admin token stored yet.
	_, settingsErr := a.settings.GetJellyfinAdminToken(r.Context())
	if settingsErr != nil && !errors.Is(settingsErr, domain.ErrSettingNotFound) {
		slog.Error("handler.Admin.HandleLoginForm: get jellyfin admin token", "error", settingsErr)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	isSetup := errors.Is(settingsErr, domain.ErrSettingNotFound)

	data := loginData{CSRFToken: csrfToken, IsSetup: isSetup}
	if msg := r.URL.Query().Get("error"); msg != "" {
		data.Flash = msg
		data.FlashType = "error"
	}
	a.render(w, a.loginTmpl, data)
}

// HandleLogin processes POST /admin/login.
func (a *Admin) HandleLogin(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if !auth.ValidateCSRF(r) {
		http.Error(w, "invalid CSRF token", http.StatusForbidden)
		return
	}

	username := r.FormValue("username")
	password := r.FormValue("password")

	jfToken, err := a.jellyfin.Authenticate(r.Context(), username, password)
	if err != nil {
		slog.Error("jellyfin login failed", "user", username, "err", err)
		http.Redirect(w, r, "/admin/login?error=invalid+credentials", http.StatusSeeOther)
		return
	}

	// Persist the admin token so the invite handler can create users without
	// requiring a live admin session.
	if err := a.settings.SetJellyfinAdminToken(r.Context(), jfToken); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	if err := a.sessions.Create(r.Context(), w, username, jfToken); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/admin", http.StatusSeeOther)
}

// HandleLogout processes POST /admin/logout.
func (a *Admin) HandleLogout(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err == nil {
		auth.ValidateCSRF(r) // best-effort; always log out
	}
	a.sessions.Delete(r.Context(), w, r)
	http.Redirect(w, r, "/admin/login", http.StatusSeeOther)
}

// HandleDashboard renders GET /admin.
func (a *Admin) HandleDashboard(w http.ResponseWriter, r *http.Request) {
	sess, _ := auth.FromContext(r.Context())

	csrfToken, err := auth.SetCSRFCookie(w, a.secure)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	invites, err := a.invites.List(r.Context())
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	libs, _ := a.jellyfin.ListLibraries(r.Context(), sess.JellyfinToken)

	// Convert all invites to views to compute counts and statuses
	allViews := a.toViews(r.Context(), invites)

	// Get the active tab from query params, default to "active"
	activeTab := r.URL.Query().Get("tab")
	if activeTab == "" {
		activeTab = "active"
	}

	// Compute counts for each tab
	var activeCount, usedCount, expiredCount, revokedCount int
	for _, v := range allViews {
		switch v.StatusClass {
		case "active":
			activeCount++
		case "revoked":
			revokedCount++
		case "expired", "exhausted":
			expiredCount++
		}
		if v.RegistrationCount > 0 {
			usedCount++
		}
	}

	// Filter invites based on active tab
	var filteredViews []inviteView
	for _, v := range allViews {
		switch activeTab {
		case "active":
			if v.StatusClass == "active" {
				filteredViews = append(filteredViews, v)
			}
		case "used":
			if v.RegistrationCount > 0 {
				filteredViews = append(filteredViews, v)
			}
		case "expired":
			if v.StatusClass == "expired" || v.StatusClass == "exhausted" {
				filteredViews = append(filteredViews, v)
			}
		case "revoked":
			if v.StatusClass == "revoked" {
				filteredViews = append(filteredViews, v)
			}
		}
	}

	data := dashboardData{
		Username:     sess.Username,
		CSRFToken:    csrfToken,
		Invites:      filteredViews,
		Libraries:    libs,
		ActiveTab:    activeTab,
		ActiveCount:  activeCount,
		UsedCount:    usedCount,
		ExpiredCount: expiredCount,
		RevokedCount: revokedCount,
	}
	if msg := r.URL.Query().Get("flash"); msg != "" {
		data.Flash = msg
		data.FlashType = r.URL.Query().Get("type")
		if data.FlashType == "" {
			data.FlashType = "success"
		}
	}
	a.render(w, a.dashboardTmpl, data)
}

// HandleCreateInvite processes POST /admin/invites.
func (a *Admin) HandleCreateInvite(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if !auth.ValidateCSRF(r) {
		http.Error(w, "invalid CSRF token", http.StatusForbidden)
		return
	}

	sess, _ := auth.FromContext(r.Context())
	inv := domain.Invite{
		ID:         uuid.NewString(),
		Token:      mustGenerateToken(),
		Label:      r.FormValue("label"),
		CreatedBy:  sess.Username,
		LibraryIDs: r.Form["library_ids"],
	}

	if raw := r.FormValue("expires_at"); raw != "" {
		t, err := time.ParseInLocation("2006-01-02T15:04", raw, time.Local)
		if err == nil {
			inv.ExpiresAt = &t
		}
	}
	if raw := r.FormValue("max_uses"); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n > 0 {
			inv.MaxUses = &n
		}
	}

	if err := a.invites.Create(r.Context(), inv); err != nil {
		http.Redirect(w, r, "/admin?flash=failed+to+create+invite&type=error", http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/admin?flash=invite+created", http.StatusSeeOther)
}

// HandleRevokeInvite processes POST /admin/invites/{id}/revoke.
func (a *Admin) HandleRevokeInvite(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if !auth.ValidateCSRF(r) {
		http.Error(w, "invalid CSRF token", http.StatusForbidden)
		return
	}

	id := r.PathValue("id")
	if err := a.invites.Revoke(r.Context(), id); err != nil {
		http.Redirect(w, r, "/admin?flash=failed+to+revoke+invite&type=error", http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/admin?flash=invite+revoked", http.StatusSeeOther)
}

// --- helpers ---

func (a *Admin) render(w http.ResponseWriter, tmpl *template.Template, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.ExecuteTemplate(w, "base", data); err != nil {
		http.Error(w, "render error", http.StatusInternalServerError)
	}
}

func (a *Admin) toViews(ctx context.Context, invites []domain.Invite) []inviteView {
	views := make([]inviteView, len(invites))
	for i, inv := range invites {
		regCount, _ := a.registrations.CountByInviteID(ctx, inv.ID)
		v := inviteView{
			Invite:            inv,
			URL:               a.baseURL + "/invite/" + inv.Token,
			CanRevoke:         !inv.Revoked,
			RegistrationCount: regCount,
		}
		switch {
		case inv.Revoked:
			v.Status, v.StatusClass = "revoked", "revoked"
		case inv.ExpiresAt != nil && time.Now().After(*inv.ExpiresAt):
			v.Status, v.StatusClass = "expired", "expired"
		case inv.MaxUses != nil && inv.UseCount >= *inv.MaxUses:
			v.Status, v.StatusClass = "exhausted", "exhausted"
		default:
			v.Status, v.StatusClass = "active", "active"
		}
		views[i] = v
	}
	return views
}

func mustGenerateToken() string {
	tok, err := auth.GenerateToken()
	if err != nil {
		panic(fmt.Sprintf("mustGenerateToken: %v", err))
	}
	return tok
}
