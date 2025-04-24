package tracks

import (
	"encoding/json"
	"encoding/xml"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestRouter_Get(t *testing.T) {
	// Create a new router
	router := New()

	// Register a simple handler using Action
	router.Get("/test", "default", "test", func(r *http.Request) (interface{}, error) {
		// Return an opaque data object, which will automatically get a StatusOK
		return "Hello, Test!", nil
	})

	// Test with the default Accept header (should default to JSON)
	t.Run("Default Content Type", func(t *testing.T) {
		req, err := http.NewRequest("GET", "/test", nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}

		// Check content type
		contentType := rr.Header().Get("Content-Type")
		if contentType != "application/json" {
			t.Errorf("Handler returned wrong content type: got %v want %v", contentType, "application/json")
		}

		// Check the response body (should be JSON)
		var responseData string
		if err := json.Unmarshal(rr.Body.Bytes(), &responseData); err != nil {
			t.Errorf("Failed to unmarshal JSON response: %v", err)
		}
		expected := "Hello, Test!"
		if responseData != expected {
			t.Errorf("Handler returned unexpected body: got %v want %v", responseData, expected)
		}
	})

	// Test with JSON Accept header
	t.Run("JSON Content Type", func(t *testing.T) {
		req, err := http.NewRequest("GET", "/test", nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		req.Header.Set("Accept", "application/json")

		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}

		// Check content type
		contentType := rr.Header().Get("Content-Type")
		if contentType != "application/json" {
			t.Errorf("Handler returned wrong content type: got %v want %v", contentType, "application/json")
		}

		// Check the response body (should be JSON)
		var responseData string
		if err := json.Unmarshal(rr.Body.Bytes(), &responseData); err != nil {
			t.Errorf("Failed to unmarshal JSON response: %v", err)
		}
		expected := "Hello, Test!"
		if responseData != expected {
			t.Errorf("Handler returned unexpected body: got %v want %v", responseData, expected)
		}
	})

	// Test with XML Accept header
	t.Run("XML Content Type", func(t *testing.T) {
		req, err := http.NewRequest("GET", "/test", nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		req.Header.Set("Accept", "application/xml")

		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}

		// Check content type
		contentType := rr.Header().Get("Content-Type")
		if contentType != "application/xml" {
			t.Errorf("Handler returned wrong content type: got %v want %v", contentType, "application/xml")
		}

		// Check the response body (should be XML)
		var responseData string
		if err := xml.Unmarshal(rr.Body.Bytes(), &responseData); err != nil {
			t.Errorf("Failed to unmarshal XML response: %v", err)
		}
		expected := "Hello, Test!"
		if responseData != expected {
			t.Errorf("Handler returned unexpected body: got %v want %v", responseData, expected)
		}
	})

	// Test with HTML Accept header
	t.Run("HTML Content Type", func(t *testing.T) {
		// Create a test template file
		if err := os.MkdirAll("views/test", 0755); err != nil {
			t.Fatalf("Failed to create views directory: %v", err)
		}

		templateContent := `<!DOCTYPE html>
<html>
<head>
    <title>Test</title>
</head>
<body>
    <h1>{{.}}</h1>
</body>
</html>`

		if err := os.WriteFile("views/test/index.gohtml", []byte(templateContent), 0644); err != nil {
			t.Fatalf("Failed to create template file: %v", err)
		}

		// Clean up after the test
		defer func() {
			os.RemoveAll("views")
		}()

		req, err := http.NewRequest("GET", "/test", nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		req.Header.Set("Accept", "text/html")

		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}

		// Check content type
		contentType := rr.Header().Get("Content-Type")
		if contentType != "text/html" {
			t.Errorf("Handler returned wrong content type: got %v want %v", contentType, "text/html")
		}

		// Check the response body (should contain HTML)
		responseBody := rr.Body.String()
		if !strings.Contains(responseBody, "<h1>Hello, Test!</h1>") {
			t.Errorf("Handler returned unexpected body: %v", responseBody)
		}
	})
}

func TestRouter_Get_WithError(t *testing.T) {
	// Create a new router
	router := New()

	// Register a handler that returns an error
	router.Get("/error", "default", "error", func(r *http.Request) (interface{}, error) {
		// Return a Response object with error data
		return &Response{
			StatusCode: http.StatusBadRequest,
			Data: map[string]interface{}{
				"message": "Bad Request",
				"code":    "INVALID_PARAMETER",
				"details": map[string]string{"field": "id", "issue": "missing"},
			},
		}, nil
	})

	// Test with JSON Accept header
	t.Run("JSON Error Response", func(t *testing.T) {
		req, err := http.NewRequest("GET", "/error", nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		req.Header.Set("Accept", "application/json")

		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusBadRequest {
			t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusBadRequest)
		}

		// Check content type
		contentType := rr.Header().Get("Content-Type")
		if contentType != "application/json" {
			t.Errorf("Handler returned wrong content type: got %v want %v", contentType, "application/json")
		}

		// Check the response body
		var errorResp map[string]interface{}
		if err := json.Unmarshal(rr.Body.Bytes(), &errorResp); err != nil {
			t.Errorf("Failed to unmarshal JSON response: %v", err)
		}

		if message, ok := errorResp["message"].(string); !ok || message != "Bad Request" {
			t.Errorf("Handler returned unexpected error message: got %v want %v", errorResp["message"], "Bad Request")
		}

		if code, ok := errorResp["code"].(string); !ok || code != "INVALID_PARAMETER" {
			t.Errorf("Handler returned unexpected error code: got %v want %v", errorResp["code"], "INVALID_PARAMETER")
		}
	})
}
