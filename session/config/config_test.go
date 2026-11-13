package config

import (
	"testing"
	"time"

	"github.com/tmeire/tracks/session"
)

func TestLifetime(t *testing.T) {
	l, err := Config{}.Lifetime()
	if err != nil || l.Resolved().Absolute != session.DefaultAbsoluteTimeout {
		t.Fatalf("defaults: %v %v", l, err)
	}
	l, err = Config{AbsoluteTimeout: "1h", IdleTimeout: "0"}.Lifetime()
	if err != nil {
		t.Fatal(err)
	}
	r := l.Resolved()
	if r.Absolute != time.Hour || r.Idle != 0 {
		t.Fatalf("unexpected %v", r)
	}
	if _, err := (Config{IdleTimeout: "bogus"}).Lifetime(); err == nil {
		t.Fatal("expected error")
	}
}
