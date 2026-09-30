package handlers

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

// ipRateLimiter is a simple in-memory per-IP token bucket.
type ipRateLimiter struct {
	mu      sync.Mutex
	buckets map[string]*tokenBucket
	burst   float64
	refill  float64 // tokens per second
	lastGC  time.Time
}

type tokenBucket struct {
	tokens float64
	last   time.Time
}

func newIPRateLimiter(burst int, every time.Duration) *ipRateLimiter {
	return &ipRateLimiter{
		buckets: make(map[string]*tokenBucket),
		burst:   float64(burst),
		refill:  1 / every.Seconds(),
		lastGC:  time.Now(),
	}
}

// allow consumes a token for ip, reporting whether the request may proceed.
func (l *ipRateLimiter) allow(ip string) bool {
	now := time.Now()
	l.mu.Lock()
	defer l.mu.Unlock()

	// drop buckets that have fully refilled so the map cannot grow without bound
	if now.Sub(l.lastGC) > time.Minute {
		for key, b := range l.buckets {
			if b.tokens+now.Sub(b.last).Seconds()*l.refill >= l.burst {
				delete(l.buckets, key)
			}
		}
		l.lastGC = now
	}

	b, ok := l.buckets[ip]
	if !ok {
		b = &tokenBucket{tokens: l.burst, last: now}
		l.buckets[ip] = b
	}
	b.tokens += now.Sub(b.last).Seconds() * l.refill
	if b.tokens > l.burst {
		b.tokens = l.burst
	}
	b.last = now

	if b.tokens < 1 {
		return false
	}
	b.tokens--
	return true
}

// getClientIP returns the peer IP. The clientIP middleware has already rewritten
// RemoteAddr from trusted proxy headers, so they are not consulted again here.
func getClientIP(r *http.Request) string {
	addr := strings.TrimSpace(r.RemoteAddr)
	if host, _, err := net.SplitHostPort(addr); err == nil {
		return host
	}
	return addr
}

// rateLimited writes a 429 and returns true if the client has exhausted its bucket.
func rateLimited(w http.ResponseWriter, r *http.Request, l *ipRateLimiter) bool {
	if l.allow(getClientIP(r)) {
		return false
	}
	w.Header().Set("Retry-After", "10")
	WriteAPIError(w, http.StatusTooManyRequests, "TooManyRequests", "Too many attempts. Please try again later.")
	return true
}
