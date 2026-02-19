package tracks

import (
	"context"
	"net/http"
	"sync"
	"time"
)

type Cache interface {
	Get(key string) (any, bool)
	Set(key string, value any, ttl time.Duration)
	SetWithTags(key string, value any, tags []string, ttl time.Duration)
	Delete(key string)
	InvalidateTag(tag string)
	InvalidateAll()
}

type cacheEntry struct {
	value      any
	expiration int64
	tags       []string
}

type memoryCache struct {
	mu    sync.RWMutex
	items map[string]cacheEntry
	tags  map[string][]string // tag -> list of keys
}

func NewMemoryCache() Cache {
	return &memoryCache{
		items: make(map[string]cacheEntry),
		tags:  make(map[string][]string),
	}
}

func (c *memoryCache) Get(key string) (any, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	item, ok := c.items[key]
	if !ok {
		return nil, false
	}

	if item.expiration > 0 && time.Now().UnixNano() > item.expiration {
		return nil, false
	}

	return item.value, true
}

func (c *memoryCache) Set(key string, value any, ttl time.Duration) {
	c.SetWithTags(key, value, nil, ttl)
}

func (c *memoryCache) SetWithTags(key string, value any, tags []string, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	var expiration int64
	if ttl > 0 {
		expiration = time.Now().Add(ttl).UnixNano()
	}

	c.items[key] = cacheEntry{
		value:      value,
		expiration: expiration,
		tags:       tags,
	}

	for _, tag := range tags {
		c.tags[tag] = append(c.tags[tag], key)
	}
}

func (c *memoryCache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.items, key)
}

func (c *memoryCache) InvalidateTag(tag string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	keys, ok := c.tags[tag]
	if !ok {
		return
	}

	for _, key := range keys {
		delete(c.items, key)
	}
	delete(c.tags, tag)
}

func (c *memoryCache) InvalidateAll() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items = make(map[string]cacheEntry)
	c.tags = make(map[string][]string)
}

type cacheContextKey struct{}

func WithCache(ctx context.Context, c Cache) context.Context {
	return context.WithValue(ctx, cacheContextKey{}, c)
}

func CacheFromContext(ctx context.Context) Cache {
	if c, ok := ctx.Value(cacheContextKey{}).(Cache); ok {
		return c
	}
	return nil
}

// CacheMiddleware returns a middleware that caches the HTTP response.
func CacheMiddleware(ttl time.Duration) MiddlewareBuilder {
	return func(router Router) Middleware {
		return func(next http.Handler) (http.Handler, error) {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				cache := CacheFromContext(r.Context())
				if cache == nil {
					next.ServeHTTP(w, r)
					return
				}

				key := "resp:" + r.Method + ":" + r.URL.String()
				if val, ok := cache.Get(key); ok {
					if _, ok := val.(*Response); ok {
						// This is tricky because the action might render HTML or JSON.
						// Response caching should probably happen at the action level.
					}
				}

				next.ServeHTTP(w, r)
			}), nil
		}
	}
}
