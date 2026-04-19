package main

import (
	"context"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/rmewborne/jellygate/internal/auth"
	"github.com/rmewborne/jellygate/internal/domain"
	"github.com/rmewborne/jellygate/internal/handler"
	"github.com/rmewborne/jellygate/internal/jellyfin"
	"github.com/rmewborne/jellygate/internal/middleware"
	"github.com/rmewborne/jellygate/internal/notifications"
	"github.com/rmewborne/jellygate/internal/store"
	"github.com/rmewborne/jellygate/web"
)

// version is set at build time: -ldflags "-X main.version=v1.2.3"
var version = "dev"

func main() {
	ctx := context.Background()

	// --- config ---
	addr        := envOr("LISTEN_ADDR", ":8080")
	dbURL       := mustEnv("DATABASE_URL")
	jfURL       := mustEnv("JELLYFIN_URL")
	baseURL     := mustEnv("BASE_URL")
	discordURL  := os.Getenv("DISCORD_WEBHOOK_URL")
	mediaURL    := mustEnv("MEDIA_URL")
	seerrURL    := os.Getenv("SEERR_URL")
	secure      := os.Getenv("SECURE_COOKIES") != "false"
	behindProxy := os.Getenv("BEHIND_PROXY") == "true"

	// --- database ---
	pool, err := store.Open(ctx, dbURL)
	if err != nil {
		slog.Error("db connect", "err", err)
		os.Exit(1)
	}
	defer pool.Close()

	if err := store.Migrate(dbURL); err != nil {
		slog.Error("migrate", "err", err)
		os.Exit(1)
	}
	slog.Info("migrations applied")

	// --- stores ---
	inviteStore       := store.NewInviteStore(pool)
	sessionStore      := store.NewSessionStore(pool)
	registrationStore := store.NewRegistrationStore(pool)
	settingsStore     := store.NewSettingsStore(pool)

	// --- jellyfin client ---
	jf := jellyfin.New(jfURL)
	slog.Info("jellyfin configured", "url", jfURL)
	slog.Info("base url configured", "baseURL", baseURL)

	// --- notifier ---
	notifier := buildNotifier(discordURL)

	// --- handlers ---
	sessionMgr := auth.NewManager(sessionStore, secure)

	adminHandler, err := handler.NewAdmin(sessionMgr, inviteStore, jf, settingsStore, baseURL, secure)
	if err != nil {
		slog.Error("admin handler init", "err", err)
		os.Exit(1)
	}

	inviteHandler, err := handler.NewInviteHandler(
		inviteStore, registrationStore, jf, notifier, settingsStore,
	)
	if err != nil {
		slog.Error("invite handler init", "err", err)
		os.Exit(1)
	}

	tutorialHandler, err := handler.NewTutorialHandler(mediaURL, seerrURL)
	if err != nil {
		slog.Error("tutorial handler init", "err", err)
		os.Exit(1)
	}

	// --- routes ---
	mux := http.NewServeMux()

	// Static assets embedded in the binary.
	staticFS, err := fs.Sub(web.FS, "static")
	if err != nil {
		slog.Error("static assets init", "error", err)
		os.Exit(1)
	}
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServerFS(staticFS)))

	// Health — no auth, no rate limit; used by reverse proxy.
	mux.HandleFunc("GET /health", handler.Health)

	// Root redirect.
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		http.Redirect(w, r, "/admin", http.StatusFound)
	})

	// Admin — unauthenticated.
	mux.HandleFunc("GET /admin/login", adminHandler.HandleLoginForm)
	mux.HandleFunc("POST /admin/login", adminHandler.HandleLogin)

	// Admin — requires session.
	requireSession := middleware.RequireSession(sessionMgr)
	mux.Handle("POST /admin/logout",
		requireSession(http.HandlerFunc(adminHandler.HandleLogout)))
	mux.Handle("GET /admin",
		requireSession(http.HandlerFunc(adminHandler.HandleDashboard)))
	mux.Handle("POST /admin/invites",
		requireSession(http.HandlerFunc(adminHandler.HandleCreateInvite)))
	mux.Handle("POST /admin/invites/{id}/revoke",
		requireSession(http.HandlerFunc(adminHandler.HandleRevokeInvite)))

	// Invite registration — rate limited (10 requests/hour per IP).
	rateLimit := middleware.RateLimit(10, time.Hour, behindProxy)
	mux.Handle("GET /invite/{token}",
		rateLimit(http.HandlerFunc(inviteHandler.HandleInviteForm)))
	mux.Handle("POST /invite/{token}",
		rateLimit(http.HandlerFunc(inviteHandler.HandleInviteSubmit)))

	// Tutorial/onboarding — no auth required (shown after registration).
	mux.HandleFunc("GET /tutorial", tutorialHandler.HandleTutorial)
	mux.HandleFunc("GET /tutorial/skip", tutorialHandler.HandleTutorialSkip)
	mux.HandleFunc("GET /tutorial/complete", tutorialHandler.HandleTutorialComplete)

	// Wrap everything in security headers.
	srv := middleware.SecureHeaders(mux)

	slog.Info("jellygate starting", "version", version, "addr", addr)
	if err := http.ListenAndServe(addr, srv); err != nil {
		slog.Error("server stopped", "err", err)
		os.Exit(1)
	}
}

func buildNotifier(webhookURL string) domain.Notifier {
	if webhookURL != "" {
		slog.Info("discord notifications enabled")
		return notifications.NewDiscordNotifier(webhookURL)
	}
	return &notifications.NoopNotifier{}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		slog.Error("required environment variable not set", "key", key)
		os.Exit(1)
	}
	return v
}
