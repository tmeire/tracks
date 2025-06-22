package i18n

import (
	"context"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/tmeire/tracks/session"
)

func TestLanguageFromContext(t *testing.T) {
	// Test with language in context
	ctx := context.Background()
	ctx = WithLanguage(ctx, "fr")

	lang := LanguageFromContext(ctx)
	if lang != "fr" {
		t.Errorf("Expected language 'fr', got '%s'", lang)
	}

	// Test with no language in context
	emptyCtx := context.Background()
	lang = LanguageFromContext(emptyCtx)
	if lang != "" {
		t.Errorf("Expected empty language, got '%s'", lang)
	}
}

func TestWithLanguage(t *testing.T) {
	// Test setting language in context
	ctx := context.Background()
	ctx = WithLanguage(ctx, "es")

	// Verify language was set
	value := ctx.Value(langKey)
	if value == nil {
		t.Fatal("Language not set in context")
	}

	lang, ok := value.(string)
	if !ok {
		t.Fatal("Language value is not a string")
	}

	if lang != "es" {
		t.Errorf("Expected language 'es', got '%s'", lang)
	}

	// Test overriding language
	ctx = WithLanguage(ctx, "de")
	lang = LanguageFromContext(ctx)
	if lang != "de" {
		t.Errorf("Expected language 'de', got '%s'", lang)
	}
}

// Mock session for testing
type mockSession struct {
	data map[string]string
}

func (m *mockSession) Authenticate(userId string)    {}
func (m *mockSession) Authenticated() (string, bool) { return "", false }
func (m *mockSession) IsAuthenticated() bool         { return false }
func (m *mockSession) Get(key string) (string, bool) {
	val, ok := m.data[key]
	return val, ok
}
func (m *mockSession) Put(key string, value string)     { m.data[key] = value }
func (m *mockSession) Forget(key string)                { delete(m.data, key) }
func (m *mockSession) ID() string                       { return "test-session" }
func (m *mockSession) Flash(key string, value string)   {}
func (m *mockSession) FlashMessages() map[string]string { return nil }
func (m *mockSession) Save(ctx context.Context) error   { return nil }
func (m *mockSession) Invalidate(ctx context.Context)   {}

// Helper function for testing DetectLanguage without relying on session.FromRequest
func detectLanguageWithSession(r *http.Request, defaultLang string, sess session.Session) string {
	// 1. Check if language is stored in session
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

func TestDetectLanguage(t *testing.T) {

	// Test with language in session
	t.Run("Language from session", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)

		// Create mock session with language
		sess := &mockSession{
			data: map[string]string{
				"language": "fr",
			},
		}

		lang := detectLanguageWithSession(req, "en", sess)
		if lang != "fr" {
			t.Errorf("Expected language 'fr' from session, got '%s'", lang)
		}
	})

	// Test with language in Accept-Language header
	t.Run("Language from Accept-Language header", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("Accept-Language", "es-ES,es;q=0.9")

		// No session, should use header
		lang := detectLanguageWithSession(req, "en", nil)
		if lang != "es" {
			t.Errorf("Expected language 'es' from header, got '%s'", lang)
		}
	})

	// Test with complex Accept-Language header
	t.Run("Complex Accept-Language header", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("Accept-Language", "de-DE,de;q=0.9,en;q=0.8")

		// No session, should use header
		lang := detectLanguageWithSession(req, "en", nil)
		if lang != "de" {
			t.Errorf("Expected language 'de' from header, got '%s'", lang)
		}
	})

	// Test fallback to default language
	t.Run("Fallback to default", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)

		// No session, no header, should use default
		lang := detectLanguageWithSession(req, "en", nil)
		if lang != "en" {
			t.Errorf("Expected default language 'en', got '%s'", lang)
		}
	})

	// Test the actual DetectLanguage function with Accept-Language header
	t.Run("Actual DetectLanguage function", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("Accept-Language", "it-IT,it;q=0.9")

		lang := DetectLanguage(req, "en")
		if lang != "it" {
			t.Errorf("Expected language 'it' from header, got '%s'", lang)
		}
	})
}

func TestMiddleware(t *testing.T) {
	// Create middleware
	middleware := Middleware("en")

	// Create a test handler that checks if language is in context
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		lang := LanguageFromContext(r.Context())
		w.Header().Set("X-Language", lang)
		w.WriteHeader(http.StatusOK)
	})

	// Wrap the test handler with the middleware
	handler, err := middleware(testHandler)
	assert.NoError(t, err)

	// Test with Accept-Language header
	t.Run("Sets language from header", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("Accept-Language", "fr-FR,fr;q=0.9")

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}

		lang := rr.Header().Get("X-Language")
		if lang != "fr" {
			t.Errorf("Expected language 'fr', got '%s'", lang)
		}
	})

	// Test with no language header (should use default)
	t.Run("Uses default language", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}

		lang := rr.Header().Get("X-Language")
		if lang != "en" {
			t.Errorf("Expected default language 'en', got '%s'", lang)
		}
	})
}
