package tracks

import (
	"net/http"
	"path/filepath"
	"reflect"
	"strings"
)

type Router struct {
	mux         *http.ServeMux
	middlewares Middlewares
}

func New() *Router {
	return &Router{
		http.NewServeMux(),
		[]Middleware{},
	}
}

func (t *Router) normalize(path string) string {
	path = strings.ToLower(path)
	path = strings.TrimSuffix(path, "resource")

	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return path
}

// ServeHTTP is the HTTP handler implementation for the Router.
//
// This method allows the Router to act as an http.Handler, which means it can be used directly
// as the root handler for an HTTP server. It delegates the actual request handling to the
// underlying ServeMux, which routes requests based on their registered handlers.
//
// Parameters:
// - w: An http.ResponseWriter used to send the HTTP response back to the client.
// - r: A pointer to the http.Request representing the incoming HTTP request.
//
// Example usage:
//
//	router := tracks.New()
//	http.ListenAndServe(":8080", router)
func (t *Router) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	t.mux.ServeHTTP(w, r)
}

// serve is a helper method for the Router type that registers a handler for a specific HTTP method
// and path in the ServeMux. It constructs a key using the HTTP method and a normalized path,
// then registers the provided Action as the handler for that key.
//
// Parameters:
// - method: the HTTP method (e.g., "GET", "POST", etc.).
// - path: the URL path for which the handler should be registered.
// - r: the Action function that handles HTTP requests.
func (t *Router) serve(method, urlPath string, controller, action string, a Action) {
	normalizedPath := t.normalize(urlPath)

	t.mux.Handle(method+" "+normalizedPath, t.middlewares.Wrap(wrap(controller, action, a)))
}

// Get registers a handler for HTTP GET requests to the specified path.
//
// Parameters:
// - path: the URL path for which the handler should be registered.
// - r: the Action to handle the HTTP GET request.
func (t *Router) Get(path string, controller, action string, r Action) {
	t.serve(http.MethodGet, path, controller, action, r)
}

// Post registers a handler for HTTP POST requests to the specified path.
//
// Parameters:
// - path: the URL path for which the handler should be registered.
// - r: the Action to handle the HTTP POST request.
func (t *Router) Post(path string, controller, action string, r Action) {
	t.serve(http.MethodPost, path, controller, action, r)
}

// Put registers a handler for HTTP PUT requests to the specified path.
//
// Parameters:
// - path: the URL path for which the handler should be registered.
// - r: the Action to handle the HTTP PUT request.
func (t *Router) Put(path string, controller, action string, r Action) {
	t.serve(http.MethodPut, path, controller, action, r)
}

// Patch registers a handler for HTTP PATCH requests to the specified path.
//
// Parameters:
// - path: the URL path for which the handler should be registered.
// - r: the Action to handle the HTTP PATCH request.
func (t *Router) Patch(path string, controller, action string, r Action) {
	t.serve(http.MethodPatch, path, controller, action, r)
}

// Delete registers a handler for HTTP DELETE requests to the specified path.
//
// Parameters:
// - path: the URL path for which the handler should be registered.
// - r: the Action to handle the HTTP DELETE request.
func (t *Router) Delete(path string, controller, action string, r Action) {
	t.serve(http.MethodDelete, path, controller, action, r)
}

// Resource registers a resourceful route for a given resource and tasks it with handling
// a set of HTTP methods (GET, POST, PUT, DELETE) for resourceful paths.
//
// A resource provides CRUD-like routes including:
//
// - Index: Handles GET requests for the base path (e.g., `/resource`).
// - New: Handles GET requests for creating a new resource (e.g., `/resource/new`).
// - Create: Handles POST requests to create a new resource at the base path.
// - Show: Handles GET requests for a specific resource identified by its ID (e.g., `/resource/:id`).
// - Edit: Handles GET requests to edit a specific resource (e.g., `/resource/:id/edit`).
// - Update: Handles PUT requests for updating a specific resource (e.g., `/resource/:id`).
// - Destroy: Handles DELETE requests for deleting a specific resource (e.g., `/resource/:id`).
//
// Parameters:
// - name: The name of the resource (e.g., "users").
// - r: An implementation of the Resource interface, providing handler methods for resourceful paths.
//
// Returns:
// - A pointer to the sub-router created for the resource.
func (t *Router) Resource(r Resource) {
	rt := reflect.TypeOf(r)
	if rt.Kind() == reflect.Ptr {
		rt = rt.Elem()
	}
	name := strings.TrimSuffix(strings.ToLower(rt.Name()), "resource")

	basePath := t.normalize(name)

	// Register resource actions with the controller name
	t.Get(basePath+"/", name, "index", r.Index)
	t.Get(basePath+"/new", name, "new", r.New)
	t.Post(basePath+"/", name, "create", r.Create)

	t.Get(basePath+"/{id}", name, "show", r.Show)
	t.Get(basePath+"/{id}/edit", name, "edit", r.Edit)
	t.Put(basePath+"/{id}", name, "update", r.Update)
	t.Post(basePath+"/{id}", name, "update", r.Update)

	t.Delete(basePath+"/{id}", name, "destroy", r.Destroy)
}

func (t *Router) Middleware(m Middleware) {
	t.middlewares = append(t.middlewares, m)
}

// Static registers a directory to serve static files from.
// The provided path will be used as the base URL path for the static files.
// For example, if path is "/assets", then files in the directory will be
// available at "/assets/filename".
//
// Parameters:
// - urlPath: the URL path prefix for static files (e.g., "/assets").
// - dir: the directory containing the static files to serve.
func (t *Router) Static(urlPath, dir string) {
	// Ensure the path starts with a slash
	if !strings.HasPrefix(urlPath, "/") {
		urlPath = "/" + urlPath
	}

	// Ensure the path ends with a slash for proper path matching
	if !strings.HasSuffix(urlPath, "/") {
		urlPath = urlPath + "/"
	}

	// Create a file server handler for the specified directory
	fileServer := http.FileServer(http.Dir(dir))

	// Strip the URL path prefix when looking for files
	// This allows the file server to correctly map URL paths to file paths
	handler := http.StripPrefix(urlPath, fileServer)

	// Register the handler for the URL path
	t.mux.Handle(http.MethodGet+" "+urlPath, handler)

	// Also handle requests for files within the directory
	t.mux.Handle(http.MethodGet+" "+filepath.Join(urlPath, "*"), handler)
}
