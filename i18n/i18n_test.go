package i18n

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewTranslator(t *testing.T) {
	translator := NewTranslator("en")

	if translator == nil {
		t.Fatal("NewTranslator returned nil")
	}

	if translator.defaultLang != "en" {
		t.Errorf("Expected default language to be 'en', got '%s'", translator.defaultLang)
	}

	if translator.translations == nil {
		t.Error("Translations map should be initialized")
	}

	if translator.flatCache == nil {
		t.Error("Flat cache map should be initialized")
	}
}

func TestLoadTranslations(t *testing.T) {
	// Create a temporary directory for test translations
	tempDir, err := os.MkdirTemp("", "translations")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test translation files with flat structure
	enContent := `{
		"test.key": "Test value",
		"greeting": "Hello"
	}`
	frContent := `{
		"test.key": "Valeur de test",
		"greeting": "Bonjour"
	}`

	if err := os.WriteFile(filepath.Join(tempDir, "en.json"), []byte(enContent), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	if err := os.WriteFile(filepath.Join(tempDir, "fr.json"), []byte(frContent), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Test loading translations
	translator := NewTranslator("en")
	err = translator.LoadTranslations(tempDir)
	if err != nil {
		t.Fatalf("LoadTranslations failed: %v", err)
	}

	// Verify translations were loaded
	if len(translator.translations) != 2 {
		t.Errorf("Expected 2 languages, got %d", len(translator.translations))
	}

	// Check English translations using Translate method
	if result := translator.Translate("en", "test.key"); result != "Test value" {
		t.Errorf("Expected 'Test value', got '%s'", result)
	}

	if result := translator.Translate("en", "greeting"); result != "Hello" {
		t.Errorf("Expected 'Hello', got '%s'", result)
	}

	// Check French translations using Translate method
	if result := translator.Translate("fr", "test.key"); result != "Valeur de test" {
		t.Errorf("Expected 'Valeur de test', got '%s'", result)
	}

	if result := translator.Translate("fr", "greeting"); result != "Bonjour" {
		t.Errorf("Expected 'Bonjour', got '%s'", result)
	}
}

func TestLoadNestedTranslations(t *testing.T) {
	// Create a temporary directory for test translations
	tempDir, err := os.MkdirTemp("", "nested-translations")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test translation files with nested structure
	enContent := `{
		"pricing": {
			"plans": {
				"freelance": "Freelance",
				"basic": "Basic",
				"pro": "Professional"
			}
		},
		"user": {
			"greeting": "Hello, %s!",
			"profile": {
				"title": "User Profile"
			}
		},
		"flat_key": "Flat value"
	}`

	if err := os.WriteFile(filepath.Join(tempDir, "en.json"), []byte(enContent), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Test loading translations
	translator := NewTranslator("en")
	err = translator.LoadTranslations(tempDir)
	if err != nil {
		t.Fatalf("LoadTranslations failed: %v", err)
	}

	// Test accessing nested keys with dot notation
	testCases := []struct {
		key      string
		expected string
	}{
		{"pricing.plans.freelance", "Freelance"},
		{"pricing.plans.basic", "Basic"},
		{"pricing.plans.pro", "Professional"},
		{"user.greeting", "Hello, %s!"},
		{"user.profile.title", "User Profile"},
		{"flat_key", "Flat value"},
		{"nonexistent.key", "nonexistent.key"}, // Missing key should return the key itself
	}

	for _, tc := range testCases {
		result := translator.Translate("en", tc.key)
		if result != tc.expected {
			t.Errorf("For key '%s', expected '%s', got '%s'", tc.key, tc.expected, result)
		}
	}

	// Test with parameters
	result := translator.TranslateWithParams("en", "user.greeting", "John")
	expected := "Hello, John!"
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

func TestTranslate(t *testing.T) {
	// Create a translator with test translations
	translator := NewTranslator("en")

	// Set up flat cache for testing
	translator.flatCache = map[string]map[string]string{
		"en": {
			"test.key": "Test value",
			"greeting": "Hello",
		},
		"fr": {
			"test.key": "Valeur de test",
			"greeting": "Bonjour",
		},
	}

	// Test translating to English
	if result := translator.Translate("en", "test.key"); result != "Test value" {
		t.Errorf("Expected 'Test value', got '%s'", result)
	}

	// Test translating to French
	if result := translator.Translate("fr", "test.key"); result != "Valeur de test" {
		t.Errorf("Expected 'Valeur de test', got '%s'", result)
	}

	// Test fallback to default language
	if result := translator.Translate("es", "test.key"); result != "Test value" {
		t.Errorf("Expected fallback to 'Test value', got '%s'", result)
	}

	// Test missing key returns the key itself
	if result := translator.Translate("en", "missing.key"); result != "missing.key" {
		t.Errorf("Expected key itself for missing key, got '%s'", result)
	}
}

func TestTranslateWithParams(t *testing.T) {
	// Create a translator with test translations
	translator := NewTranslator("en")

	// Set up flat cache for testing
	translator.flatCache = map[string]map[string]string{
		"en": {
			"welcome": "Welcome, %s!",
			"count":   "You have %d items",
		},
		"fr": {
			"welcome": "Bienvenue, %s !",
			"count":   "Vous avez %d articles",
		},
	}

	// Test with string parameter
	if result := translator.TranslateWithParams("en", "welcome", "John"); result != "Welcome, John!" {
		t.Errorf("Expected 'Welcome, John!', got '%s'", result)
	}

	// Test with integer parameter
	if result := translator.TranslateWithParams("en", "count", 5); result != "You have 5 items" {
		t.Errorf("Expected 'You have 5 items', got '%s'", result)
	}

	// Test with French translation
	if result := translator.TranslateWithParams("fr", "welcome", "Marie"); result != "Bienvenue, Marie !" {
		t.Errorf("Expected 'Bienvenue, Marie !', got '%s'", result)
	}

	// Test fallback to default language
	if result := translator.TranslateWithParams("es", "welcome", "Carlos"); result != "Welcome, Carlos!" {
		t.Errorf("Expected fallback to 'Welcome, Carlos!', got '%s'", result)
	}

	// Test missing key returns the key itself with formatting
	if result := translator.TranslateWithParams("en", "missing.key", "test"); result != "missing.key" {
		t.Errorf("Expected key itself for missing key, got '%s'", result)
	}
}

func TestGetNestedValue(t *testing.T) {
	// Test nested structure
	nested := map[string]interface{}{
		"pricing": map[string]interface{}{
			"plans": map[string]interface{}{
				"freelance": "Freelance",
				"basic":     "Basic",
				"pro":       "Professional",
			},
		},
		"user": map[string]interface{}{
			"greeting": "Hello, %s!",
			"profile": map[string]interface{}{
				"title": "User Profile",
			},
		},
		"flat_key": "Flat value",
	}

	testCases := []struct {
		key      string
		expected string
		found    bool
	}{
		{"pricing.plans.freelance", "Freelance", true},
		{"pricing.plans.basic", "Basic", true},
		{"pricing.plans.pro", "Professional", true},
		{"user.greeting", "Hello, %s!", true},
		{"user.profile.title", "User Profile", true},
		{"flat_key", "Flat value", true},
		{"nonexistent.key", "", false},
		{"pricing.nonexistent", "", false},
		{"pricing.plans.nonexistent", "", false},
	}

	for _, tc := range testCases {
		result, found := getNestedValue(nested, tc.key)
		if found != tc.found {
			t.Errorf("For key '%s', expected found=%v, got found=%v", tc.key, tc.found, found)
		}

		if found && result != tc.expected {
			t.Errorf("For key '%s', expected '%s', got '%s'", tc.key, tc.expected, result)
		}
	}
}
