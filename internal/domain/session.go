package domain

import "time"

// Session represents an authenticated admin session.
type Session struct {
	Token         string
	Username      string
	JellyfinToken string    // Jellyfin access token for subsequent API calls
	ExpiresAt     time.Time
}

// IsExpired reports whether the session has passed its expiry time.
func (s *Session) IsExpired() bool {
	return time.Now().After(s.ExpiresAt)
}
