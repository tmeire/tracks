package tracks

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMiddlewares_Wrap(t *testing.T) {
	// Test that globalMiddlewares are applied in reverse order (last added, first executed)
	t.Run("Middlewares are applied in reverse order", func(t *testing.T) {
		// Create a slice to record the order of middleware execution
		var executionOrder []string

		m := func(i int) Middleware {
			return func(h http.Handler) http.Handler {
				return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					executionOrder = append(executionOrder, fmt.Sprintf("middleware%d-before", i))
					h.ServeHTTP(w, r)
					executionOrder = append(executionOrder, fmt.Sprintf("middleware%d-after", i))
				})
			}
		}

		// Create a final handler
		finalHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			executionOrder = append(executionOrder, "handler")
			w.WriteHeader(http.StatusOK)
		})

		// Create a globalMiddlewares struct and add the globalMiddlewares in order
		ms := middlewares{}
		ms.Apply(m(1))
		ms.Apply(m(2))
		ms.Apply(m(3))

		// Wrap the final handler with the globalMiddlewares
		wrappedHandler := ms.Wrap(finalHandler)

		// Create a test request
		req := httptest.NewRequest("GET", "/test", nil)
		rr := httptest.NewRecorder()

		// Execute the wrapped handler
		wrappedHandler.ServeHTTP(rr, req)

		// Check the status code
		if status := rr.Code; status != http.StatusOK {
			t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}

		// Check the execution order
		expectedOrder := []string{
			"middleware1-before", // Last added, first executed
			"middleware2-before",
			"middleware3-before", // First added, last executed
			"handler",            // The final handler
			"middleware3-after",  // Unwinding the stack
			"middleware2-after",
			"middleware1-after", // Last added, last unwound
		}

		// Print the actual execution order for debugging
		t.Logf("Actual execution order: %v", executionOrder)
		t.Logf("Expected execution order: %v", expectedOrder)

		// Check if the execution order matches the expected order
		if len(executionOrder) != len(expectedOrder) {
			t.Errorf("Wrong number of execution steps: got %d want %d", len(executionOrder), len(expectedOrder))
		}

		for i, step := range executionOrder {
			if i >= len(expectedOrder) {
				t.Errorf("Unexpected execution step at index %d: %s", i, step)
				continue
			}
			if step != expectedOrder[i] {
				t.Errorf("Wrong execution order at index %d: got %s want %s", i, step, expectedOrder[i])
			}
		}
	})
}
