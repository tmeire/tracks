package tracks

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/tmeire/tracks/database"
)

type VersionConfig struct {
	Deprecated     bool
	SunsetDate     time.Time
	MigrationGuide string
}

func (r *router) Version(v string, config ...VersionConfig) Router {
	var conf VersionConfig
	if len(config) > 0 {
		conf = config[0]
	}

	return &versionRouter{
		router: r,
		prefix: "/" + v,
		name:   v,
		config: conf,
	}
}

type versionRouter struct {
	router *router
	prefix string
	name   string
	config VersionConfig
}

func (v *versionRouter) Clone() Router {
	return &versionRouter{
		router: v.router.Clone().(*router),
		prefix: v.prefix,
		name:   v.name,
		config: v.config,
	}
}

func (v *versionRouter) versionMiddleware(r Router) Middleware {
	return func(next http.Handler) (http.Handler, error) {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			w.Header().Set("API-Version", v.name)
			if v.config.Deprecated {
				w.Header().Set("Deprecation", "true")
				if !v.config.SunsetDate.IsZero() {
					w.Header().Set("Sunset", v.config.SunsetDate.Format(http.TimeFormat))
				}
				if v.config.MigrationGuide != "" {
					w.Header().Set("Link", fmt.Sprintf("<%s>; rel=\"migration\"", v.config.MigrationGuide))
				}
			}
			next.ServeHTTP(w, req)
		}), nil
	}
}

func (v *versionRouter) GetFunc(path, controller, action string, a ActionFunc, mws ...MiddlewareBuilder) Router {
	mws = append([]MiddlewareBuilder{v.versionMiddleware}, mws...)
	v.router.GetFunc(v.prefix+path, controller, action, a, mws...)
	return v
}

func (v *versionRouter) PostFunc(path, controller, action string, a ActionFunc, mws ...MiddlewareBuilder) Router {
	mws = append([]MiddlewareBuilder{v.versionMiddleware}, mws...)
	v.router.PostFunc(v.prefix+path, controller, action, a, mws...)
	return v
}

func (v *versionRouter) PutFunc(path, controller, action string, a ActionFunc, mws ...MiddlewareBuilder) Router {
	mws = append([]MiddlewareBuilder{v.versionMiddleware}, mws...)
	v.router.PutFunc(v.prefix+path, controller, action, a, mws...)
	return v
}

func (v *versionRouter) PatchFunc(path, controller, action string, a ActionFunc, mws ...MiddlewareBuilder) Router {
	mws = append([]MiddlewareBuilder{v.versionMiddleware}, mws...)
	v.router.PatchFunc(v.prefix+path, controller, action, a, mws...)
	return v
}

func (v *versionRouter) DeleteFunc(path, controller, action string, a ActionFunc, mws ...MiddlewareBuilder) Router {
	mws = append([]MiddlewareBuilder{v.versionMiddleware}, mws...)
	v.router.DeleteFunc(v.prefix+path, controller, action, a, mws...)
	return v
}

// Satisfy Router interface
func (v *versionRouter) Secure() bool { return v.router.Secure() }
func (v *versionRouter) BaseDomain() string { return v.router.BaseDomain() }
func (v *versionRouter) Database() database.Database { return v.router.Database() }
func (v *versionRouter) Module(m Module) Router { m(v); return v }
func (v *versionRouter) GlobalMiddleware(m Middleware) Router { v.router.GlobalMiddleware(m); return v }
func (v *versionRouter) DomainMiddleware() Middleware { return v.router.DomainMiddleware() }
func (v *versionRouter) DomainDatabase(config DomainDBConfig) Router { v.router.DomainDatabase(config); return v }
func (v *versionRouter) DomainScopedRepositories() Router { v.router.DomainScopedRepositories(); return v }
func (v *versionRouter) RequestMiddleware(m Middleware) Router { v.router.RequestMiddleware(m); return v }
func (v *versionRouter) Static(urlPath, dir string) Router { v.router.Static(urlPath, dir); return v }
func (v *versionRouter) StaticWithConfig(urlPath, dir string, config StaticConfig) Router { v.router.StaticWithConfig(urlPath, dir, config); return v }
func (v *versionRouter) Content(path, dir string, config ContentConfig) Router { v.router.Content(path, dir, config); return v }
func (v *versionRouter) LogHostEntries() Router { v.router.LogHostEntries(); return v }
func (v *versionRouter) LogHostEntriesWithMessage(m string) Router { v.router.LogHostEntriesWithMessage(m); return v }
func (v *versionRouter) HealthCheck(p string, c ...HealthConfig) Router { v.router.HealthCheck(p, c...); return v }
func (v *versionRouter) Version(vv string, c ...VersionConfig) Router { return v.router.Version(vv, c...) }
func (v *versionRouter) VersionFromHeader(h, val string, r Router) Router { v.router.VersionFromHeader(h, val, r); return v }
func (v *versionRouter) VersionFromQuery(p, val string, r Router) Router { v.router.VersionFromQuery(p, val, r); return v }
func (v *versionRouter) RateLimit(c RateLimitConfig) Router { v.router.RateLimit(c); return v }
func (v *versionRouter) WebSocket(path string, h WebSocketHandler, mws ...MiddlewareBuilder) Router {
	v.router.WebSocket(v.prefix+path, h, mws...)
	return v
}
func (v *versionRouter) CSRFProtection(c CSRFConfig) Router { v.router.CSRFProtection(c); return v }
func (v *versionRouter) Cache() Cache { return v.router.Cache() }
func (v *versionRouter) WithCache(c Cache) Router { v.router.WithCache(c); return v }
func (v *versionRouter) Queue() Queue { return v.router.Queue() }
func (v *versionRouter) Func(name string, fn any) Router { v.router.Func(name, fn); return v }
func (v *versionRouter) Views(path string) Router { v.router.Views(path); return v }
func (v *versionRouter) Page(path, view string) Router { v.router.Page(v.prefix+path, view); return v }
func (v *versionRouter) Redirect(o, d string) Router { v.router.Redirect(o, d); return v }
func (v *versionRouter) Serve(a Action) Router { a.Path = v.prefix + a.Path; v.router.Serve(a); return v }
func (v *versionRouter) Controller(c Controller) Router { return v.router.ControllerAtPath(v.prefix, c) }
func (v *versionRouter) ControllerAtPath(path string, c Controller) Router { return v.router.ControllerAtPath(v.prefix+path, c) }
func (v *versionRouter) Get(path, c, a string, ac ActionController, mws ...MiddlewareBuilder) Router {
	mws = append([]MiddlewareBuilder{v.versionMiddleware}, mws...)
	v.router.Get(v.prefix+path, c, a, ac, mws...)
	return v
}
func (v *versionRouter) Resource(r Resource, mws ...MiddlewareBuilder) Router {
	return v.ResourceAtPath("/", r, mws...)
}
func (v *versionRouter) ResourceAtPath(path string, rs Resource, mws ...MiddlewareBuilder) Router {
	mws = append([]MiddlewareBuilder{v.versionMiddleware}, mws...)
	return v.router.ResourceAtPath(v.prefix+path, rs, mws...)
}
func (v *versionRouter) Templates() *Templates { return v.router.Templates() }
func (v *versionRouter) Config() Config { return v.router.Config() }
func (v *versionRouter) Handler() (http.Handler, error) { return v.router.Handler() }
func (v *versionRouter) Run(ctx context.Context) error { return v.router.Run(ctx) }
