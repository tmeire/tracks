package featureflags

import (
	"sort"
	"strings"
	"sync"
	"time"
)

type cacheKey struct {
	user   string
	tenant string
	roles  string // sorted, joined
	ver    uint64 // registry version
}

type cacheEntry struct {
	at   time.Time
	data map[string]bool
}

type lruCache struct {
	mu    sync.Mutex
	items map[cacheKey]cacheEntry
	order []cacheKey
	ttl   time.Duration
	cap   int
}

func newLRU(ttl time.Duration, cap int) *lruCache {
	return &lruCache{items: make(map[cacheKey]cacheEntry), ttl: ttl, cap: cap}
}

func (c *lruCache) get(k cacheKey, now time.Time) (map[string]bool, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if e, ok := c.items[k]; ok {
		if now.Sub(e.at) <= c.ttl {
			// move to end
			c.touch(k)
			return e.data, true
		}
		// expired
		delete(c.items, k)
		c.removeFromOrder(k)
	}
	return nil, false
}

func (c *lruCache) set(k cacheKey, v map[string]bool, now time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, ok := c.items[k]; !ok {
		c.order = append(c.order, k)
	} else {
		c.touch(k)
	}
	c.items[k] = cacheEntry{at: now, data: v}
	for len(c.items) > c.cap {
		// evict oldest
		oldest := c.order[0]
		c.order = c.order[1:]
		delete(c.items, oldest)
	}
}

func (c *lruCache) touch(k cacheKey) {
	// move key to end
	c.removeFromOrder(k)
	c.order = append(c.order, k)
}

func (c *lruCache) removeFromOrder(k cacheKey) {
	for i, kk := range c.order {
		if kk == k {
			c.order = append(c.order[:i], c.order[i+1:]...)
			return
		}
	}
}

var globalCache = newLRU(60*time.Second, 1024)

func makeCacheKey(p Principals) cacheKey {
	var user, tenant string
	if p.UserID != nil {
		user = *p.UserID
	}
	if p.TenantID != nil {
		tenant = *p.TenantID
	}
	roles := append([]string(nil), p.RoleIDs...)
	sort.Strings(roles)
	return cacheKey{user: user, tenant: tenant, roles: strings.Join(roles, ","), ver: registryVersion()}
}
