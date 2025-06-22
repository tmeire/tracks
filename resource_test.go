package tracks

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"testing"

	"github.com/tmeire/tracks/database/sqlite"
)

// Product is a test struct for the Resource test
type Product struct {
	ID    string  `json:"id" xml:"id"`
	Name  string  `json:"name" xml:"name"`
	Price float64 `json:"price" xml:"price"`
}

// ProductResource implements the Resource interface for testing
type ProductResource struct{}

func (r ProductResource) BasePath() string {
	return "/products"
}

func (r ProductResource) Index(req *http.Request) (interface{}, error) {
	products := []Product{
		{ID: "1", Name: "Product 1", Price: 10.99},
		{ID: "2", Name: "Product 2", Price: 20.99},
	}
	return products, nil
}

func (r ProductResource) New(req *http.Request) (interface{}, error) {
	return "New Product Form", nil
}

func (r ProductResource) Create(req *http.Request) (interface{}, error) {
	// For status codes other than OK, we need to return a Response object
	return &Response{
		StatusCode: http.StatusCreated,
		Data:       Product{ID: "3", Name: "New Product", Price: 15.99},
	}, nil
}

func (r ProductResource) Show(req *http.Request) (interface{}, error) {
	return Product{ID: "1", Name: "Product 1", Price: 10.99}, nil
}

func (r ProductResource) Edit(req *http.Request) (interface{}, error) {
	return "Edit Product Form", nil
}

func (r ProductResource) Update(req *http.Request) (interface{}, error) {
	return Product{ID: "1", Name: "Updated Product", Price: 12.99}, nil
}

func (r ProductResource) Destroy(req *http.Request) (interface{}, error) {
	// For status codes other than OK, we need to return a Response object
	return &Response{
		StatusCode: http.StatusNoContent,
		Data:       nil,
	}, nil
}

func writeTemplate(file, content string) error {
	if err := os.MkdirAll(path.Dir(file), 0755); err != nil {
		return err
	}
	return os.WriteFile(file, []byte(content), 0644)
}

// TestResource tests the Resource functionality
func TestResource(t *testing.T) {
	err := writeTemplate("views/product/index.gohtml", `
    <h1>Products List</h1>
    <ul>
    {{range .}}
        <li>{{.Name}} - ${{.Price}}</li>
    {{end}}
    </ul>
`)
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
		os.RemoveAll("views/product/")
	}()

	// Create a temporary database for testing
	tempDB, err := sqlite.New(":memory:")
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer tempDB.Close()

	// Create a new router
	router := New(t.Context(), tempDB)

	// Module the resource
	router.Resource(ProductResource{})

	h, _ := router.Handler()

	// Test Index action
	t.Run("Index Action", func(t *testing.T) {
		req, err := http.NewRequest("GET", "/product/", nil)
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

		// Check the response body
		var products []Product
		if err := json.Unmarshal(rr.Body.Bytes(), &products); err != nil {
			t.Errorf("Failed to unmarshal JSON response: %v", err)
		}

		if len(products) != 2 {
			t.Errorf("Handler returned unexpected number of products: got %v want %v", len(products), 2)
		}
	})

	// Test Show action
	t.Run("Show Action", func(t *testing.T) {
		req, err := http.NewRequest("GET", "/product/1", nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		req.Header.Set("Accept", "application/json")

		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}

		// Check the response body
		var product Product
		if err := json.Unmarshal(rr.Body.Bytes(), &product); err != nil {
			t.Errorf("Failed to unmarshal JSON response: %v", err)
		}

		if product.ID != "1" {
			t.Errorf("Handler returned unexpected product ID: got %v want %v", product.ID, "1")
		}
	})
}
