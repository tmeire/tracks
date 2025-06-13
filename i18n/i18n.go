package i18n

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// Translator handles translations for different languages
type Translator struct {
	translations map[string]map[string]interface{} // Nested structure
	flatCache    map[string]map[string]string      // Flattened for quick lookup
	defaultLang  string
	mutex        sync.RWMutex
}

// NewTranslator creates a new translator with the given default language
func NewTranslator(defaultLang string) *Translator {
	return &Translator{
		translations: make(map[string]map[string]interface{}),
		flatCache:    make(map[string]map[string]string),
		defaultLang:  defaultLang,
	}
}

// LoadTranslations loads translations from JSON files in the given directory
func (t *Translator) LoadTranslations(dir string) error {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	files, err := filepath.Glob(filepath.Join(dir, "*.json"))
	if err != nil {
		return err
	}

	for _, file := range files {
		lang := strings.TrimSuffix(filepath.Base(file), ".json")
		data, err := os.ReadFile(file)
		if err != nil {
			return err
		}

		// Parse the JSON into a nested structure
		var translations map[string]interface{}
		if err := json.Unmarshal(data, &translations); err != nil {
			return err
		}

		// Store the nested structure
		t.translations[lang] = translations

		// Create a flattened version for quick lookups
		if t.flatCache[lang] == nil {
			t.flatCache[lang] = make(map[string]string)
		}
		t.flattenTranslations(translations, "", lang)
	}

	return nil
}

// flattenTranslations recursively flattens nested translations into dot notation
// e.g., {"pricing":{"plans":{"freelance":"Freelance"}}} becomes {"pricing.plans.freelance":"Freelance"}
func (t *Translator) flattenTranslations(nested map[string]interface{}, prefix string, lang string) {
	for key, value := range nested {
		// Create the new key with dot notation
		newKey := key
		if prefix != "" {
			newKey = prefix + "." + key
		}

		// If the value is a nested object, recurse
		if nestedValue, isNested := value.(map[string]interface{}); isNested {
			t.flattenTranslations(nestedValue, newKey, lang)
		} else if stringValue, isString := value.(string); isString {
			// If it's a string value, add it to the flat cache
			t.flatCache[lang][newKey] = stringValue
		} else if value != nil {
			// For other types, convert to string
			t.flatCache[lang][newKey] = fmt.Sprintf("%v", value)
		}
	}
}

// getNestedValue retrieves a value from a nested structure using dot notation
func getNestedValue(nested map[string]interface{}, key string) (string, bool) {
	parts := strings.Split(key, ".")
	current := nested

	// Navigate through the nested structure
	for i, part := range parts {
		if i == len(parts)-1 {
			// Last part, get the value
			if value, ok := current[part]; ok {
				if strValue, isString := value.(string); isString {
					return strValue, true
				} else if value != nil {
					return fmt.Sprintf("%v", value), true
				}
				return "", false
			}
			return "", false
		}

		// Not the last part, navigate deeper
		if nextLevel, ok := current[part]; ok {
			if nextMap, isMap := nextLevel.(map[string]interface{}); isMap {
				current = nextMap
			} else {
				return "", false
			}
		} else {
			return "", false
		}
	}

	return "", false
}

// Translate translates a key to the given language
func (t *Translator) Translate(lang, key string) string {
	t.mutex.RLock()
	defer t.mutex.RUnlock()

	// First check the flat cache for quick lookup
	if cache, ok := t.flatCache[lang]; ok {
		if translation, ok := cache[key]; ok {
			return translation
		}
	}

	// Try the nested structure for the requested language
	if translations, ok := t.translations[lang]; ok {
		if translation, ok := getNestedValue(translations, key); ok {
			return translation
		}
	}

	// Fall back to default language
	if lang != t.defaultLang {
		// Check flat cache first
		if cache, ok := t.flatCache[t.defaultLang]; ok {
			if translation, ok := cache[key]; ok {
				return translation
			}
		}

		// Try nested structure
		if translations, ok := t.translations[t.defaultLang]; ok {
			if translation, ok := getNestedValue(translations, key); ok {
				return translation
			}
		}
	}

	// Return the key if no translation is found
	return key
}

// TranslateWithParams translates a key to the given language and applies parameters
func (t *Translator) TranslateWithParams(lang, key string, params ...interface{}) string {
	translation := t.Translate(lang, key)
	if len(params) == 0 {
		return translation
	}

	// If the translation is the same as the key, it means no translation was found
	if translation == key && !strings.Contains(key, "%") {
		return key
	}

	// If there are parameters, format the string
	if len(params) > 0 {
		return fmt.Sprintf(translation, params...)
	}

	return translation
}
