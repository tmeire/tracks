package tracks

import (
	"encoding/json"
	"encoding/xml"
	"github.com/stretchr/testify/assert"
	"github.com/tmeire/tracks/database/sqlite"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestRouter_Get(t *testing.T) {
	// Create a temporary database for testing
	tempDB, err := sqlite.New(":memory:")
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer tempDB.Close()

	err = writeTemplate("views/default/test.gohtml", `{{.}}`)
	if err != nil {
		t.Fatalf("Failed to create template file: %v", err)
	}

	err = writeTemplate("views/layouts/application.gohtml", `
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <title>{{ .Title }}</title>
</head>
<body>
    {{ template "yield" .Content }}
</body>
</html>
`)
	if err != nil {
		t.Fatalf("Failed to create template file: %v", err)
	}

	// Clean up after the test
	defer func() {
		os.RemoveAll("views")
	}()

	// Create a new router and register a simple handler using Action
	h, err := New(t.Context(), tempDB).
		GetFunc("/test", "default", "test", func(r *http.Request) (any, error) {
			// Return an opaque data object, which will automatically get a StatusOK
			return "Hello, Test!", nil
		}).
		Handler()

	if !assert.NoError(t, err, "Failed to create router") {
		t.FailNow()
	}
	// Test with the default Accept header (should default to JSON)
	t.Run("Default Content Type", func(t *testing.T) {
		req, err := http.NewRequest("GET", "/test", nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}

		// Check content type
		contentType := rr.Header().Get("Content-Type")
		if contentType != "text/html" {
			t.Errorf("Handler returned wrong content type: got %v want %v", contentType, "text/html")
		}

		// Check the response body (should be JSON)
		responseBody := rr.Body.String()
		expected := "\n<!DOCTYPE html>\n<html lang=\"en\">\n<head>\n    <meta charset=\"UTF-8\">\n    <title></title>\n</head>\n<body>\n    Hello, Test!\n</body>\n</html>\n"
		if responseBody != expected {
			t.Errorf("Handler returned unexpected body: got %q want %q", responseBody, expected)
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
		h.ServeHTTP(rr, req)

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
		h.ServeHTTP(rr, req)

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
			t.Errorf("Handler returned unexpected body: got %q want %v", responseData, expected)
		}
	})

	// Test with HTML Accept header
	t.Run("HTML Content Type", func(t *testing.T) {
		req, err := http.NewRequest("GET", "/test", nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		req.Header.Set("Accept", "text/html")

		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)

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
		if !strings.Contains(responseBody, "\n<!DOCTYPE html>\n<html lang=\"en\">\n<head>\n    <meta charset=\"UTF-8\">\n    <title></title>\n</head>\n<body>\n    Hello, Test!\n</body>\n</html>\n") {
			t.Errorf("Handler returned unexpected body: %v", responseBody)
		}
	})
}

func TestRouter_Get_WithError(t *testing.T) {
	// Create a temporary database for testing
	tempDB, err := sqlite.New(":memory:")
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer tempDB.Close()

	// Create a new router and register a handler that returns an error
	h, err := New(t.Context(), tempDB).
		GetFunc("/error", "default", "error", func(r *http.Request) (any, error) {
			// Return a Response object with error data
			return &Response{
				StatusCode: http.StatusBadRequest,
				Data: map[string]any{
					"message": "Bad Request",
					"code":    "INVALID_PARAMETER",
					"details": map[string]string{"field": "id", "issue": "missing"},
				},
			}, nil
		}).
		Handler()

	if !assert.NoError(t, err, "Failed to create router") {
		t.FailNow()
	}

	// Test with JSON Accept header
	t.Run("JSON Error Response", func(t *testing.T) {
		req, err := http.NewRequest("GET", "/error", nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		req.Header.Set("Accept", "application/json")

		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusBadRequest {
			t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusBadRequest)
		}

		// Check content type
		contentType := rr.Header().Get("Content-Type")
		if contentType != "application/json" {
			t.Errorf("Handler returned wrong content type: got %v want %v", contentType, "application/json")
		}

		t.Logf("Response body: %s", rr.Body.String())

		// Check the response body
		var errorResp map[string]any
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
