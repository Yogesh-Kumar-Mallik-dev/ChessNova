package http

import (
	"crypto/subtle"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

// ipRateLimiter implements a sliding-window token bucket per IP address.
type ipRateLimiter struct {
	mu       sync.Mutex
	limits   map[string]*clientBucket
	rate     int           // allowed requests per window
	window   time.Duration // window duration
	cleanDur time.Duration
}

type clientBucket struct {
	tokens    int
	lastReset time.Time
}

func newIPRateLimiter(rate int, window time.Duration) *ipRateLimiter {
	limiter := &ipRateLimiter{
		limits:   make(map[string]*clientBucket),
		rate:     rate,
		window:   window,
		cleanDur: window * 2,
	}

	go func() {
		ticker := time.NewTicker(window * 2)
		for range ticker.C {
			limiter.cleanup()
		}
	}()

	return limiter
}

func (l *ipRateLimiter) allow(ip string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	bucket, exists := l.limits[ip]
	if !exists || now.Sub(bucket.lastReset) >= l.window {
		l.limits[ip] = &clientBucket{
			tokens:    1,
			lastReset: now,
		}
		return true
	}

	if bucket.tokens < l.rate {
		bucket.tokens++
		return true
	}

	return false
}

func (l *ipRateLimiter) cleanup() {
	l.mu.Lock()
	defer l.mu.Unlock()

	cutoff := time.Now().Add(-l.cleanDur)
	for ip, bucket := range l.limits {
		if bucket.lastReset.Before(cutoff) {
			delete(l.limits, ip)
		}
	}
}

// getClientIP extracts the real IP address from proxy headers or RemoteAddr.
func getClientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		if len(parts) > 0 {
			ip := strings.TrimSpace(parts[0])
			if ip != "" {
				return ip
			}
		}
	}
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return strings.TrimSpace(xri)
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}
	return r.RemoteAddr
}

// Middleware wraps an HTTP handler func with IP-based rate limiting.
func (l *ipRateLimiter) Middleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ip := getClientIP(r)
		if !l.allow(ip) {
			writeError(w, http.StatusTooManyRequests, "RATE_LIMIT_EXCEEDED", "too many requests, please slow down")
			return
		}
		next(w, r)
	}
}

// withAdminAuth ensures the request contains a valid admin secret either in
// the X-Admin-Secret header or as a 'secret' query parameter.
func (rt *Router) withAdminAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		secret := r.Header.Get("X-Admin-Secret")
		if secret == "" {
			secret = r.URL.Query().Get("secret")
		}
		expected := rt.adminSecret
		if expected == "" {
			writeError(w, http.StatusForbidden, "ADMIN_DISABLED", "admin actions are disabled")
			return
		}
		if subtle.ConstantTimeCompare([]byte(secret), []byte(expected)) != 1 {
			writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "valid admin secret required")
			return
		}
		next(w, r)
	}
}
