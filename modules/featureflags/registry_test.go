package featureflags

import (
    "os"
    "testing"
)

// helper to reset global registry between tests
func resetRegistry() {
    regMu.Lock()
    defer regMu.Unlock()
    registered = map[string]Flag{}
    regVer = 0
}

func withEnv(key, val string, fn func()) {
    old, had := os.LookupEnv(key)
    _ = os.Setenv(key, val)
    defer func() {
        if had {
            _ = os.Setenv(key, old)
        } else {
            _ = os.Unsetenv(key)
        }
    }()
    fn()
}

func TestRegisterFlagsStoresAndIncrementsVersion(t *testing.T) {
    resetRegistry()
    // In prod, duplicate registrations do not panic; we can verify version bumps
    withEnv("GO_ENV", "prod", func() {
        RegisterFlags(Flag{Key: "a", Description: "A", Default: false})
        if _, ok := getDefault("a"); !ok {
            t.Fatalf("flag not registered")
        }
        v1 := registryVersion()
        if v1 == 0 {
            t.Fatalf("expected non-zero version after register")
        }
        RegisterFlags(Flag{Key: "b", Description: "B", Default: true})
        v2 := registryVersion()
        if v2 <= v1 {
            t.Fatalf("expected version to increase: %d -> %d", v1, v2)
        }
        // change description triggers version bump
        RegisterFlags(Flag{Key: "b", Description: "B2", Default: true})
        v3 := registryVersion()
        if v3 <= v2 {
            t.Fatalf("expected version to increase on change: %d -> %d", v2, v3)
        }
    })
}

func TestRegisterFlagsDuplicatePanicsInDevTest(t *testing.T) {
    resetRegistry()
    withEnv("GO_ENV", "test", func() {
        RegisterFlags(Flag{Key: "dup", Description: "A", Default: false})
        defer func() {
            if r := recover(); r == nil {
                t.Fatalf("expected panic on duplicate registration in test env")
            }
        }()
        RegisterFlags(Flag{Key: "dup", Description: "A2", Default: true})
    })
}
