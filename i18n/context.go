package i18n

import (
	"context"
	"net/http"
	"strings"

	"github.com/tmeire/tracks/session"
)

type contextKey string

const langKey contextKey = "language"

// LanguageFromContext gets the language from the context
func LanguageFromContext(ctx context.Context) string {
	if lang, ok := ctx.Value(langKey).(string); ok {
		return lang
	}
	return ""
}

// WithLanguage adds a language to the context
func WithLanguage(ctx context.Context, lang string) context.Context {
	return context.WithValue(ctx, langKey, lang)
}

// DetectLanguage detects the preferred language from the request
func DetectLanguage(r *http.Request, defaultLang string) string {
	// 1. Check if language is stored in session
	sess := session.FromRequest(r)
	if sess != nil {
		if lang, ok := sess.Get("language"); ok && lang != "" {
			return lang
		}
	}

	// 2. Check Accept-Language header
	acceptLang := r.Header.Get("Accept-Language")
	if acceptLang != "" {
		langs := strings.Split(acceptLang, ",")
		if len(langs) > 0 {
			// Extract language code from something like "en-US,en;q=0.9"
			langCode := strings.Split(langs[0], ";")[0]
			langCode = strings.Split(langCode, "-")[0] // Get just "en" from "en-US"
			return langCode
		}
	}

	// 3. Return default language
	return defaultLang
}

// Middleware adds language detection to the request context
func Middleware(defaultLang string) func(next http.Handler) (http.Handler, error) {
	return func(next http.Handler) (http.Handler, error) {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			lang := DetectLanguage(r, defaultLang)
			ctx := WithLanguage(r.Context(), lang)
			r = r.WithContext(ctx)
			next.ServeHTTP(w, r)
		}), nil
	}
}
