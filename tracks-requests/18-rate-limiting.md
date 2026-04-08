# Feature Request: Rate Limiting Middleware

**Priority:** Medium  
**Status:** Open

## Description

The framework lacks built-in rate limiting, which is essential for protecting APIs from abuse and ensuring fair usage.

## Current Gap

No rate limiting for:
- API endpoints
- Form submissions (waitlist, registration)
- Authentication endpoints
- General request throttling

## Required Functionality

1. **Request-Based Limiting**: Limit by IP, user, or API key
2. **Multiple Algorithms**: Token bucket, fixed window, sliding window
3. **Storage Backends**: Memory, Redis, database
4. **Configurable Limits**: Requests per time window
5. **Response Headers**: X-RateLimit-* headers
6. **Custom Keys**: Define rate limit keys (IP, user ID, etc.)
7. **Whitelist/Blacklist**: Exempt or block specific IPs/users
8. **Response Handling**: Custom responses when limit exceeded

## Proposed API

```go
// Global rate limiting
router.RateLimit(tracks.RateLimitConfig{
    Requests: 100,
    Window:   time.Minute,
    Storage:  "redis", // or "memory", "database"
})

// Per-route rate limiting
router.PostFunc("/api/waitlist", "waitlist", "create", waitlistHandler,
    tracks.RateLimitMiddleware(tracks.RateLimitConfig{
        Requests: 5,
        Window:   time.Hour,
        KeyFunc: func(r *http.Request) string {
            // Limit by email + IP combination
            email := r.FormValue("email")
            ip := r.RemoteAddr
            return email + ":" + ip
        },
    }),
)

// Authentication endpoint - stricter limits
router.PostFunc("/login", "auth", "login", loginHandler,
    tracks.RateLimitMiddleware(tracks.RateLimitConfig{
        Requests: 5,
        Window:   15 * time.Minute,
        KeyFunc: func(r *http.Request) string {
            return r.RemoteAddr // Limit by IP
        },
        OnLimitExceeded: func(w http.ResponseWriter, r *http.Request) {
            tracks.RespondJSON(w, http.StatusTooManyRequests, map[string]any{
                "error": "Too many login attempts. Please try again later.",
                "retry_after": 900, // seconds
            })
        },
    }),
)

// API key based limiting
router.RateLimit(tracks.RateLimitConfig{
    Requests: 1000,
    Window:   time.Hour,
    KeyFunc: func(r *http.Request) string {
        apiKey := r.Header.Get("X-API-Key")
        if apiKey == "" {
            return r.RemoteAddr // Fallback to IP
        }
        return apiKey
    },
})

// Different limits for different tiers
router.Group("/api", func(api tracks.Router) {
    // Free tier: 100/hour
    api.RateLimit(tracks.RateLimitConfig{
        Requests: 100,
        Window:   time.Hour,
        KeyFunc:  tierBasedKeyFunc("free"),
    })
    
    // Paid tier: 10000/hour
    api.RateLimit(tracks.RateLimitConfig{
        Requests: 10000,
        Window:   time.Hour,
        KeyFunc:  tierBasedKeyFunc("paid"),
    })
})
```

## Response Headers

```
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 95
X-RateLimit-Reset: 1640995200
X-RateLimit-Window: 3600
```

**When limit exceeded:**
```
HTTP 429 Too Many Requests
Retry-After: 300

{
  "error": "Rate limit exceeded",
  "retry_after": 300,
  "limit": 100,
  "window": "1h"
}
```

## Use Cases

- API endpoint protection
- Form spam prevention
- Brute force protection (login)
- Fair usage enforcement
- DDoS mitigation
- Tiered API plans (free vs paid)

## Acceptance Criteria

- [ ] Token bucket algorithm
- [ ] Fixed window algorithm
- [ ] Sliding window algorithm
- [ ] Multiple storage backends (memory, Redis, DB)
- [ ] Configurable rate limits
- [ ] Custom key functions
- [ ] Rate limit headers in responses
- [ ] Custom exceeded responses
- [ ] Whitelist/blacklist support
- [ ] Per-route and global middleware
- [ ] Documentation and examples

## Algorithms

**Token Bucket:**
- Smooth rate limiting
- Allows bursts up to bucket size
- Good for APIs

**Fixed Window:**
- Simple counter reset
- Potential burst at window boundary
- Good for simple use cases

**Sliding Window:**
- Smooth limiting like token bucket
- More accurate than fixed window
- Higher memory overhead
