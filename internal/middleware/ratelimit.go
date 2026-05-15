package middleware

import (
	"net/http"
	"sync"
	"time"
)

// RateLimitMiddleware — 간단한 IP 기반 Rate Limiting (프로덕션에서는 Redis 기반으로 전환)
// Step 1에서는 인메모리로 MVP 구현, 이후 Step 9에서 Redis 기반으로 교체
func RateLimitMiddleware(next http.Handler) http.Handler {
	limiter := newIPRateLimiter(100, time.Minute) // 분당 100 요청

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := r.RemoteAddr
		if !limiter.Allow(ip) {
			w.Header().Set("Retry-After", "60")
			http.Error(w, `{"error":"너무 많은 요청입니다. 잠시 후 다시 시도해주세요."}`, http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}

type ipRateLimiter struct {
	mu       sync.Mutex
	requests map[string][]time.Time
	limit    int
	window   time.Duration
}

func newIPRateLimiter(limit int, window time.Duration) *ipRateLimiter {
	return &ipRateLimiter{
		requests: make(map[string][]time.Time),
		limit:    limit,
		window:   window,
	}
}

func (l *ipRateLimiter) Allow(ip string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-l.window)

	// 오래된 요청 제거
	var recent []time.Time
	for _, t := range l.requests[ip] {
		if t.After(cutoff) {
			recent = append(recent, t)
		}
	}

	if len(recent) >= l.limit {
		l.requests[ip] = recent
		return false
	}

	l.requests[ip] = append(recent, now)
	return true
}
