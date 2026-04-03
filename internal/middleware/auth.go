package middleware

import (
	"errors"
	"net/http"

	"github.com/rmewborne/jellygate/internal/auth"
	"github.com/rmewborne/jellygate/internal/domain"
)

// RequireSession is middleware that rejects unauthenticated requests.
// Valid sessions are stored in the request context via auth.FromContext.
func RequireSession(mgr *auth.Manager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			sess, err := mgr.Get(r.Context(), r)
			if err != nil {
				if errors.Is(err, domain.ErrSessionNotFound) || errors.Is(err, domain.ErrSessionExpired) {
					http.Redirect(w, r, "/admin/login", http.StatusSeeOther)
					return
				}
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
			next.ServeHTTP(w, r.WithContext(auth.WithSessionCtx(r.Context(), sess)))
		})
	}
}
