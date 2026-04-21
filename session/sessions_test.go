package session

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"github.com/stretchr/testify/assert"
)

type mockStore struct{}
func (m *mockStore) Load(ctx context.Context, id string) (Session, bool) { return nil, false }
func (m *mockStore) Create(ctx context.Context) Session { return &mockSession{id: "test"} }

type mockSession struct { id string }
func (s *mockSession) Authenticate(userId string) {}
func (s *mockSession) Authenticated() (string, bool) { return "", false }
func (s *mockSession) IsAuthenticated() bool { return false }
func (s *mockSession) Get(key string) (string, bool) { return "", false }
func (s *mockSession) Put(key string, value string) {}
func (s *mockSession) Forget(key string) {}
func (s *mockSession) ID() string { return s.id }
func (s *mockSession) Flash(key string, value string) {}
func (s *mockSession) FlashMessages() map[string]string { return nil }
func (s *mockSession) Save(ctx context.Context) error { return nil }
func (s *mockSession) Invalidate(ctx context.Context) {}

func TestMiddleware_CookieDomain(t *testing.T) {
	store := &mockStore{}
	
	t.Run("Domain without port", func(t *testing.T) {
		mw := Middleware("floralynx.com", store)
		handler, _ := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
		
		req := httptest.NewRequest("GET", "http://floralynx.com/", nil)
		w := httptest.NewRecorder()
		
		handler.ServeHTTP(w, req)
		
		header := w.Header().Get("Set-Cookie")
		assert.Contains(t, header, "Domain=floralynx.com")
	})

	t.Run("Domain with port", func(t *testing.T) {
		mw := Middleware("localhost:8080", store)
		handler, _ := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
		
		req := httptest.NewRequest("GET", "http://localhost:8080/", nil)
		w := httptest.NewRecorder()
		
		handler.ServeHTTP(w, req)
		
		header := w.Header().Get("Set-Cookie")
		assert.Contains(t, header, "Domain=localhost")
	})
}
