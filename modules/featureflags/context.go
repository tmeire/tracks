package featureflags

import (
	"context"
)

type ctxKey string

const flagsCtxKey ctxKey = "featureflags.flags"

// withFlags stores a computed flags map into the context.
func withFlags(ctx context.Context, values map[string]bool) context.Context {
	return context.WithValue(ctx, flagsCtxKey, values)
}

// FromContext returns the flags map if present.
func FromContext(ctx context.Context) map[string]bool {
	if ctx == nil {
		return nil
	}
	if v, ok := ctx.Value(flagsCtxKey).(map[string]bool); ok {
		return v
	}
	return nil
}

// Enabled returns true if a flag is enabled for the given context.
// In dev/test: panic on unknown keys. In prod: return false for unknown keys.
func Enabled(ctx context.Context, key string) bool {
	// Check registration
	if _, ok := getDefault(key); !ok {
		if isDevOrTest() {
			panic("featureflags: unknown key: " + key)
		}
		return false
	}
	m := FromContext(ctx)
	if m == nil {
		// Not computed yet: fall back to default
		def, _ := getDefault(key)
		return def
	}
	if v, ok := m[key]; ok {
		return v
	}
	def, _ := getDefault(key)
	return def
}

// Get returns the value and whether it was present in the computed map.
// Presence false means fallback to default might be applied by callers.
// In dev/test: panic on unknown keys. In prod: return false,false.
func Get(ctx context.Context, key string) (bool, bool) {
	if _, ok := getDefault(key); !ok {
		if isDevOrTest() {
			panic("featureflags: unknown key: " + key)
		}
		return false, false
	}
	m := FromContext(ctx)
	if m == nil {
		return false, false
	}
	v, ok := m[key]
	return v, ok
}
