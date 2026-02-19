package tracks

import (
	"context"
	"fmt"
	"html/template"
	"log"
	"log/slog"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"path/filepath"
	"reflect"
	"strings"
	"syscall"

	"github.com/tmeire/tracks/database"
	"github.com/tmeire/tracks/i18n"
	"github.com/tmeire/tracks/otel"

	"github.com/iancoleman/strcase"
)

type Router interface {
	Clone() Router
	Secure() bool
	BaseDomain() string
	Database() database.Database
	Module(m Module) Router
	GlobalMiddleware(m Middleware) Router
	DomainMiddleware() Middleware
	DomainDatabase(config DomainDBConfig) Router
	DomainScopedRepositories() Router
	RequestMiddleware(m Middleware) Router
	Static(urlPath, dir string) Router
	StaticWithConfig(urlPath, dir string, config StaticConfig) Router
	Content(path, dir string, config ContentConfig) Router
	LogHostEntries() Router
	LogHostEntriesWithMessage(message string) Router
	HealthCheck(path string, config ...HealthConfig) Router
	Version(v string, config ...VersionConfig) Router
	VersionFromHeader(header, value string, r Router) Router
	VersionFromQuery(param, value string, r Router) Router
	RateLimit(config RateLimitConfig) Router
	CSRFProtection(config CSRFConfig) Router
	Cache() Cache
	WithCache(c Cache) Router
	Queue() Queue
	Func(name string, fn any) Router
	Views(path string) Router
	Page(path string, view string) Router
	Redirect(origin string, destination string) Router
	Serve(a Action) Router
	Controller(c Controller) Router
	ControllerAtPath(path string, c Controller) Router
	Get(path string, controller, action string, r ActionController, mws ...MiddlewareBuilder) Router
	GetFunc(path string, controller, action string, r ActionFunc, mws ...MiddlewareBuilder) Router
	PostFunc(path string, controller, action string, r ActionFunc, mws ...MiddlewareBuilder) Router
	PutFunc(path string, controller, action string, r ActionFunc, mws ...MiddlewareBuilder) Router
	PatchFunc(path string, controller, action string, r ActionFunc, mws ...MiddlewareBuilder) Router
	DeleteFunc(path string, controller, action string, r ActionFunc, mws ...MiddlewareBuilder) Router
	Resource(r Resource, mws ...MiddlewareBuilder) Router
	ResourceAtPath(path string, r Resource, mws ...MiddlewareBuilder) Router
	Templates() *Templates
	Config() Config
	Handler() (http.Handler, error)
	Run(ctx context.Context) error
}

type router struct {
	parent             *router
	config             Config
	port               int
	baseDomain         string
	database           database.Database
	cache              Cache
	queue              Queue
	mux                *http.ServeMux
	globalMiddlewares  *middlewares
	requestMiddlewares *middlewares
	templates          *Templates
	translator         *i18n.Translator
	shutdownOtel       otel.Shutdown
}

// New creates a new router with a database-backed session store
func New(ctx context.Context) Router {
	conf, err := loadConfig()
	if err != nil {
		slog.ErrorContext(ctx, "Failed to load config", "error", err)
		return errRouter{err: err}
	}
	return NewFromConfig(ctx, conf)
}

