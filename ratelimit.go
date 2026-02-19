package tracks

import (
	"fmt"
	"net/http"
	"sync"
	"time"
)

type RateLimitConfig struct {
	Requests int
	Window   time.Duration
	KeyFunc  func(r *http.Request) string
}

type rateLimiter struct {
	config RateLimitConfig
	mu     sync.Mutex
	data   map[string][]time.Time
}

func NewRateLimiter(config RateLimitConfig) *rateLimiter {
	if config.KeyFunc == nil {
		config.KeyFunc = func(r *http.Request) string {
			return r.RemoteAddr
		}
	}
	return &rateLimiter{
		config: config,
		data:   make(map[string][]time.Time),
	}
}

func (l *rateLimiter) Allow(key string) (bool, int, time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	windowStart := now.Add(-l.config.Window)

	// Clean up old entries
	hits := l.data[key]
	var newHits []time.Time
	for _, t := range hits {
		if t.After(windowStart) {
			newHits = append(newHits, t)
		}
	}

	if len(newHits) < l.config.Requests {
		newHits = append(newHits, now)
		l.data[key] = newHits
		remaining := l.config.Requests - len(newHits)
		return true, remaining, 0
	}

	l.data[key] = newHits
	retryAfter := newHits[0].Add(l.config.Window).Sub(now)
	return false, 0, retryAfter
}

func RateLimitMiddleware(config RateLimitConfig) MiddlewareBuilder {
	limiter := NewRateLimiter(config)
	
	return func(router Router) Middleware {
		return func(next http.Handler) (http.Handler, error) {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				key := limiter.config.KeyFunc(r)
				allowed, remaining, retryAfter := limiter.Allow(key)

				w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", limiter.config.Requests))
				w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", remaining))

				if !allowed {
					w.Header().Set("Retry-After", fmt.Sprintf("%d", int(retryAfter.Seconds())))
					http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
					return
				}

				next.ServeHTTP(w, r)
			}), nil
		}
	}
}
