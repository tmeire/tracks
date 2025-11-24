package featureflags

import (
    "context"
    "os"
    "testing"
)

func TestEnabledAndGetWithContextAndDefaults(t *testing.T) {
    resetRegistry()
    RegisterFlags(Flag{Key: "k1", Default: true})
    // no context map → fall back to default
    if !Enabled(context.Background(), "k1") {
        t.Fatalf("expected default true for k1")
    }
    // with context map present
    m := map[string]bool{"k1": false}
    ctx := withFlags(context.Background(), m)
    if Enabled(ctx, "k1") {
        t.Fatalf("expected overridden false for k1")
    }
    if v, ok := Get(ctx, "k1"); !ok || v != false {
        t.Fatalf("expected Get to return false,true; got %v,%v", v, ok)
    }
}

func TestUnknownKeyPanicsInDevTest(t *testing.T) {
    resetRegistry()
    os.Setenv("GO_ENV", "test")
    defer os.Unsetenv("GO_ENV")

    defer func() {
        if r := recover(); r == nil {
            t.Fatalf("expected panic for unknown key in dev/test")
        }
    }()
    _ = Enabled(context.Background(), "does.not.exist")
}

func TestUnknownKeyFalseInProd(t *testing.T) {
    resetRegistry()
    os.Setenv("GO_ENV", "prod")
    defer os.Unsetenv("GO_ENV")

    if Enabled(context.Background(), "nope") {
        t.Fatalf("expected false for unknown key in prod")
    }
    if v, ok := Get(context.Background(), "nope"); v || ok {
        t.Fatalf("expected false,false for unknown key in prod, got %v,%v", v, ok)
    }
}