func NewFromConfig(ctx context.Context, conf Config) Router {
	// Set up OpenTelemetry
	shutdownOtel, err := otel.Setup(ctx, conf.Name, conf.Version)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to initialize tracer provider", "error", err)
		// Continue without tracing
	}

	db, err := conf.Database.Create(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to create database connection", "error", err)
		return errRouter{err: err}
	}

	// Initialize the translator with English as the default language
	translator := i18n.NewTranslator("en")

	// Try to load translations from the translations directory
	err = translator.LoadTranslations("./translations")
	if err != nil {
		slog.ErrorContext(ctx, "Failed to load translations, continuing without", "error", err)
		// Continue without translations, using keys as fallback
	}

	var c Cache
	if conf.Cache.Driver == "memory" {
		c = NewMemoryCache()
	}

	var q Queue
	if conf.Jobs.Driver == "memory" {
		workers := conf.Jobs.Workers
		if workers == 0 {
			workers = 5
		}
		q = NewMemoryQueue(workers)
	}

	r := &router{
		parent:             nil,
		config:             conf,
		port:               conf.Port,
		baseDomain:         conf.BaseDomain,
		database:           db,
		cache:              c,
		queue:              q,
		mux:                http.NewServeMux(),
		globalMiddlewares:  &middlewares{},
		requestMiddlewares: &middlewares{},
		translator:         translator,
		shutdownOtel:       shutdownOtel,
		templates:          newTemplates(conf.BaseDomain),
	}

	// HTTP traces for every request
	r.GlobalMiddleware(otel.Trace)

	// Inject queue into context
	if q != nil {
		r.GlobalMiddleware(func(next http.Handler) (http.Handler, error) {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				ctx := WithQueue(r.Context(), q)
				next.ServeHTTP(w, r.WithContext(ctx))
			}), nil
		})
	}

	// Inject cache into context
	if c != nil {
		r.GlobalMiddleware(func(next http.Handler) (http.Handler, error) {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				ctx := WithCache(r.Context(), c)
				next.ServeHTTP(w, r.WithContext(ctx))
			}), nil
		})
	}

	// Extract and store the full domain context
	r.GlobalMiddleware(r.DomainMiddleware())

	// Catch all panics to make sure no weird output is written to the client
	r.GlobalMiddleware(CatchAll)

	// Serving static files on the root domain
	r.static("/assets/", "public")

	r.GlobalMiddleware(database.Middleware(db))

	// Set up i18n middleware for language detection
	r.GlobalMiddleware(i18n.Middleware(translator, "en"))

	// Expose the detected locale in the view context
	r.GlobalMiddleware(func(next http.Handler) (http.Handler, error) {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			lang := i18n.LanguageFromContext(r.Context())
			r = AddViewVar(r, "locale", lang)
			next.ServeHTTP(w, r)
		}), nil
	})

	// Set up sessions for all the domains
	sessionMW, err := conf.Sessions.Middleware(ctx, conf.BaseDomain)
	if err != nil {
		log.Printf("Failed to create session middleware: %v", err)
		return errRouter{err: err}
	}
	r.GlobalMiddleware(sessionMW)

	return r
}

func (r *router) Clone() Router {
	return &router{
		parent:            r,
		config:            r.config,
		port:              r.port,
		baseDomain:        r.baseDomain,
		database:          r.database,
		mux:               http.NewServeMux(),
		globalMiddlewares: &middlewares{},
		requestMiddlewares: &middlewares{
			l: r.requestMiddlewares.l,
		},
		templates:  r.templates,
		translator: r.translator,
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

func (r *router) normalize(path string) string {
	path = strings.ToLower(path)
	if strings.HasSuffix(path, "resource") {
		path = strings.TrimSuffix(path, "resource")
	}
	if strings.HasSuffix(path, "controller") {
		path = strings.TrimSuffix(path, "controller")
	}
	path = strings.TrimSuffix(path, "_")

	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return path
}

const defaultLayout = "application"

// serve is a helper method for the router type that registers a handler for a specific HTTP method
// and path in the ServeMux. It constructs a key using the HTTP method and a normalized path,
// then registers the provided ActionFunc as the handler for that key.
//
// Parameters:
// - method: the HTTP method (e.g., "GET", "POST", etc.).
// - path: the URL path for which the handler should be registered.
// - r: the ActionFunc function that handles HTTP requests.
func (r *router) serve(method, urlPath string, controller, action string, a ActionFunc, layout string, mws ...MiddlewareBuilder) Router {
	normalizedPath := r.normalize(urlPath)

	pattern := method + " " + normalizedPath
	println(pattern)

	if layout == "" {
		layout = defaultLayout
	}

	tpl, err := r.templates.Load(layout, controller, action)
	if err != nil {
		return errRouter{err}
	}

	h, err := r.requestMiddlewares.Wrap(r, a.wrap(controller, action, tpl, r.translator), mws...)
	if err != nil {
		return errRouter{err}
	}

	r.mux.Handle(pattern, h)
	return r
}

type StaticConfig struct {
	CacheControl   string
	StripPrefix    bool
	DisableListing bool
}

// Static registers a directory to serve static files from.
func (r *router) Static(urlPath, dir string) Router {
	return r.StaticWithConfig(urlPath, dir, StaticConfig{
		StripPrefix: true,
	})
}

// StaticWithConfig registers a directory to serve static files from with additional configuration.
func (r *router) StaticWithConfig(urlPath, dir string, config StaticConfig) Router {
	// Ensure the path starts with a slash
	if !strings.HasPrefix(urlPath, "/") {
		urlPath = "/" + urlPath
	}

	// Ensure the path ends with a slash for proper path matching
	if !strings.HasSuffix(urlPath, "/") {
		urlPath = urlPath + "/"
	}

	// Create a file server handler for the specified directory
	var handler http.Handler
	if config.DisableListing {
		// Custom file server that disables listing could be implemented here
		// For now, use standard FileServer
		handler = http.FileServer(http.Dir(dir))
	} else {
		handler = http.FileServer(http.Dir(dir))
	}

	// Strip the URL path prefix when looking for files
	if config.StripPrefix {
		handler = http.StripPrefix(urlPath, handler)
	}

	// Add cache control if configured
	if config.CacheControl != "" {
		next := handler
		handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Cache-Control", config.CacheControl)
			next.ServeHTTP(w, r)
		})
	}

	// Register the handler for the URL path
	r.mux.Handle(http.MethodGet+" "+urlPath, handler)

	return r
}

