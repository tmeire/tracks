package tracks

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/tmeire/tracks/database"
	"github.com/tmeire/tracks/i18n"
	"github.com/tmeire/tracks/otel"
	"github.com/tmeire/tracks/session"
	sessiondb "github.com/tmeire/tracks/session/db"
)

type Router interface {
	Clone() Router
	Secure() bool
	BaseDomain() string
	Port() int
	Database() database.Database
	ServeHTTP(w http.ResponseWriter, r *http.Request)
	Module(m Module) Router
	Middleware(m Middleware) Router
	Func(name string, fn any) Router
	Views(path string) Router
	Page(path string, view string) Router
	Redirect(origin string, destination string) Router
	Get(path string, controller, action string, r Controller) Router
	GetFunc(path string, controller, action string, r Action) Router
	PostFunc(path string, controller, action string, r Action) Router
	PutFunc(path string, controller, action string, r Action) Router
	PatchFunc(path string, controller, action string, r Action) Router
	DeleteFunc(path string, controller, action string, r Action) Router
	Resource(r Resource) Router
	Run()
}

type router struct {
	port        int
	baseDomain  string
	database    database.Database
	mux         *http.ServeMux
	middlewares *middlewares
	templates   Templates
	translator  *i18n.Translator
}

// New creates a new router with a database-backed session store
func New(baseDomain string, db database.Database) Router {
	portVal := os.Getenv("PORT")
	port, err := strconv.Atoi(portVal)
	if err != nil || port == 0 {
		port = 8080
	}

	// Initialize the translator with English as the default language
	translator := i18n.NewTranslator("en")

	// Try to load translations from the translations directory
	err = translator.LoadTranslations("./translations")
	if err != nil {
		log.Printf("Failed to load translations: %v", err)
		// Continue without translations, using keys as fallback
	}

	r := &router{
		port:        port,
		baseDomain:  baseDomain,
		database:    db,
		mux:         http.NewServeMux(),
		middlewares: &middlewares{},
		translator:  translator,
		templates: Templates{
			basedir: "./views",
			fns: template.FuncMap{
				"t": func(key string) template.HTML {
					// This is a placeholder implementation to make sure the templates can be loaded on boot.
					// Every request will overwrite this func with a method that contains the request context to make
					// sure it's able to access the requested language.
					return template.HTML(key)
				},
				"now": func() string {
					return time.Now().Format("2006-01-02T15:04")
				},
				"today": func() string {
					return time.Now().Format(time.DateOnly)
				},
				"year": func() string {
					return time.Now().Format("2006")
				},
				"add": func(a, b int) int {
					return a + b
				},
				"link": func(s string) template.URL {
					// TODO: very naive implementation
					if s[0] != '/' {
						s = "/" + s
					}
					return template.URL("//" + baseDomain + s)
				},
			},
		},
	}

	// HTTP traces for every request
	r.Middleware(otel.Trace)

	// Catch all panics to make sure no weird output is written to the client
	r.Middleware(CatchAll)

	// Serving static files on the root domain
	r.static("/assets/", "public")

	r.Middleware(database.Middleware(db))

	// Set up i18n middleware for language detection
	r.Middleware(i18n.Middleware("en"))

	// Set up sessions for all the domains
	r.Middleware(session.Middleware(
		baseDomain,
		sessiondb.NewStore(db),
	))

	return r
}

func (r *router) Clone() Router {
	return &router{
		port:        r.port,
		baseDomain:  r.baseDomain,
		database:    r.database,
		mux:         http.NewServeMux(),
		middlewares: r.middlewares,
		templates:   r.templates,
		translator:  r.translator,
	}
}

// Secure returns true of all the links on the site should use HTTPS
func (r *router) Secure() bool {
	// TODO: if all traffic is served over https; for now the same as serveTLS, but this could be different if TLS is terminated by a proxy
	return false
}

