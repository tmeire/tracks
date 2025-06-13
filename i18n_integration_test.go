package tracks

import (
	"html/template"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tmeire/tracks/database/sqlite"
	"github.com/tmeire/tracks/i18n"
)

func TestTranslationFunction(t *testing.T) {
	// Create a temporary database for testing
	tempDB, err := sqlite.New(":memory:")
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer tempDB.Close()

	// Create temporary translation files
	tempDir, err := os.MkdirTemp("", "test-translations")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test translation files
	enContent := `{
		"welcome": "Welcome",
		"greeting": "Hello, %s!",
		"items_count": "You have %d items"
	}`
	frContent := `{
		"welcome": "Bienvenue",
		"greeting": "Bonjour, %s !",
		"items_count": "Vous avez %d articles"
	}`

	if err := os.WriteFile(filepath.Join(tempDir, "en.json"), []byte(enContent), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	if err := os.WriteFile(filepath.Join(tempDir, "fr.json"), []byte(frContent), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Create a translator with the test translations
	translator := i18n.NewTranslator("en")
	err = translator.LoadTranslations(tempDir)
	if err != nil {
		t.Fatalf("Failed to load translations: %v", err)
	}

	// Create a template with the translation function
	tmpl := template.New("test")
	
	// Add the translation function to the template
	tmpl = tmpl.Funcs(template.FuncMap{
		"t": func(key string, args ...interface{}) string {
			// This simulates the router's translation function
			// The last argument is expected to be the template data
			if len(args) == 0 {
				return key // No context available, return the key
			}

			// Try to extract the request from the template data
			var req *http.Request
			lastArg := args[len(args)-1]

			// Check if the last argument is a struct with a Request field
			if data, ok := lastArg.(struct{ Request *http.Request }); ok {
				req = data.Request
			}

			if req == nil {
				return key // No request available, return the key
			}

			// Get language from context
			lang := i18n.LanguageFromContext(req.Context())
			if lang == "" {
				lang = "en" // Default language
			}

			// If there are additional arguments, use them as parameters for the translation
			if len(args) > 1 {
				return translator.TranslateWithParams(lang, key, args[:len(args)-1]...)
			}

			// Translate the key
			return translator.Translate(lang, key)
		},
	})

	// Test with English language
	t.Run("English translation", func(t *testing.T) {
		// Create a request with English language
		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("Accept-Language", "en-US,en;q=0.9")
		
		// Add language to context
		ctx := i18n.WithLanguage(req.Context(), "en")
		req = req.WithContext(ctx)

		// Parse a template with translation
		tmpl, err := tmpl.Parse(`{{ t "welcome" . }}`)
		if err != nil {
			t.Fatalf("Failed to parse template: %v", err)
		}

		// Execute the template
		var buf strings.Builder
		err = tmpl.Execute(&buf, struct{ Request *http.Request }{Request: req})
		if err != nil {
			t.Fatalf("Failed to execute template: %v", err)
		}

		// Check the result
		if buf.String() != "Welcome" {
			t.Errorf("Expected 'Welcome', got '%s'", buf.String())
		}
	})

	// Test with French language
	t.Run("French translation", func(t *testing.T) {
		// Create a request with French language
		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("Accept-Language", "fr-FR,fr;q=0.9")
		
		// Add language to context
		ctx := i18n.WithLanguage(req.Context(), "fr")
		req = req.WithContext(ctx)

		// Parse a template with translation
		tmpl, err := tmpl.Parse(`{{ t "welcome" . }}`)
		if err != nil {
			t.Fatalf("Failed to parse template: %v", err)
		}

		// Execute the template
		var buf strings.Builder
		err = tmpl.Execute(&buf, struct{ Request *http.Request }{Request: req})
		if err != nil {
			t.Fatalf("Failed to execute template: %v", err)
		}

		// Check the result
		if buf.String() != "Bienvenue" {
			t.Errorf("Expected 'Bienvenue', got '%s'", buf.String())
		}
	})

	// Test with parameters
	t.Run("Translation with parameters", func(t *testing.T) {
		// Create a request with English language
		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("Accept-Language", "en-US,en;q=0.9")
		
		// Add language to context
		ctx := i18n.WithLanguage(req.Context(), "en")
		req = req.WithContext(ctx)

		// Parse a template with translation and parameters
		tmpl, err := tmpl.Parse(`{{ t "greeting" "John" . }}`)
		if err != nil {
			t.Fatalf("Failed to parse template: %v", err)
		}

		// Execute the template
		var buf strings.Builder
		err = tmpl.Execute(&buf, struct{ Request *http.Request }{Request: req})
		if err != nil {
			t.Fatalf("Failed to execute template: %v", err)
		}

		// Check the result
		if buf.String() != "Hello, John!" {
			t.Errorf("Expected 'Hello, John!', got '%s'", buf.String())
		}
	})

	// Test with missing key
	t.Run("Missing translation key", func(t *testing.T) {
		// Create a request with English language
		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("Accept-Language", "en-US,en;q=0.9")
		
		// Add language to context
		ctx := i18n.WithLanguage(req.Context(), "en")
		req = req.WithContext(ctx)

		// Parse a template with a missing translation key
		tmpl, err := tmpl.Parse(`{{ t "missing_key" . }}`)
		if err != nil {
			t.Fatalf("Failed to parse template: %v", err)
		}

		// Execute the template
		var buf strings.Builder
		err = tmpl.Execute(&buf, struct{ Request *http.Request }{Request: req})
		if err != nil {
			t.Fatalf("Failed to execute template: %v", err)
		}

		// Check the result - should return the key itself
		if buf.String() != "missing_key" {
			t.Errorf("Expected 'missing_key', got '%s'", buf.String())
		}
	})

	// Test with fallback to default language
	t.Run("Fallback to default language", func(t *testing.T) {
		// Create a request with a language that doesn't have translations
		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("Accept-Language", "es-ES,es;q=0.9")
		
		// Add language to context
		ctx := i18n.WithLanguage(req.Context(), "es")
		req = req.WithContext(ctx)

		// Parse a template with translation
		tmpl, err := tmpl.Parse(`{{ t "welcome" . }}`)
		if err != nil {
			t.Fatalf("Failed to parse template: %v", err)
		}

		// Execute the template
		var buf strings.Builder
		err = tmpl.Execute(&buf, struct{ Request *http.Request }{Request: req})
		if err != nil {
			t.Fatalf("Failed to execute template: %v", err)
		}

		// Check the result - should fall back to English
		if buf.String() != "Welcome" {
			t.Errorf("Expected fallback to 'Welcome', got '%s'", buf.String())
		}
	})

	// Test with no request in context
	t.Run("No request in context", func(t *testing.T) {
		// Parse a template with translation
		tmpl, err := tmpl.Parse(`{{ t "welcome" }}`)
		if err != nil {
			t.Fatalf("Failed to parse template: %v", err)
		}

		// Execute the template without a request
		var buf strings.Builder
		err = tmpl.Execute(&buf, nil)
		if err != nil {
			t.Fatalf("Failed to execute template: %v", err)
		}

		// Check the result - should return the key itself
		if buf.String() != "welcome" {
			t.Errorf("Expected 'welcome', got '%s'", buf.String())
		}
	})
}