// static is an internal helper that registers a directory to serve static files from.
func (r *router) static(urlPath, dir string) Router {
	return r.Static(urlPath, dir)
}

type Module func(Router) Router

// Module registers all the module functionality (controllers, middlewares,...) into the router.
// This is equivalent to `m(r)`, but enables chaining when setting up the router.
func (r *router) Module(m Module) Router {
	return m(r)
}

func (r *router) GlobalMiddleware(m Middleware) Router {
	r.globalMiddlewares.Apply(m)
	return r
}

func (r *router) DomainMiddleware() Middleware {
	return DomainMiddleware()
}

func (r *router) DomainDatabase(config DomainDBConfig) Router {
	r.GlobalMiddleware(DomainDatabase(config))
	return r
}

func (r *router) DomainScopedRepositories() Router {
	r.GlobalMiddleware(func(next http.Handler) (http.Handler, error) {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := database.WithDomainFiltering(r.Context(), true)
			next.ServeHTTP(w, r.WithContext(ctx))
		}), nil
	})
	return r
}

func (r *router) Content(path, dir string, config ContentConfig) Router {
	// Generic registration is tricky with the current router API since it's not generic.
	// But we can use a helper or just register the controller.
	// However, ContentController is generic.
	// Let's use a non-generic wrapper if needed, or just let users use NewContentController.
	// For the sake of the API request, I'll use a type-agnostic controller for registration.
	c := NewContentController[map[string]any](dir, config)
	return c.Register(r, path)
}

func (r *router) LogHostEntries() Router {
	return r.LogHostEntriesWithMessage("Add these entries to /etc/hosts for local testing:")
}

func (r *router) LogHostEntriesWithMessage(message string) Router {
	if !r.config.Development {
		return r
	}

	fmt.Println("\n=== Local Development Setup ===")
	fmt.Println(message)
	fmt.Println()

	for _, domain := range r.config.Domains {
		fullDomain := domain
		if r.config.BaseDomain != "" && !strings.Contains(domain, ".") {
			fullDomain = fmt.Sprintf("%s.%s", domain, r.config.BaseDomain)
		}
		fmt.Printf("127.0.0.1  %s\n", fullDomain)
	}

	fmt.Println("\nOr run this command:")
	var domains []string
	for _, domain := range r.config.Domains {
		fullDomain := domain
		if r.config.BaseDomain != "" && !strings.Contains(domain, ".") {
			fullDomain = fmt.Sprintf("%s.%s", domain, r.config.BaseDomain)
		}
		domains = append(domains, fullDomain)
	}
	if len(domains) > 0 {
		fmt.Printf("sudo sh -c 'echo \"127.0.0.1 %s\" >> /etc/hosts'\n", strings.Join(domains, " "))
	}
	fmt.Println("==============================")

	return r
}

func (r *router) VersionFromHeader(header, value string, vr Router) Router {
	r.GlobalMiddleware(func(next http.Handler) (http.Handler, error) {
		h, err := vr.Handler()
		if err != nil {
			return nil, err
		}
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			if req.Header.Get(header) == value {
				h.ServeHTTP(w, req)
				return
			}
			next.ServeHTTP(w, req)
		}), nil
	})
	return r
}

