package middleware

import (
	"encoding/json"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// RateLimiter implements IP-based rate limiting for security
type RateLimiter struct {
	visitors map[string]*Visitor
	mu       sync.RWMutex
	rate     rate.Limit
	burst    int
}

// Visitor tracks rate limiting state for a single IP
type Visitor struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// SecurityRateLimit creates a rate limiting middleware for security
// rate: requests per second per IP (default: 100/60 = ~1.7 per second)
// burst: maximum burst of requests (default: 10)
func SecurityRateLimit(requestsPerSecond float64, burst int) func(http.Handler) http.Handler {
	limiter := &RateLimiter{
		visitors: make(map[string]*Visitor),
		rate:     rate.Limit(requestsPerSecond),
		burst:    burst,
	}

	// Start cleanup goroutine
	go limiter.cleanupVisitors()

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := extractIP(r)
			if ip == "" {
				// If we can't extract IP, allow request but log
				log.Printf("Rate limiting: unable to extract IP from %s", r.RemoteAddr)
				next.ServeHTTP(w, r)
				return
			}

			if !limiter.Allow(ip) {
				log.Printf("Rate limit exceeded for IP: %s", ip)
				writeRateLimitErrorResponse(w)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// Allow checks if an IP is allowed to make a request
func (rl *RateLimiter) Allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	// If rate is 0, allow all requests
	if rl.rate == 0 {
		return true
	}

	visitor, exists := rl.visitors[ip]
	if !exists {
		// Create new limiter for this IP
		limiter := rate.NewLimiter(rl.rate, rl.burst)
		rl.visitors[ip] = &Visitor{limiter, time.Now()}
		return limiter.Allow()
	}

	// Update last seen time
	visitor.lastSeen = time.Now()
	return visitor.limiter.Allow()
}

// cleanupVisitors removes old visitors to prevent memory leaks
func (rl *RateLimiter) cleanupVisitors() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		rl.mu.Lock()
		for ip, visitor := range rl.visitors {
			// Remove visitors not seen in the last 10 minutes
			if time.Since(visitor.lastSeen) > 10*time.Minute {
				delete(rl.visitors, ip)
			}
		}
		rl.mu.Unlock()
	}
}

// extractIP extracts the real client IP from request
func extractIP(r *http.Request) string {
	// Try X-Forwarded-For header first (for proxies/load balancers)
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		// X-Forwarded-For can contain multiple IPs, take the first one
		ips := strings.Split(xff, ",")
		ip := strings.TrimSpace(ips[0])
		if isValidIP(ip) {
			return ip
		}
	}

	// Try X-Real-IP header
	xri := r.Header.Get("X-Real-IP")
	if xri != "" && isValidIP(xri) {
		return xri
	}

	// Fall back to RemoteAddr
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		// If SplitHostPort fails, try using RemoteAddr directly
		if isValidIP(r.RemoteAddr) {
			return r.RemoteAddr
		}
		return ""
	}

	if isValidIP(host) {
		return host
	}

	return ""
}

// isValidIP checks if the string is a valid IP address
func isValidIP(ip string) bool {
	return net.ParseIP(ip) != nil
}

// writeRateLimitErrorResponse writes a rate limit error response
func writeRateLimitErrorResponse(w http.ResponseWriter) {
	response := map[string]string{
		"error": "Too many requests",
		"code":  "RATE_LIMIT_EXCEEDED",
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Retry-After", "60") // Suggest retry after 60 seconds
	w.WriteHeader(http.StatusTooManyRequests)

	json.NewEncoder(w).Encode(response)
}