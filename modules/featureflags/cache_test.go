package featureflags

import (
	"testing"
	"time"
)

func TestLRUGetSetAndTTL(t *testing.T) {
	c := newLRU(100*time.Millisecond, 2)
	k := cacheKey{user: "u", tenant: "t", roles: "r", ver: 1}
	v := map[string]bool{"a": true}
	c.set(k, v, time.Unix(0, 0))
	if got, ok := c.get(k, time.Unix(0, int64(50*time.Millisecond))); !ok || !got["a"] {
		t.Fatalf("expected cache hit before TTL expiry")
	}
	if _, ok := c.get(k, time.Unix(0, int64(200*time.Millisecond))); ok {
		t.Fatalf("expected cache miss after TTL expiry")
	}
}

func TestLRUEvictionOrder(t *testing.T) {
	c := newLRU(time.Minute, 2)
	k1 := cacheKey{user: "1"}
	k2 := cacheKey{user: "2"}
	k3 := cacheKey{user: "3"}
	c.set(k1, map[string]bool{"a": true}, time.Now())
	c.set(k2, map[string]bool{"b": true}, time.Now())
	// add third triggers eviction of oldest (k1)
	c.set(k3, map[string]bool{"c": true}, time.Now())
	if _, ok := c.get(k1, time.Now()); ok {
		t.Fatalf("expected k1 to be evicted")
	}
	if _, ok := c.get(k2, time.Now()); !ok {
		t.Fatalf("expected k2 present")
	}
	if _, ok := c.get(k3, time.Now()); !ok {
		t.Fatalf("expected k3 present")
	}
}

func TestMakeCacheKeyIncludesPrincipalsAndVersion(t *testing.T) {
	resetRegistry()
	RegisterFlags(Flag{Key: "x", Default: false})
	p1 := Principals{UserID: strPtr("u1"), TenantID: strPtr("t1"), RoleIDs: []string{"r2", "r1"}}
	p2 := Principals{UserID: strPtr("u1"), TenantID: strPtr("t1"), RoleIDs: []string{"r1", "r2"}}
	k1 := makeCacheKey(p1)
	k2 := makeCacheKey(p2)
	if k1 != k2 {
		t.Fatalf("expected roles order to be normalized: %v vs %v", k1, k2)
	}
	v1 := k1.ver
	RegisterFlags(Flag{Key: "y", Default: true})
	k3 := makeCacheKey(p1)
	if k3.ver == v1 {
		t.Fatalf("expected registry version to change in cache key")
	}
}

func strPtr(s string) *string { return &s }
