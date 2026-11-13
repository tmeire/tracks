package tracks

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func csrfCookie(t *testing.T, cfg CSRFConfig, mutate func(*http.Request)) *http.Cookie {
	t.Helper()
	h, err := CSRFProtection(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	assert.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "http://example.com/", nil)
	if mutate != nil {
		mutate(req)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	for _, c := range rec.Result().Cookies() {
		if c.Name == defaultCSRFCookieName {
			return c
		}
	}
	t.Fatal("no csrf cookie set")
	return nil
}

func TestCSRFCookieFlags(t *testing.T) {
	t.Run("plain http is not secure", func(t *testing.T) {
		c := csrfCookie(t, CSRFConfig{}, nil)
		assert.False(t, c.Secure)
		assert.True(t, c.HttpOnly)
		assert.Equal(t, http.SameSiteLaxMode, c.SameSite)
	})
	t.Run("behind a TLS terminating proxy", func(t *testing.T) {
		c := csrfCookie(t, CSRFConfig{}, func(r *http.Request) { r.Header.Set("X-Forwarded-Proto", "https") })
		assert.True(t, c.Secure)
	})
	t.Run("configured secure", func(t *testing.T) {
		c := csrfCookie(t, CSRFConfig{Secure: true}, nil)
		assert.True(t, c.Secure)
	})
}
