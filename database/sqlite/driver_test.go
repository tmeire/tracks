package sqlite

import (
	"context"
	"path/filepath"
	"testing"
)

func pragmaOnTwoConns(t *testing.T, opts Options, pragma string) []int {
	t.Helper()
	db, err := NewWithOptions(filepath.Join(t.TempDir(), "t.sqlite"), opts)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	db.SetMaxOpenConns(3)

	ctx := context.Background()
	// Hold several connections at the same time so the pool has to open more than the first one.
	var out []int
	for i := 0; i < 3; i++ {
		c, err := db.Conn(ctx)
		if err != nil {
			t.Fatal(err)
		}
		defer c.Close()
		var v int
		if err := c.QueryRowContext(ctx, "PRAGMA "+pragma).Scan(&v); err != nil {
			t.Fatal(err)
		}
		out = append(out, v)
	}
	return out
}

func TestBusyTimeoutAppliedToEveryPooledConnection(t *testing.T) {
	for _, v := range pragmaOnTwoConns(t, Options{}, "busy_timeout") {
		if v != DefaultBusyTimeoutMs {
			t.Fatalf("busy_timeout = %d, want %d", v, DefaultBusyTimeoutMs)
		}
	}
	for _, v := range pragmaOnTwoConns(t, Options{BusyTimeoutMs: 1234}, "busy_timeout") {
		if v != 1234 {
			t.Fatalf("busy_timeout = %d, want 1234", v)
		}
	}
}

func TestForeignKeysOptIn(t *testing.T) {
	for _, v := range pragmaOnTwoConns(t, Options{}, "foreign_keys") {
		if v != 0 {
			t.Fatalf("foreign_keys = %d, want 0 by default", v)
		}
	}
	for _, v := range pragmaOnTwoConns(t, Options{ForeignKeys: true}, "foreign_keys") {
		if v != 1 {
			t.Fatalf("foreign_keys = %d, want 1 when enabled", v)
		}
	}
}

func TestMemoryAndFileURIPaths(t *testing.T) {
	db, err := New(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	db.Close()

	if got := dsn("file:x.db?cache=shared", Options{}); got != "file:x.db?cache=shared&_busy_timeout=5000" {
		t.Fatalf("unexpected dsn %q", got)
	}
}
