# Feature Request: General Purpose Caching Layer

**Priority:** Medium  
**Status:** Open

## Description

While tracks has caching for feature flags, it lacks a general-purpose caching layer for application data, query results, and computed values.

## Current Implementation

Only feature flags have caching (in-memory LRU cache):

```go
// modules/featureflags/cache.go
type lruCache struct {
    mu    sync.Mutex
    items map[cacheKey]cacheEntry
    order []cacheKey
    ttl   time.Duration
    cap   int
}
```

No general caching interface exists for:
- Database query results
- Template fragments
- External API responses
- Computed/expensive operations

## Required Functionality

1. **Cache Interface**: Generic cache interface with multiple backends
2. **In-Memory Backend**: Built-in LRU cache (like feature flags)
3. **Redis Backend**: Redis support for distributed caching
4. **Cache Keys**: Automatic key generation with namespaces
5. **TTL Support**: Time-based expiration
6. **Cache Tags**: Tag-based invalidation
7. **Cache Middleware**: Automatic caching of responses

## Proposed API

```go
// Configuration
config := tracks.Config{
    Cache: cache.Config{
        Driver: "redis", // or "memory"
        Redis: cache.RedisConfig{
            Addr: "localhost:6379",
        },
        Memory: cache.MemoryConfig{
            Size: 1000,
            TTL: 5 * time.Minute,
        },
    },
}

// Using cache directly
func (h *Handler) GetData(r *http.Request) (any, error) {
    cache := cache.FromContext(r.Context())
    
    // Try cache first
    if data, ok := cache.Get("expensive_data"); ok {
        return data, nil
    }
    
    // Compute expensive data
    data := computeExpensiveData()
    
    // Store in cache
    cache.Set("expensive_data", data, 10*time.Minute)
    
    return data, nil
}

// Cache tags for bulk invalidation
cache.SetWithTags("user_123", user, []string{"users", "user_123"}, 1*time.Hour)
cache.InvalidateTag("users") // Clear all user entries

// Repository-level caching
repo := database.NewRepository[User](schema)
repo.WithCache(5 * time.Minute) // Cache all queries for 5 minutes

// Middleware for response caching
router.GetFunc("/api/slow-endpoint", "api", "slow", slowHandler,
    tracks.CacheMiddleware(1*time.Hour),
)
```

## Use Cases

- Database query result caching
- API response caching
- Template fragment caching
- Session data caching
- Rate limit counters
- Computed data caching

## Acceptance Criteria

- [ ] Generic cache interface
- [ ] In-memory LRU implementation
- [ ] Redis backend implementation
- [ ] Context-based cache access
- [ ] TTL support
- [ ] Cache tags for bulk invalidation
- [ ] Middleware for HTTP response caching
- [ ] Repository integration for query caching
- [ ] Cache statistics and monitoring
- [ ] Documentation and examples

## Cache Interface

```go
type Cache interface {
    Get(key string) (value any, ok bool)
    Set(key string, value any, ttl time.Duration)
    SetWithTags(key string, value any, tags []string, ttl time.Duration)
    Delete(key string)
    InvalidateTag(tag string)
    InvalidateAll()
    Stats() Stats
}
```
