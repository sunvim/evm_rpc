package middleware

import (
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"
	"github.com/sunvim/evm_rpc/pkg/logger"
	"github.com/sunvim/evm_rpc/pkg/metrics"
)

// RateLimiter manages rate limiting for RPC requests
type RateLimiter struct {
	global       *rate.Limiter
	ipLimiters   sync.Map // map[string]*rate.Limiter
	methodLimits map[string]int
	ipRate       int
	ipBurst      int
	enabled      bool
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(enabled bool, globalRate, globalBurst, ipRate, ipBurst int, methodLimits map[string]int) *RateLimiter {
	var global *rate.Limiter
	if globalRate > 0 {
		global = rate.NewLimiter(rate.Limit(globalRate), globalBurst)
	}

	return &RateLimiter{
		global:       global,
		ipLimiters:   sync.Map{},
		methodLimits: methodLimits,
		ipRate:       ipRate,
		ipBurst:      ipBurst,
		enabled:      enabled,
	}
}

// getIPLimiter returns or creates a rate limiter for an IP address
func (rl *RateLimiter) getIPLimiter(ip string) *rate.Limiter {
	if rl.ipRate <= 0 {
		return nil
	}

	limiter, ok := rl.ipLimiters.Load(ip)
	if !ok {
		limiter = rate.NewLimiter(rate.Limit(rl.ipRate), rl.ipBurst)
		rl.ipLimiters.Store(ip, limiter)
	}
	return limiter.(*rate.Limiter)
}

// Allow checks if a request should be allowed based on rate limits
func (rl *RateLimiter) Allow(ip, method string) (bool, string) {
	if !rl.enabled {
		return true, ""
	}

	// Check global rate limit
	if rl.global != nil && !rl.global.Allow() {
		metrics.RecordRateLimit("global")
		logger.Warnf("Global rate limit exceeded for IP %s, method %s", ip, method)
		return false, "global"
	}

	// Check IP-based rate limit
	if ipLimiter := rl.getIPLimiter(ip); ipLimiter != nil && !ipLimiter.Allow() {
		metrics.RecordRateLimit("ip")
		logger.Warnf("IP rate limit exceeded for IP %s, method %s", ip, method)
		return false, "ip"
	}

	// Check method-based rate limit
	if methodRate, ok := rl.methodLimits[method]; ok && methodRate > 0 {
		// For method-based limits, we use a per-method limiter
		// This is a simplified approach; in production, you might want per-IP-per-method limiters
		key := "method:" + method
		limiter, _ := rl.ipLimiters.LoadOrStore(key, rate.NewLimiter(rate.Limit(methodRate), methodRate))
		if !limiter.(*rate.Limiter).Allow() {
			metrics.RecordRateLimit("method")
			logger.Warnf("Method rate limit exceeded for IP %s, method %s", ip, method)
			return false, "method"
		}
	}

	return true, ""
}

// Cleanup removes old IP limiters (should be called periodically)
func (rl *RateLimiter) Cleanup(maxAge time.Duration) {
	// This is a simple implementation; in production, you might want to track last access time
	// For now, we'll just keep all limiters
}

// Middleware creates an HTTP middleware for rate limiting
func (rl *RateLimiter) Middleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !rl.enabled {
				next.ServeHTTP(w, r)
				return
			}

			// Extract IP from request
			ip := extractIP(r)

			// For middleware, we check global and IP limits only
			// Method-specific limits are checked in the handler
			if rl.global != nil && !rl.global.Allow() {
				metrics.RecordRateLimit("global")
				http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
				return
			}

			if ipLimiter := rl.getIPLimiter(ip); ipLimiter != nil && !ipLimiter.Allow() {
				metrics.RecordRateLimit("ip")
				http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// extractIP extracts the client IP address from the request
func extractIP(r *http.Request) string {
	// Try X-Forwarded-For header first
	ip := r.Header.Get("X-Forwarded-For")
	if ip != "" {
		return ip
	}

	// Try X-Real-IP header
	ip = r.Header.Get("X-Real-IP")
	if ip != "" {
		return ip
	}

	// Fall back to RemoteAddr
	return r.RemoteAddr
}
