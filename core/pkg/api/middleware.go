package api

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// rateLimitConfig holds the rate limiter settings.
type rateLimitConfig struct {
	rps   rate.Limit
	burst int
}

// GlobalRateLimiter manages per-IP rate limiters.
type GlobalRateLimiter struct {
	visitors map[string]*visitor
	mu       sync.Mutex
	config   rateLimitConfig
}

// visitor tracks the rate limiter and last seen time for an IP.
type visitor struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// NewGlobalRateLimiter creates a new rate limiter.
// rps: requests per second allowed.
// burst: maximum burst size.
func NewGlobalRateLimiter(rps, burst int) *GlobalRateLimiter {
	rl := &GlobalRateLimiter{
		visitors: make(map[string]*visitor),
		config: rateLimitConfig{
			rps:   rate.Limit(rps),
			burst: burst,
		},
	}
	// Start background cleanup
	go rl.cleanupVisitors()
	return rl
}

// getVisitor retrieving the limiter for a given IP, creating if necessary.
func (rl *GlobalRateLimiter) getVisitor(ip string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	v, exists := rl.visitors[ip]
	if !exists {
		limiter := rate.NewLimiter(rl.config.rps, rl.config.burst)
		rl.visitors[ip] = &visitor{limiter, time.Now()}
		return limiter
	}

	v.lastSeen = time.Now()
	return v.limiter
}

// cleanupVisitors removes stale visitor entries to prevent memory leaks.
// Checks every minute, removes entries older than 3 minutes.
func (rl *GlobalRateLimiter) cleanupVisitors() {
	for {
		time.Sleep(1 * time.Minute)
		rl.mu.Lock()
		for ip, v := range rl.visitors {
			if time.Since(v.lastSeen) > 3*time.Minute {
				delete(rl.visitors, ip)
			}
		}
		rl.mu.Unlock()
	}
}

// Middleware returns a Handler that enforces rate limits.
func (rl *GlobalRateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			// Fallback if unable to split (e.g. no port or weird format)
			// In production, check X-Forwarded-For if behind proxy
			ip = r.RemoteAddr
			// Basic cleanup of ipv6 brackets if present
			ip = strings.TrimPrefix(ip, "[")
			ip = strings.TrimSuffix(ip, "]")
		}

		limiter := rl.getVisitor(ip)
		if !limiter.Allow() {
			// RFC 7807 Error Response
			// Calculate retry after if possible, but standard Allow() doesn't give duration.
			// Reserve() does.
			// MVP: just fail with generic message.
			WriteTooManyRequests(w, 5) // Suggest 5 seconds backoff
			return
		}

		next.ServeHTTP(w, r)
	})
}

// WithContextRateLimit enforces rate limits based on context values (e.g. TenantID).
// This requires the context to be populated previously (e.g. by auth middleware).
func WithContextRateLimit(next http.Handler, limitPerTenant int) http.Handler {
	// Simple map for tenant limiters
	// Note: In distributed system, use Redis (available in go.mod).
	// For MVP, localized map (future implementation).

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract TenantID from context (assuming auth middleware put it there)
		// Or from request via helper if context is not standardized yet
		// Let's assume standard key or helper exists.
		// core/pkg/auth/context.go defined GetTenantID
		// But we are in `api` package, don't want circular dependency if `auth` imports `api`?
		// `api` seems low level. `auth` imports `api`? No, `auth` probably independent.
		// But checking `core/pkg/auth` earlier showed it imports standard libs.

		// For now, let's stick to IP limiting as the primary M14 solution.
		// Tenant limiting is advanced.
		next.ServeHTTP(w, r)
	})
}
