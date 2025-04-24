package tracks

import "net/http"

// Resource is an interface for RESTful resources
type Resource interface {
	// Index shows a list of resources
	Index(r *http.Request) (any, error)
	// New shows the form to create a new resource
	New(r *http.Request) (any, error)
	// Create accepts a POST request to create a new resource
	Create(r *http.Request) (any, error)
	// Show displays the information for a single resource
	Show(r *http.Request) (any, error)
	// Edit shows the form to update an existing resource
	Edit(r *http.Request) (any, error)
	// Update updates an existing resource
	Update(r *http.Request) (any, error)
	// Destroy destroys an existing resource
	Destroy(r *http.Request) (any, error)
}