func (r *router) VersionFromQuery(param, value string, vr Router) Router {
	r.GlobalMiddleware(func(next http.Handler) (http.Handler, error) {
		h, err := vr.Handler()
		if err != nil {
			return nil, err
		}
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			if req.URL.Query().Get(param) == value {
				h.ServeHTTP(w, req)
				return
			}
			next.ServeHTTP(w, req)
		}), nil
	})
	return r
}

func (r *router) RateLimit(config RateLimitConfig) Router {
	r.GlobalMiddleware(RateLimitMiddleware(config)(r))
	return r
}

func (r *router) CSRFProtection(config CSRFConfig) Router {
	r.GlobalMiddleware(CSRFProtection(config))
	return r
}

func (r *router) Cache() Cache {
	return r.cache
}

func (r *router) WithCache(c Cache) Router {
	r.cache = c
	return r
}

func (r *router) Queue() Queue {
	return r.queue
}

func (r *router) RequestMiddleware(m Middleware) Router {
	r.requestMiddlewares.Apply(m)
	return r
}

func (r *router) Func(name string, fn any) Router {
	r.templates.Func(name, fn)
	return r
}

func (r *router) Views(path string) Router {
	r.templates.basedir = path
	r.templates.layouts = make(map[string]*template.Template)
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

func (r *router) Serve(a Action) Router {
	return r.serve(a.Method, a.Path, a.Controller, a.Name, a.Func, a.Layout, a.Middlewares...)
}

func (r *router) Controller(c Controller) Router {
	return c.Register(r, "/")
}

func (r *router) ControllerAtPath(path string, c Controller) Router {
	return c.Register(r, path)
}

// Get registers a handler for HTTP GET requests to the specified path.
//
// Parameters:
// - path: the URL path for which the handler should be registered.
// - r: the ActionFunc to handle the HTTP GET request.
func (r *router) Get(path string, controller, action string, c ActionController, mws ...MiddlewareBuilder) Router {
	if nr, needsRouter := c.(interface {
		Inject(r Router)
	}); needsRouter {
		nr.Inject(r)
	}
	return r.serve(http.MethodGet, path, controller, action, c.Index, defaultLayout, mws...)
}

// GetFunc registers a handler for HTTP GET requests to the specified path.
//
// Parameters:
// - path: the URL path for which the handler should be registered.
// - r: the ActionFunc to handle the HTTP GET request.
func (r *router) GetFunc(path string, controller, action string, a ActionFunc, mws ...MiddlewareBuilder) Router {
	return r.serve(http.MethodGet, path, controller, action, a, defaultLayout, mws...)
}

// PostFunc registers a handler for HTTP POST requests to the specified path.
//
// Parameters:
// - path: the URL path for which the handler should be registered.
// - r: the ActionFunc to handle the HTTP POST request.
func (r *router) PostFunc(path string, controller, action string, a ActionFunc, mws ...MiddlewareBuilder) Router {
	return r.serve(http.MethodPost, path, controller, action, a, defaultLayout, mws...)
}

// PutFunc registers a handler for HTTP PUT requests to the specified path.
//
// Parameters:
// - path: the URL path for which the handler should be registered.
// - r: the ActionFunc to handle the HTTP PUT request.
func (r *router) PutFunc(path string, controller, action string, a ActionFunc, mws ...MiddlewareBuilder) Router {
	return r.serve(http.MethodPut, path, controller, action, a, defaultLayout, mws...)
}

// PatchFunc registers a handler for HTTP PATCH requests to the specified path.
//
// Parameters:
// - path: the URL path for which the handler should be registered.
// - r: the ActionFunc to handle the HTTP PATCH request.
func (r *router) PatchFunc(path string, controller, action string, a ActionFunc, mws ...MiddlewareBuilder) Router {
	return r.serve(http.MethodPatch, path, controller, action, a, defaultLayout, mws...)
}

// DeleteFunc registers a handler for HTTP DELETE requests to the specified path.
//
// Parameters:
// - path: the URL path for which the handler should be registered.
// - r: the ActionFunc to handle the HTTP DELETE request.
func (r *router) DeleteFunc(path string, controller, action string, a ActionFunc, mws ...MiddlewareBuilder) Router {
	return r.serve(http.MethodDelete, path, controller, action, a, defaultLayout, mws...)
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
func (r *router) Resource(rs Resource, mws ...MiddlewareBuilder) Router {
	nr := r.ResourceAtPath("/", rs, mws...)

	return nr
}

func (r *router) ResourceAtPath(rootPath string, rs Resource, mws ...MiddlewareBuilder) Router {
	// This little piece of reflection is OK since it only runs once on boot,
	// it's not a reflection penalty on every request.
	rt := reflect.TypeOf(rs)
	if rt.Kind() == reflect.Ptr {
		rt = rt.Elem()
	}
	name := strcase.ToSnake(rt.Name())
	if strings.HasSuffix(name, "_resource") {
		name = strings.TrimSuffix(name, "_resource")
	}

	pathParamName := fmt.Sprintf(`{%s_id}`, name)

	basePath := filepath.Join(rootPath, r.normalize(name))

	// Register resource actions with the controller name
	nr := r.GetFunc(basePath+"/", name, "index", rs.Index, mws...).
		GetFunc(basePath+"/new", name, "new", rs.New, mws...).
		PostFunc(basePath+"/", name, "create", rs.Create, mws...).
		GetFunc(basePath+"/"+pathParamName, name, "show", rs.Show, mws...).
		GetFunc(basePath+"/"+pathParamName+"/edit", name, "edit", rs.Edit, mws...).
		PutFunc(basePath+"/"+pathParamName, name, "update", rs.Update, mws...).
		PostFunc(basePath+"/"+pathParamName, name, "update", rs.Update, mws...).
		DeleteFunc(basePath+"/"+pathParamName, name, "destroy", rs.Destroy, mws...)

	if nr, needsRouter := rs.(interface {
		Inject(r Router)
	}); needsRouter {
		nr.Inject(r)
	}

	// If this resource has subresources, register these as well.
	if withSubresouces, ok := rs.(interface {
		Subresources() []Resource
	}); ok {
		basePath = fmt.Sprintf("%s/{%s_id}/", basePath, name)

		for _, sr := range withSubresouces.Subresources() {
			nr = nr.ResourceAtPath(basePath, sr)
		}
	}

	// If this resource has subcontrollers, register these as well.
	if withSubcontrollers, ok := rs.(interface {
		Subcontrollers() []Controller
	}); ok {
		basePath = fmt.Sprintf("%s/{%s_id}/", basePath, name)

		for _, sr := range withSubcontrollers.Subcontrollers() {
			nr = nr.ControllerAtPath(basePath, sr)
		}
	}

	return nr
}

// Handler creates an HTTP handler for this router that can be used to launch
func (r *router) Templates() *Templates {
	return r.templates
}

func (r *router) Config() Config {
	return r.config
}

func (r *router) Handler() (http.Handler, error) {
	return r.globalMiddlewares.Wrap(r, r.mux)
}

// Run starts the HTTP server using the router as the handler on the specified port or default port 8080 if unset.
// It retrieves the port from the PORT environment variable and logs the server address before starting it.
func (r *router) Run(ctx context.Context) error {
	if r.parent != nil {
		return r.parent.Run(ctx)
	}
	h, err := r.Handler()
	if err != nil {
		return err
	}
	return r.run(ctx, h)
}

func (r *router) run(ctx context.Context, h http.Handler) error {
	defer func() {
		if err := r.shutdownOtel(ctx); err != nil {
			log.Fatalf("failed to shut down open telemetry provider: %v", err)
		}
	}()

	if r.queue != nil {
		if err := r.queue.Start(ctx); err != nil {
			return err
		}
		defer r.queue.Stop()
	}

	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", r.port),
		Handler: h,
	}

	done := make(chan struct{})

	var err error
	go func() {
		defer close(done)

		slog.InfoContext(ctx, "Starting server", "port", r.port)
		err = server.ListenAndServe()
		if err != nil {
			slog.ErrorContext(ctx, "HTTP server failed", "error", err)
		}
	}()

	go func() {
		slog.InfoContext(ctx, "Starting debug server on localhost:6060")
		err = http.ListenAndServe("localhost:6060", nil)
		if err != nil {
			slog.ErrorContext(ctx, "HTTP debug server failed", "error", err)
		}
	}()

	// This will block the function until the server stops or an OS signal is received
	select {
	case <-ctx.Done():
		err = server.Close()
		if err != nil {
			slog.ErrorContext(ctx, "Failed to close server", "error", err)
		}
	case <-sigc:
		err = server.Close()
		if err != nil {
			slog.ErrorContext(ctx, "Failed to close server", "error", err)
		}
	case <-done:
	}
	return err
}

type errRouter struct {
	err error
}

func (e errRouter) Clone() Router {
	return e
}

func (e errRouter) Secure() bool {
	return false
}

func (e errRouter) BaseDomain() string {
	return ""
}

func (e errRouter) Port() int {
	return 0
}

func (e errRouter) Database() database.Database {
	return nil
}

func (e errRouter) Module(m Module) Router {
	return e
}

func (e errRouter) GlobalMiddleware(m Middleware) Router {
	return e
}

func (e errRouter) DomainMiddleware() Middleware {
	return func(h http.Handler) (http.Handler, error) {
		return h, nil
	}
}

func (e errRouter) DomainDatabase(config DomainDBConfig) Router {
	return e
}

func (e errRouter) DomainScopedRepositories() Router {
	return e
}

func (e errRouter) Static(urlPath, dir string) Router {
	return e
}

func (e errRouter) StaticWithConfig(urlPath, dir string, config StaticConfig) Router {
	return e
}

func (e errRouter) Content(path, dir string, config ContentConfig) Router {
	return e
}

func (e errRouter) LogHostEntries() Router {
	return e
}

func (e errRouter) LogHostEntriesWithMessage(message string) Router {
	return e
}

func (e errRouter) HealthCheck(path string, config ...HealthConfig) Router {
	return e
}

func (e errRouter) Version(v string, config ...VersionConfig) Router {
	return e
}

func (e errRouter) VersionFromHeader(header, value string, r Router) Router {
	return e
}

func (e errRouter) VersionFromQuery(param, value string, r Router) Router {
	return e
}

func (e errRouter) RateLimit(config RateLimitConfig) Router {
	return e
}

func (e errRouter) CSRFProtection(config CSRFConfig) Router {
	return e
}

func (e errRouter) Cache() Cache {
	return nil
}

func (e errRouter) WithCache(c Cache) Router {
	return e
}

func (e errRouter) Queue() Queue {
	return nil
}

func (e errRouter) RequestMiddleware(m Middleware) Router {
	return e
}

func (e errRouter) Func(name string, fn any) Router {
	return e
}

func (e errRouter) Views(path string) Router {
	return e
}

func (e errRouter) Page(path string, view string) Router {
	return e
}

func (e errRouter) Redirect(origin string, destination string) Router {
	return e
}

func (e errRouter) Serve(a Action) Router {
	return e
}

func (e errRouter) Controller(c Controller) Router {
	return e
}

func (e errRouter) ControllerAtPath(path string, c Controller) Router {
	return e
}

func (e errRouter) Get(path string, controller, action string, r ActionController, mws ...MiddlewareBuilder) Router {
	return e
}

func (e errRouter) GetFunc(path string, controller, action string, r ActionFunc, mws ...MiddlewareBuilder) Router {
	return e
}

func (e errRouter) PostFunc(path string, controller, action string, r ActionFunc, mws ...MiddlewareBuilder) Router {
	return e
}

func (e errRouter) PutFunc(path string, controller, action string, r ActionFunc, mws ...MiddlewareBuilder) Router {
	return e
}

func (e errRouter) PatchFunc(path string, controller, action string, r ActionFunc, mws ...MiddlewareBuilder) Router {
	return e
}

func (e errRouter) DeleteFunc(path string, controller, action string, r ActionFunc, mws ...MiddlewareBuilder) Router {
	return e
}

func (e errRouter) Resource(r Resource, mws ...MiddlewareBuilder) Router {
	return e
}

func (e errRouter) ResourceAtPath(path string, r Resource, mws ...MiddlewareBuilder) Router {
	return e
}

func (e errRouter) Templates() *Templates {
	return nil
}

func (e errRouter) Config() Config {
	return Config{}
}

func (e errRouter) Handler() (http.Handler, error) {
	return nil, e.err
}

func (e errRouter) Run(ctx context.Context) error {
	return e.err
}
