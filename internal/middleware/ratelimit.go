package middleware

import (
	"net"
	"net/http"
	"sync"
	"time"
)

// RateLimit is middleware that limits requests per IP to max per window.
// It trusts X-Forwarded-For when behindProxy is true (e.g. behind Traefik).
func RateLimit(max int, window time.Duration, behindProxy bool) func(http.Handler) http.Handler {
	type entry struct {
		count     int
		windowEnd time.Time
	}
	var (
		mu      sync.Mutex
		buckets = map[string]*entry{}
	)

	// Periodically clean up stale buckets to prevent unbounded growth.
	go func() {
		for range time.Tick(window) {
			mu.Lock()
			now := time.Now()
			for ip, e := range buckets {
				if now.After(e.windowEnd) {
					delete(buckets, ip)
				}
			}
			mu.Unlock()
		}
	}()

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := clientIP(r, behindProxy)

			mu.Lock()
			e, ok := buckets[ip]
			if !ok || time.Now().After(e.windowEnd) {
				e = &entry{windowEnd: time.Now().Add(window)}
				buckets[ip] = e
			}
			e.count++
			over := e.count > max
			mu.Unlock()

			if over {
				http.Error(w, "too many requests", http.StatusTooManyRequests)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// clientIP extracts the real client IP, optionally reading X-Forwarded-For.
func clientIP(r *http.Request, behindProxy bool) string {
	if behindProxy {
		if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
			// X-Forwarded-For may be a comma-separated list; take the first.
			if idx := len(fwd); idx > 0 {
				for i, c := range fwd {
					if c == ',' {
						fwd = fwd[:i]
						break
					}
				}
			}
			if ip := net.ParseIP(fwd); ip != nil {
				return ip.String()
			}
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
