package tracks

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tmeire/tracks/session"
)

func TestSession_Integration(t *testing.T) {
	app := NewTestApp(t, TestConfig{})
	// Manually override config to use DB sessions
	conf := app.Router().Config()
	conf.BaseDomain = "floral-crm.localhost:8080"
	conf.Sessions.Store.Type = "db"
	conf.Sessions.Store.Config = []byte(`{"type": "sqlite", "config": {"path": ":memory:"}}`)
	
	app.router = NewFromConfig(context.Background(), conf)
	router := app.Router()

	router.GetFunc("/login", "test", "login", func(r *http.Request) (any, error) {
		sess := session.FromRequest(r)
		sess.Authenticate("user123")
		return &Response{
			StatusCode: http.StatusNoContent,
			Location:   "/",
		}, nil
	})

	router.GetFunc("/get", "test", "get", func(r *http.Request) (any, error) {
		sess := session.FromRequest(r)
		userID, ok := sess.Authenticated()
		if !ok {
			return "not authenticated", nil
		}
		return userID, nil
	})

	// 1. Login
	w1 := httptest.NewRecorder()
	req1 := httptest.NewRequest("GET", "http://floral-crm.localhost:8080/login", nil)
	req1.Header.Set("Accept", "text/html")
	h, _ := router.Handler()
	h.ServeHTTP(w1, req1)

	assert.Equal(t, http.StatusSeeOther, w1.Code)
	cookie := w1.Header().Get("Set-Cookie")
	assert.NotEmpty(t, cookie)

	// 2. Check if authenticated
	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest("GET", "http://floral-crm.localhost:8080/get", nil)
	req2.Header.Set("Cookie", cookie)
	req2.Header.Set("Accept", "application/json")
	h.ServeHTTP(w2, req2)

	assert.Equal(t, http.StatusOK, w2.Code)
	assert.Contains(t, w2.Body.String(), "user123")
}