func (r *router) BaseDomain() string {
	return r.baseDomain
}

func (r *router) Port() int {
	return r.port
}

func (r *router) Database() database.Database {
	return r.database
}

func (r *router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	r.mux.ServeHTTP(w, req)
}

func (r *router) normalize(path string) string {
	path = strings.ToLower(path)
	path = strings.TrimSuffix(path, "resource")

	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return path
}

// serve is a helper method for the router type that registers a handler for a specific HTTP method
// and path in the ServeMux. It constructs a key using the HTTP method and a normalized path,
// then registers the provided Action as the handler for that key.
//
// Parameters:
// - method: the HTTP method (e.g., "GET", "POST", etc.).
// - path: the URL path for which the handler should be registered.
// - r: the Action function that handles HTTP requests.
func (r *router) serve(method, urlPath string, controller, action string, a Action) {
	normalizedPath := r.normalize(urlPath)

	pattern := method + " " + normalizedPath
	println(pattern)

	tpl := template.Must(r.templates.Load(controller, action))

	r.mux.Handle(pattern, r.middlewares.Wrap(a.wrap(controller, action, tpl, r.translator)))
}

// static registers a directory to serve static files from.
// The provided path will be used as the base URL path for the static files.
// For example, if path is "/assets", then files in the directory will be
// available at "/assets/filename".
//
// Parameters:
// - urlPath: the URL path prefix for static files (e.g., "/assets").
// - dir: the directory containing the static files to serve.
func (r *router) static(urlPath, dir string) Router {
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
	r.mux.Handle(http.MethodGet+" "+urlPath, handler)

	return r
}

type Module func(Router) Router

// Module registers all the module functionality (controllers, middlewares,...) into the router.
// This is equivalent to `m(r)`, but enables chaining when setting up the router.
func (r *router) Module(m Module) Router {
	return m(r)
}

func (r *router) Middleware(m Middleware) Router {
	r.middlewares.Apply(m)
	return r
}

func (r *router) Func(name string, fn any) Router {
	r.templates.Func(name, fn)
	return r
}

func (r *router) Views(path string) Router {
	r.templates.basedir = path
	r.templates.layout = nil
	return r
}

// Page registers a handler for HTTP GET requests to the specified path.
//
// Parameters:
// - path: the URL path for which the handler should be registered.
// - view: the "controller#action" name of the view to serve as the response to the HTTP GET request.
//
// Returns:
// - A pointer to the router instance, enabling chaining.
func (r *router) Page(path string, view string) Router {
	parts := strings.Split(view, "#")
	if len(parts) > 2 {
		panic("invalid view " + view)
	}

	var controller, action string
	if len(parts) == 1 {
		action = parts[0]
	} else {
		controller = parts[0]
		action = parts[1]
	}

	if controller == "" {
		controller = "default"
	}

	return r.GetFunc(path, controller, action, func(r *http.Request) (any, error) {
		return "", nil
	})
}

func (r *router) Redirect(origin string, destination string) Router {
	r.mux.Handle(origin, http.RedirectHandler(destination, http.StatusMovedPermanently))
	return r
}

// Get registers a handler for HTTP GET requests to the specified path.
//
// Parameters:
// - path: the URL path for which the handler should be registered.
// - r: the Action to handle the HTTP GET request.
func (r *router) Get(path string, controller, action string, c Controller) Router {
	if nr, needsRouter := c.(interface {
		Inject(r Router)
	}); needsRouter {
		nr.Inject(r)
	}
	r.serve(http.MethodGet, path, controller, action, c.Index)
	return r
}

// GetFunc registers a handler for HTTP GET requests to the specified path.
//
// Parameters:
// - path: the URL path for which the handler should be registered.
// - r: the Action to handle the HTTP GET request.
func (r *router) GetFunc(path string, controller, action string, a Action) Router {
	r.serve(http.MethodGet, path, controller, action, a)
	return r
}

