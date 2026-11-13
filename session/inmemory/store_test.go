package inmemory

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/tmeire/tracks/session"
)

func TestExpiry(t *testing.T) {
	ctx := context.Background()
	s := NewStore()
	s.SetLifetime(session.Lifetime{Absolute: time.Hour, Idle: 10 * time.Minute})

	sess := s.Create(ctx).(*sessionData)
	if _, ok := s.Load(ctx, sess.ID()); !ok {
		t.Fatal("fresh session should load")
	}

	sess.updatedAt = time.Now().Add(-11 * time.Minute)
	if _, ok := s.Load(ctx, sess.ID()); ok {
		t.Fatal("idle session should be expired")
	}

	sess2 := s.Create(ctx).(*sessionData)
	sess2.createdAt = time.Now().Add(-2 * time.Hour)
	if _, ok := s.Load(ctx, sess2.ID()); ok {
		t.Fatal("old session should be expired")
	}
}

func TestAuthenticateRotatesID(t *testing.T) {
	ctx := context.Background()
	s := NewStore()
	sess := s.Create(ctx)
	old := sess.ID()
	sess.Put("k", "v")

	sess.Authenticate("u1")
	if sess.ID() == old {
		t.Fatal("session id must change on login")
	}
	if _, ok := s.Load(ctx, old); ok {
		t.Fatal("old session id must no longer load")
	}
	loaded, ok := s.Load(ctx, sess.ID())
	if !ok || !loaded.IsAuthenticated() {
		t.Fatal("new id should load the authenticated session")
	}
	if v, _ := loaded.Get("k"); v != "v" {
		t.Fatal("session data should be kept across rotation")
	}

	id := sess.ID()
	sess.Authenticate("u1")
	if sess.ID() != id {
		t.Fatal("re-authenticating the same user should not rotate again")
	}
}

func TestInvalidateRemovesServerSide(t *testing.T) {
	ctx := context.Background()
	s := NewStore()
	sess := s.Create(ctx)
	sess.Authenticate("u1")
	old := sess.ID()
	sess.Invalidate(ctx)
	if _, ok := s.Load(ctx, old); ok {
		t.Fatal("invalidated session must not load")
	}
	if sess.IsAuthenticated() {
		t.Fatal("invalidated session must be anonymous")
	}
}

func TestMiddlewareSetsNewCookieOnLogin(t *testing.T) {
	store := NewStore()
	mw, _ := session.Middleware("example.com", store)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/login" {
			session.FromRequest(r).Authenticate("u1")
		}
		w.WriteHeader(http.StatusOK)
	}))

	last := func(rec *httptest.ResponseRecorder) string {
		cs := rec.Result().Cookies()
		return cs[len(cs)-1].Value
	}

	rec := httptest.NewRecorder()
	mw.ServeHTTP(rec, httptest.NewRequest("GET", "/", nil))
	pre := last(rec)

	req := httptest.NewRequest("GET", "/login", nil)
	req.AddCookie(&http.Cookie{Name: "sessions", Value: pre})
	rec = httptest.NewRecorder()
	mw.ServeHTTP(rec, req)
	post := last(rec)
	if post == pre {
		t.Fatal("cookie value must change on login")
	}

	// the pre-login (possibly attacker-fixated) cookie no longer yields the authenticated session
	if _, ok := store.Load(context.Background(), pre); ok {
		t.Fatal("pre-login id must be dead")
	}
	if s, ok := store.Load(context.Background(), post); !ok || !s.IsAuthenticated() {
		t.Fatal("post-login id must be the authenticated session")
	}
}
