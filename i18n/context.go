package i18n

import (
	"context"
	"net/http"
	"strings"

	"github.com/tmeire/tracks/session"
)

type contextKey string

const (
	langKey       contextKey = "language"
	translatorKey contextKey = "translator"
)

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

// WithTranslator adds a translator to the context
func WithTranslator(ctx context.Context, t *Translator) context.Context {
	return context.WithValue(ctx, translatorKey, t)
}

// TranslatorFromContext gets the translator from the context
func TranslatorFromContext(ctx context.Context) *Translator {
	if t, ok := ctx.Value(translatorKey).(*Translator); ok {
		return t
	}
	return nil
}

// T translates a key using the translator and language found in the context
func T(ctx context.Context, key string, params ...interface{}) string {
	t := TranslatorFromContext(ctx)
	if t == nil {
		return key
	}
	lang := LanguageFromContext(ctx)
	return t.TranslateWithParams(lang, key, params...)
}

// SetLocale sets the locale in the session and returns a cookie that can be used to set the locale.
func SetLocale(r *http.Request, locale string) *http.Cookie {
	sess := session.FromRequest(r)
	if sess != nil {
		sess.Put("language", locale)
	}
	return &http.Cookie{
		Name:     "locale",
		Value:    locale,
		Path:     "/",
		HttpOnly: true,
		MaxAge:   31536000, // 1 year
	}
}

// DetectLanguage detects the preferred language from the request
func DetectLanguage(r *http.Request, defaultLang string) string {
	// 1. Query Parameter: ?locale=fr
	if locale := r.URL.Query().Get("locale"); locale != "" {
		return locale
	}

	// 2. Cookie: locale=fr
	if cookie, err := r.Cookie("locale"); err == nil && cookie.Value != "" {
		return cookie.Value
	}

	// 3. Check if the language is stored in session
	sess := session.FromRequest(r)
	if sess != nil {
		if lang, ok := sess.Get("language"); ok && lang != "" {
			return lang
		}
	}

	// 4. Check Accept-Language header
	acceptLang := r.Header.Get("Accept-Language")
	if acceptLang != "" {
		langs := strings.Split(acceptLang, ",")
		if len(langs) > 0 {
			// Extract language code from something like "en-US,en;q=0.9"
			langCode := strings.Split(langs[0], ";")[0]
			langCode = strings.TrimSpace(langCode)
			langCode = strings.Split(langCode, "-")[0] // Get just "en" from "en-US"
			return langCode
		}
	}

	// 5. Return default language
	return defaultLang
}

// Middleware adds language detection to the request context
func Middleware(t *Translator, defaultLang string) func(next http.Handler) (http.Handler, error) {
	return func(next http.Handler) (http.Handler, error) {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			lang := DetectLanguage(r, defaultLang)
			ctx := WithLanguage(r.Context(), lang)
			ctx = WithTranslator(ctx, t)
			r = r.WithContext(ctx)
			next.ServeHTTP(w, r)
		}), nil
	}
}
