package http

import (
	"context"
	"crypto/subtle"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// defaultTrustedCIDRs contains loopback and private subnets where trusted reverse proxies typically run.
var defaultTrustedCIDRs = []string{
	"127.0.0.0/8",    // IPv4 loopback
	"::1/128",        // IPv6 loopback
	"10.0.0.0/8",     // RFC1918
	"172.16.0.0/12",  // RFC1918 / Docker default subnets
	"192.168.0.0/16", // RFC1918
	"fc00::/7",       // IPv6 ULA
}

var (
	trustedProxiesOnce sync.Once
	trustedProxyNets   []*net.IPNet
)

func initTrustedProxies() {
	var cidrs []string
	if custom := os.Getenv("TRUSTED_PROXIES"); custom != "" {
		for _, part := range strings.Split(custom, ",") {
			trimmed := strings.TrimSpace(part)
			if trimmed != "" {
				cidrs = append(cidrs, trimmed)
			}
		}
	} else {
		cidrs = defaultTrustedCIDRs
	}

	for _, cidr := range cidrs {
		if !strings.Contains(cidr, "/") {
			if ip := net.ParseIP(cidr); ip != nil {
				if ip.To4() != nil {
					cidr += "/32"
				} else {
					cidr += "/128"
				}
			}
		}
		_, ipNet, err := net.ParseCIDR(cidr)
		if err == nil && ipNet != nil {
			trustedProxyNets = append(trustedProxyNets, ipNet)
		}
	}
}

func isTrustedProxy(ipStr string) bool {
	trustedProxiesOnce.Do(initTrustedProxies)
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return false
	}
	for _, ipNet := range trustedProxyNets {
		if ipNet.Contains(ip) {
			return true
		}
	}
	return false
}

// getClientIP extracts the real IP address from proxy headers if and only if
// the direct remote address is a trusted proxy. If the direct remote address is
// NOT trusted, any forwarding headers are ignored completely to prevent spoofing.
func getClientIP(r *http.Request) string {
	remoteHost, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		remoteHost = r.RemoteAddr
	}
	remoteHost = strings.TrimSpace(remoteHost)

	// If remote host is not an authorized reverse proxy, strictly use the socket remote address.
	if !isTrustedProxy(remoteHost) {
		return remoteHost
	}

	// Remote host is a trusted reverse proxy, we can inspect forwarding headers.
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		trimmed := strings.TrimSpace(xri)
		if net.ParseIP(trimmed) != nil {
			return trimmed
		}
	}

	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		// Parse from right to left, skipping trusted proxies
		for i := len(parts) - 1; i >= 0; i-- {
			candidate := strings.TrimSpace(parts[i])
			if candidate == "" {
				continue
			}
			if net.ParseIP(candidate) == nil {
				continue
			}
			if !isTrustedProxy(candidate) {
				return candidate
			}
		}
		// If all were trusted proxies, return leftmost valid IP
		for _, part := range parts {
			candidate := strings.TrimSpace(part)
			if net.ParseIP(candidate) != nil {
				return candidate
			}
		}
	}

	return remoteHost
}

// ipRateLimiter implements rate limiting with Redis support and in-memory fallback.
type ipRateLimiter struct {
	name     string
	mu       sync.Mutex
	limits   map[string]*clientBucket
	rate     int           // allowed requests per window
	window   time.Duration // window duration
	cleanDur time.Duration
	rdb      *redis.Client
}

type clientBucket struct {
	tokens    int
	lastReset time.Time
}

func newIPRateLimiter(name string, rate int, window time.Duration, rdb ...*redis.Client) *ipRateLimiter {
	var client *redis.Client
	if len(rdb) > 0 {
		client = rdb[0]
	}
	limiter := &ipRateLimiter{
		name:     name,
		limits:   make(map[string]*clientBucket),
		rate:     rate,
		window:   window,
		cleanDur: window * 2,
		rdb:      client,
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
	// Try Redis-based rate limiting first if redis client configured
	if l.rdb != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
		defer cancel()

		key := fmt.Sprintf("ratelimit:%s:%s", l.name, ip)
		pipe := l.rdb.Pipeline()
		incrCmd := pipe.Incr(ctx, key)
		pipe.Expire(ctx, key, l.window)
		_, err := pipe.Exec(ctx)
		if err == nil {
			val := incrCmd.Val()
			return val <= int64(l.rate)
		}
		// If Redis fails, smoothly fall through to in-memory limiter
	}

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

// withAdminAuth ensures the request contains a valid admin secret in
// the X-Admin-Secret header or Bearer authorization header.
// Query parameters are rejected to prevent credentials from being logged in URLs.
func (rt *Router) withAdminAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		expected := rt.adminSecret
		if expected == "" {
			writeError(w, http.StatusForbidden, "ADMIN_DISABLED", "administrative endpoints are disabled")
			return
		}

		secret := r.Header.Get("X-Admin-Secret")
		if secret == "" {
			authHeader := r.Header.Get("Authorization")
			if strings.HasPrefix(authHeader, "Bearer ") {
				secret = strings.TrimPrefix(authHeader, "Bearer ")
			}
		}

		if subtle.ConstantTimeCompare([]byte(secret), []byte(expected)) != 1 {
			writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "valid administrative credentials required")
			return
		}
		next(w, r)
	}
}
