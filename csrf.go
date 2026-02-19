package tracks

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"html/template"
	"net/http"
	"strings"
)

type CSRFConfig struct {
	TokenLength  int
	CookieName   string
	HeaderName   string
	FieldName    string
	ExemptPaths  []string
}

const (
	defaultCSRFTokenLength = 32
	defaultCSRFCookieName  = "csrf_token"
	defaultCSRFHeaderName  = "X-CSRF-Token"
	defaultCSRFFieldName   = "csrf_token"
)

type csrfKey struct{}

// CSRFTokenFromContext returns the CSRF token stored in the context.
func CSRFTokenFromContext(r *http.Request) string {
	if token, ok := r.Context().Value(csrfKey{}).(string); ok {
		return token
	}
	return ""
}

func generateToken(length int) (string, error) {
	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(b), nil
}

func CSRFProtection(config CSRFConfig) Middleware {
	if config.TokenLength == 0 {
		config.TokenLength = defaultCSRFTokenLength
	}
	if config.CookieName == "" {
		config.CookieName = defaultCSRFCookieName
	}
	if config.HeaderName == "" {
		config.HeaderName = defaultCSRFHeaderName
	}
	if config.FieldName == "" {
		config.FieldName = defaultCSRFFieldName
	}

	return func(next http.Handler) (http.Handler, error) {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip for safe methods
			if r.Method == "GET" || r.Method == "HEAD" || r.Method == "OPTIONS" || r.Method == "TRACE" {
				// But we still need to provide a token for the form
				token := getOrCreateToken(w, r, config)
				r = r.WithContext(context.WithValue(r.Context(), csrfKey{}, token))
				next.ServeHTTP(w, r)
				return
			}

			// Check if path is exempt
			for _, path := range config.ExemptPaths {
				if matchPath(path, r.URL.Path) {
					next.ServeHTTP(w, r)
					return
				}
			}

			// Validate token
			cookie, err := r.Cookie(config.CookieName)
			if err != nil {
				http.Error(w, "CSRF token missing", http.StatusForbidden)
				return
			}

			requestToken := r.Header.Get(config.HeaderName)
			if requestToken == "" {
				requestToken = r.FormValue(config.FieldName)
			}

			if requestToken == "" || requestToken != cookie.Value {
				http.Error(w, "CSRF token mismatch", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		}), nil
	}
}

func getOrCreateToken(w http.ResponseWriter, r *http.Request, config CSRFConfig) string {
	cookie, err := r.Cookie(config.CookieName)
	if err == nil {
		return cookie.Value
	}

	token, _ := generateToken(config.TokenLength)
	http.SetCookie(w, &http.Cookie{
		Name:     config.CookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   r.TLS != nil,
		SameSite: http.SameSiteLaxMode,
	})
	return token
}

func matchPath(pattern, path string) bool {
	if strings.HasSuffix(pattern, "*") {
		return strings.HasPrefix(path, strings.TrimSuffix(pattern, "*"))
	}
	return pattern == path
}

// CSRFField returns a hidden input field containing the CSRF token.
func CSRFField(r *http.Request) template.HTML {
	token := CSRFTokenFromContext(r)
	return template.HTML(fmt.Sprintf(`<input type="hidden" name="%s" value="%s">`, defaultCSRFFieldName, token))
}
