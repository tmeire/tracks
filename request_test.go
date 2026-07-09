package tracks

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRequestContextInjection(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	// Set custom context variables
	req = SetVar(req, "ThemeClass", "theme-kids")
	req = SetVar(req, "DefaultStyle", "kids")

	// Retrieve custom context variables
	theme := GetVar(req, "ThemeClass")
	if theme != "theme-kids" {
		t.Errorf("Expected 'theme-kids', got %v", theme)
	}

	style := GetVar(req, "DefaultStyle")
	if style != "kids" {
		t.Errorf("Expected 'kids', got %v", style)
	}

	// Verify non-existent key returns nil
	nonExistent := GetVar(req, "InvalidKey")
	if nonExistent != nil {
		t.Errorf("Expected nil, got %v", nonExistent)
	}
}
