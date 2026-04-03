package handler

import "net/http"

// Health handles GET /health for Traefik and other uptime checkers.
func Health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}
