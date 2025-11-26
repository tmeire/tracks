package featureflags

import (
	"os"
	"sync"
)

// Flag represents a boolean feature flag definition registered in code.
type Flag struct {
	Key         string
	Description string
	Default     bool
}

var (
	regMu      sync.RWMutex
	registered = map[string]Flag{}
	regVer     uint64
)

// env helpers
func isDevOrTest() bool {
	env := os.Getenv("GO_ENV")
	switch env {
	case "development", "dev", "test", "testing":
		return true
	default:
		return false
	}
}

// RegisterFlags registers one or more flags. In dev/test it panics on duplicate keys.
func RegisterFlags(flags ...Flag) {
	regMu.Lock()
	defer regMu.Unlock()
	for _, f := range flags {
		if f.Key == "" {
			if isDevOrTest() {
				panic("featureflags: empty key in RegisterFlags")
			}
			// ignore in prod
			continue
		}
		if _, exists := registered[f.Key]; exists && isDevOrTest() {
			panic("featureflags: duplicate registration for key: " + f.Key)
		}
		// If new or updating description/default, bump version as it may affect effective defaults
		if prev, ok := registered[f.Key]; !ok || prev.Description != f.Description || prev.Default != f.Default {
			regVer++
		}
		registered[f.Key] = f
	}
}

// listKeys returns the keys of all registered flags (copy).
func listKeys() []string {
	regMu.RLock()
	defer regMu.RUnlock()
	keys := make([]string, 0, len(registered))
	for k := range registered {
		keys = append(keys, k)
	}
	return keys
}

// getDefault returns the default value for a registered key and whether it exists.
func getDefault(key string) (bool, bool) {
	regMu.RLock()
	defer regMu.RUnlock()
	f, ok := registered[key]
	if !ok {
		return false, false
	}
	return f.Default, true
}

// allRegistered returns a snapshot of the registry map.
func allRegistered() map[string]Flag {
	regMu.RLock()
	defer regMu.RUnlock()
	out := make(map[string]Flag, len(registered))
	for k, v := range registered {
		out[k] = v
	}
	return out
}

func registryVersion() uint64 {
	regMu.RLock()
	defer regMu.RUnlock()
	return regVer
}
