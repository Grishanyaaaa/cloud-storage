package middleware

import (
	"context"
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// limiterEntry tracks a rate limiter and its last access time.
type limiterEntry struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// RateLimiter implements a simple in-memory rate limiter per IP address.
type RateLimiter struct {
	limiters map[string]*limiterEntry
	mu       sync.RWMutex
	rate     rate.Limit
	burst    int
	stopCh   chan struct{}
	once     sync.Once
}

// NewRateLimiter creates a new rate limiter.
// rate: requests per second
// burst: maximum burst size
func NewRateLimiter(r rate.Limit, b int) *RateLimiter {
	rl := &RateLimiter{
		limiters: make(map[string]*limiterEntry),
		rate:     r,
		burst:    b,
		stopCh:   make(chan struct{}),
	}
	// Start cleanup goroutine once
	go rl.cleanupLimiters()
	return rl
}

// Stop stops the cleanup goroutine.
func (rl *RateLimiter) Stop() {
	rl.once.Do(func() {
		close(rl.stopCh)
	})
}

// getLimiter returns the rate limiter for the given IP address.
func (rl *RateLimiter) getLimiter(ip string) *rate.Limiter {
	now := time.Now()

	rl.mu.Lock()
	defer rl.mu.Unlock()

	entry, exists := rl.limiters[ip]
	if !exists {
		entry = &limiterEntry{
			limiter:  rate.NewLimiter(rl.rate, rl.burst),
			lastSeen: now,
		}
		rl.limiters[ip] = entry
	} else {
		entry.lastSeen = now
	}

	return entry.limiter
}

// Middleware returns a middleware that rate limits requests per IP.
func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := r.RemoteAddr
		limiter := rl.getLimiter(ip)

		if !limiter.Allow() {
			http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// cleanupLimiters removes inactive limiters to prevent memory leaks.
// Runs in a goroutine started by NewRateLimiter.
func (rl *RateLimiter) cleanupLimiters() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			rl.mu.Lock()
			now := time.Now()
			// Remove limiters inactive for more than 10 minutes
			for ip, entry := range rl.limiters {
				if now.Sub(entry.lastSeen) > 10*time.Minute {
					delete(rl.limiters, ip)
				}
			}
			rl.mu.Unlock()
		case <-rl.stopCh:
			return
		}
	}
}