// PostFunc registers a handler for HTTP POST requests to the specified path.
//
// Parameters:
// - path: the URL path for which the handler should be registered.
// - r: the Action to handle the HTTP POST request.
func (r *router) PostFunc(path string, controller, action string, a Action) Router {
	r.serve(http.MethodPost, path, controller, action, a)
	return r
}

// PutFunc registers a handler for HTTP PUT requests to the specified path.
//
// Parameters:
// - path: the URL path for which the handler should be registered.
// - r: the Action to handle the HTTP PUT request.
func (r *router) PutFunc(path string, controller, action string, a Action) Router {
	r.serve(http.MethodPut, path, controller, action, a)
	return r
}

// PatchFunc registers a handler for HTTP PATCH requests to the specified path.
//
// Parameters:
// - path: the URL path for which the handler should be registered.
// - r: the Action to handle the HTTP PATCH request.
func (r *router) PatchFunc(path string, controller, action string, a Action) Router {
	r.serve(http.MethodPatch, path, controller, action, a)
	return r
}

// DeleteFunc registers a handler for HTTP DELETE requests to the specified path.
//
// Parameters:
// - path: the URL path for which the handler should be registered.
// - r: the Action to handle the HTTP DELETE request.
func (r *router) DeleteFunc(path string, controller, action string, a Action) Router {
	r.serve(http.MethodDelete, path, controller, action, a)
	return r
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
// - r: An implementation of the Resource interface, providing handler methods for resourceful paths.
//
// Returns:
// - A pointer to the sub-router created for the resource.
func (r *router) Resource(rs Resource) Router {
	// This little piece of reflection is OK since it only runs once on boot,
	// it's not a reflection penalty on every request.
	rt := reflect.TypeOf(rs)
	if rt.Kind() == reflect.Ptr {
		rt = rt.Elem()
	}
	name := strings.TrimSuffix(strings.ToLower(rt.Name()), "resource")

	r.registerResource("/", name, rs)

	if withSubresouces, ok := rs.(interface {
		Subresources() map[string]Resource
	}); ok {
		rootPath := "/" + name + fmt.Sprintf("/{%s_id}", name)

		for path, sr := range withSubresouces.Subresources() {
			r.registerResource(rootPath, path, sr)
		}
	}
	return r
}

func (r *router) registerResource(rootPath, name string, rs Resource) {
	pathParamName := fmt.Sprintf(`{%s_id}`, name)

	basePath := filepath.Join(rootPath, r.normalize(name))

	// Register resource actions with the controller name
	r.GetFunc(basePath+"/", name, "index", rs.Index)
	r.GetFunc(basePath+"/new", name, "new", rs.New)
	r.PostFunc(basePath+"/", name, "create", rs.Create)

	r.GetFunc(basePath+"/"+pathParamName, name, "show", rs.Show)
	r.GetFunc(basePath+"/"+pathParamName+"/edit", name, "edit", rs.Edit)
	r.PutFunc(basePath+"/"+pathParamName, name, "update", rs.Update)
	r.PostFunc(basePath+"/"+pathParamName, name, "update", rs.Update)

	r.DeleteFunc(basePath+"/"+pathParamName, name, "destroy", rs.Destroy)
}

// Run starts the HTTP server using the router as the handler on the specified port or default port 8080 if unset.
// It retrieves the port from the PORT environment variable and logs the server address before starting it.
func (r *router) Run() {
	Serve(r, r.port)
}

func Serve(h http.Handler, port int) {
	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: h,
	}

	done := make(chan struct{})

	go func() {
		defer close(done)

		err := server.ListenAndServe()
		if err != nil {
			log.Fatalf("HTTP server failed: %v", err)
		}
	}()

	// This will block the function until the server stops or an OS signal is received
	select {
	case <-sigc:
		err := server.Close()
		if err != nil {
			panic(err)
		}
	case <-done:
	}
}
