package middleware

import (
	"errors"
	"net/http"
	"strings"

	"github.com/rmewborne/jellygate/internal/domain"
)

// RequireSetup redirects to /setup when the Jellyfin URL has not been configured yet.
// Exempt paths: /setup, /health, and anything under /static/.
func RequireSetup(settings domain.SettingsStore) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			p := r.URL.Path
			if p == "/setup" || p == "/health" || strings.HasPrefix(p, "/static/") {
				next.ServeHTTP(w, r)
				return
			}
			_, err := settings.GetJellyfinURL(r.Context())
			if errors.Is(err, domain.ErrSettingNotFound) {
				http.Redirect(w, r, "/setup", http.StatusSeeOther)
				return
			}
			if err != nil {
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
