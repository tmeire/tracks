package i18n

import (
	"context"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
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

func TestDetectLanguage(t *testing.T) {
	// Test with query parameter
	t.Run("Language from query parameter", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/?locale=fr", nil)
		lang := DetectLanguage(req, "en")
		if lang != "fr" {
			t.Errorf("Expected language 'fr' from query param, got '%s'", lang)
		}
	})

	// Test with cookie
	t.Run("Language from cookie", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		req.AddCookie(&http.Cookie{Name: "locale", Value: "de"})
		lang := DetectLanguage(req, "en")
		if lang != "de" {
			t.Errorf("Expected language 'de' from cookie, got '%s'", lang)
		}
	})

	// Test with language in Accept-Language header
	t.Run("Language from Accept-Language header", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("Accept-Language", "es-ES,es;q=0.9")

		lang := DetectLanguage(req, "en")
		if lang != "es" {
			t.Errorf("Expected language 'es' from header, got '%s'", lang)
		}
	})

	// Test fallback to default language
	t.Run("Fallback to default", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)

		lang := DetectLanguage(req, "en")
		if lang != "en" {
			t.Errorf("Expected default language 'en', got '%s'", lang)
		}
	})
}

func TestT(t *testing.T) {
	translator := NewTranslator("en")
	translator.flatCache = map[string]map[string]string{
		"en": {"hello": "Hello"},
		"fr": {"hello": "Bonjour"},
	}

	t.Run("T with language in context", func(t *testing.T) {
		ctx := context.Background()
		ctx = WithTranslator(ctx, translator)
		ctx = WithLanguage(ctx, "fr")

		result := T(ctx, "hello")
		if result != "Bonjour" {
			t.Errorf("Expected 'Bonjour', got '%s'", result)
		}
	})

	t.Run("T without translator in context", func(t *testing.T) {
		ctx := context.Background()
		ctx = WithLanguage(ctx, "fr")

		result := T(ctx, "hello")
		if result != "hello" {
			t.Errorf("Expected 'hello' (key), got '%s'", result)
		}
	})
}

func TestSetLocale(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	
	cookie := SetLocale(req, "fr")
	
	if cookie.Name != "locale" {
		t.Errorf("Expected cookie name 'locale', got '%s'", cookie.Name)
	}
	if cookie.Value != "fr" {
		t.Errorf("Expected cookie value 'fr', got '%s'", cookie.Value)
	}
}

func TestMiddleware(t *testing.T) {
	translator := NewTranslator("en")
	// Create middleware
	middleware := Middleware(translator, "en")

	// Create a test handler that checks if language is in context
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		lang := LanguageFromContext(r.Context())
		tr := TranslatorFromContext(r.Context())
		
		if tr != translator {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		
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
