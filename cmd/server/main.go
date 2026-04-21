package main

import (
	"context"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/rmewborne/jellygate/internal/auth"
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
	baseURL     := os.Getenv("BASE_URL") // optional — derived from request when empty
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
	jf := jellyfin.New(settingsStore)

	// --- notifier ---
	notifier := notifications.NewDiscordNotifier(settingsStore)

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

	tutorialHandler, err := handler.NewTutorialHandler(settingsStore)
	if err != nil {
		slog.Error("tutorial handler init", "err", err)
		os.Exit(1)
	}

	setupHandler, err := handler.NewSetupHandler(settingsStore)
	if err != nil {
		slog.Error("setup handler init", "err", err)
		os.Exit(1)
	}

	settingsHandler, err := handler.NewSettingsHandler(settingsStore, secure)
	if err != nil {
		slog.Error("settings handler init", "err", err)
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

	// First-run setup — no auth required; accessible before Jellyfin URL is configured.
	mux.HandleFunc("GET /setup", setupHandler.HandleSetupForm)
	mux.HandleFunc("POST /setup", setupHandler.HandleSetupSubmit)

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
	mux.Handle("GET /admin/settings",
		requireSession(http.HandlerFunc(settingsHandler.HandleSettingsForm)))
	mux.Handle("POST /admin/settings",
		requireSession(http.HandlerFunc(settingsHandler.HandleSettingsSubmit)))

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

	// Wrap everything in security headers and setup guard.
	srv := middleware.SecureHeaders(middleware.RequireSetup(settingsStore)(mux))

	slog.Info("jellygate starting", "version", version, "addr", addr)
	if err := http.ListenAndServe(addr, srv); err != nil {
		slog.Error("server stopped", "err", err)
		os.Exit(1)
	}
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
