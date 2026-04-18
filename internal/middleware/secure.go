package middleware

import "net/http"

// SecureHeaders sets defensive HTTP response headers on every request.
func SecureHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Frame-Options", "DENY")
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		h.Set("Content-Security-Policy",
			"default-src 'self'; "+
				"script-src 'self' 'unsafe-inline'; "+
				"style-src 'self' 'unsafe-inline' fonts.googleapis.com; "+
				"font-src 'self' fonts.gstatic.com; "+
				"img-src 'self' data: lh3.googleusercontent.com; "+
				"frame-ancestors 'none'")
		next.ServeHTTP(w, r)
	})
}
