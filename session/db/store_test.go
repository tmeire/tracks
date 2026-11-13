package db

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/tmeire/tracks/database/sqlite"
	"github.com/tmeire/tracks/session"
)

func newTestStore(t *testing.T) (*Store, context.Context) {
	t.Helper()
	sqlDB, err := sqlite.New(filepath.Join(t.TempDir(), "sessions.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { sqlDB.Close() })
	ctx := context.Background()
	s, err := NewStore(ctx, sqlDB)
	if err != nil {
		t.Fatal(err)
	}
	return s, ctx
}

func age(t *testing.T, s *Store, id string, created, updated time.Time) {
	t.Helper()
	if _, err := s.database.ExecContext(context.Background(), "UPDATE sessions SET created_at = ?, updated_at = ? WHERE id = ?", created, updated, id); err != nil {
		t.Fatal(err)
	}
}

func count(t *testing.T, s *Store) int {
	t.Helper()
	var n int
	if err := s.database.QueryRowContext(context.Background(), "SELECT COUNT(*) FROM sessions").Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func TestLoadEnforcesExpiry(t *testing.T) {
	s, ctx := newTestStore(t)
	s.SetLifetime(session.Lifetime{Absolute: time.Hour, Idle: 10 * time.Minute})
	now := time.Now()

	fresh := s.Create(ctx)
	if _, ok := s.Load(ctx, fresh.ID()); !ok {
		t.Fatal("fresh session should load")
	}

	idle := s.Create(ctx)
	age(t, s, idle.ID(), now.Add(-20*time.Minute), now.Add(-11*time.Minute))
	if _, ok := s.Load(ctx, idle.ID()); ok {
		t.Fatal("idle session should be expired")
	}

	old := s.Create(ctx)
	age(t, s, old.ID(), now.Add(-2*time.Hour), now)
	if _, ok := s.Load(ctx, old.ID()); ok {
		t.Fatal("session older than the absolute lifetime should be expired")
	}

	// expired rows are deleted on load; only the fresh one is left
	if n := count(t, s); n != 1 {
		t.Fatalf("expected 1 row left, got %d", n)
	}
}

func TestDeleteExpired(t *testing.T) {
	s, ctx := newTestStore(t)
	s.SetLifetime(session.Lifetime{Absolute: time.Hour, Idle: 10 * time.Minute})
	now := time.Now()
	keep := s.Create(ctx)
	gone := s.Create(ctx)
	age(t, s, gone.ID(), now.Add(-2*time.Hour), now.Add(-2*time.Hour))
	if err := s.DeleteExpired(ctx); err != nil {
		t.Fatal(err)
	}
	if n := count(t, s); n != 1 {
		t.Fatalf("expected 1 row, got %d", n)
	}
	if _, ok := s.Load(ctx, keep.ID()); !ok {
		t.Fatal("kept session should still load")
	}
}

func TestAuthenticateRotatesAndSaveReplacesRow(t *testing.T) {
	s, ctx := newTestStore(t)
	sess := s.Create(ctx)
	old := sess.ID()
	sess.Put("k", "v")

	sess.Authenticate("u1")
	if sess.ID() == old {
		t.Fatal("session id must change on login")
	}
	if err := sess.Save(ctx); err != nil {
		t.Fatal(err)
	}

	if _, ok := s.Load(ctx, old); ok {
		t.Fatal("old session id must be dead after login")
	}
	loaded, ok := s.Load(ctx, sess.ID())
	if !ok || !loaded.IsAuthenticated() {
		t.Fatal("new session id should load as authenticated")
	}
	if v, _ := loaded.Get("k"); v != "v" {
		t.Fatal("data should survive rotation")
	}
	if n := count(t, s); n != 1 {
		t.Fatalf("expected exactly 1 row after rotation, got %d", n)
	}

	// A further save updates in place
	if err := loaded.Save(ctx); err != nil {
		t.Fatal(err)
	}
	if n := count(t, s); n != 1 {
		t.Fatalf("expected 1 row, got %d", n)
	}
}

func TestInvalidateDeletesServerSideSession(t *testing.T) {
	s, ctx := newTestStore(t)
	sess := s.Create(ctx)
	sess.Authenticate("u1")
	if err := sess.Save(ctx); err != nil {
		t.Fatal(err)
	}
	old := sess.ID()
	sess.Invalidate(ctx)
	if _, ok := s.Load(ctx, old); ok {
		t.Fatal("invalidated session must be gone")
	}
	if sess.ID() == old || sess.IsAuthenticated() {
		t.Fatal("invalidated session should be a fresh anonymous one")
	}
}

func TestInvalidateAfterPendingRotationDeletesOriginalRow(t *testing.T) {
	s, ctx := newTestStore(t)
	sess := s.Create(ctx)
	orig := sess.ID()
	sess.Authenticate("u1") // rotation pending, original row still in DB
	sess.Invalidate(ctx)
	if _, ok := s.Load(ctx, orig); ok {
		t.Fatal("original row must be deleted")
	}
}
