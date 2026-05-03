package middleware

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

type limiterEntry struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// RateLimiter is a simple per-IP, in-memory rate limiter.
type RateLimiter struct {
	limiters   map[string]*limiterEntry
	mu         sync.RWMutex
	rate       rate.Limit
	burst      int
	trustProxy bool
	stopCh     chan struct{}
	once       sync.Once
}

func NewRateLimiter(r rate.Limit, b int, trustProxy bool) *RateLimiter {
	rl := &RateLimiter{
		limiters:   make(map[string]*limiterEntry),
		rate:       r,
		burst:      b,
		trustProxy: trustProxy,
		stopCh:     make(chan struct{}),
	}
	go rl.cleanupLimiters()
	return rl
}

func (rl *RateLimiter) Stop() {
	rl.once.Do(func() {
		close(rl.stopCh)
	})
}

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

func extractRealIP(r *http.Request, trustProxy bool) string {
	var ip string
	if trustProxy {
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			if idx := strings.Index(xff, ","); idx != -1 {
				ip = strings.TrimSpace(xff[:idx])
			} else {
				ip = strings.TrimSpace(xff)
			}
			if net.ParseIP(ip) != nil {
				return ip
			}
			ip = ""
		}
		if xri := r.Header.Get("X-Real-IP"); xri != "" {
			if net.ParseIP(xri) != nil {
				return xri
			}
		}
	}
	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		ip = host
	} else {
		ip = r.RemoteAddr
	}
	if net.ParseIP(ip) != nil {
		return ip
	}
	return ""
}

func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := extractRealIP(r, rl.trustProxy)
		limiter := rl.getLimiter(ip)
		if !limiter.Allow() {
			http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (rl *RateLimiter) cleanupLimiters() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			rl.mu.Lock()
			now := time.Now()
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
