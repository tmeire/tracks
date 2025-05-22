package multitenancy

import (
	"context"
	"fmt"
	"github.com/tmeire/floral_crm/internal/tracks"
	"github.com/tmeire/floral_crm/internal/tracks/database"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// contextKey is a type for context keys specific to the multitenancy package
type contextKey string

const (
	// tenantContextKey is the context key for the current tenant
	tenantContextKey contextKey = "tenant"
)

type router struct {
	tenantDB *TenantRepository

	root       tracks.Router
	subdomains tracks.Router
}

func (r *router) Clone() tracks.Router {
	return &router{
		tenantDB:   r.tenantDB,
		root:       r.root,
		subdomains: r.subdomains,
	}
}

func (r *router) Secure() bool {
	return r.root.Secure()
}

func (r *router) BaseDomain() string {
	return r.root.BaseDomain()
}

func (r *router) Port() int {
	return r.root.Port()
}

func (r *router) Database() database.Database {
	return r.root.Database()
}

func (r *router) Module(m tracks.Module) tracks.Router {
	r.subdomains = r.subdomains.Module(m)
	return r
}

func (r *router) Middleware(m tracks.Middleware) tracks.Router {
	r.subdomains = r.subdomains.Middleware(m)
	return r
}

func (r *router) Func(name string, fn any) tracks.Router {
	r.subdomains = r.subdomains.Func(name, fn)
	return r
}

func (r *router) Views(path string) tracks.Router {
	r.subdomains = r.subdomains.Views(path)
	return r
}

func (r *router) Page(path string, view string) tracks.Router {
	r.subdomains = r.subdomains.Page(path, view)
	return r
}

func (r *router) Redirect(origin string, destination string) tracks.Router {
	r.subdomains = r.subdomains.Redirect(origin, destination)
	return r
}

func (r *router) Get(path string, controller, action string, c tracks.Controller) tracks.Router {
	r.subdomains = r.subdomains.Get(path, controller, action, c)
	return r
}

func (r *router) GetFunc(path string, controller, action string, a tracks.Action) tracks.Router {
	r.subdomains = r.subdomains.GetFunc(path, controller, action, a)
	return r
}

func (r *router) PostFunc(path string, controller, action string, a tracks.Action) tracks.Router {
	r.subdomains = r.subdomains.PostFunc(path, controller, action, a)
	return r
}

func (r *router) PutFunc(path string, controller, action string, a tracks.Action) tracks.Router {
	r.subdomains = r.subdomains.PutFunc(path, controller, action, a)
	return r
}

func (r *router) PatchFunc(path string, controller, action string, a tracks.Action) tracks.Router {
	r.subdomains = r.subdomains.PatchFunc(path, controller, action, a)
	return r
}

func (r *router) DeleteFunc(path string, controller, action string, a tracks.Action) tracks.Router {
	r.subdomains = r.subdomains.DeleteFunc(path, controller, action, a)
	return r
}

func (r *router) Resource(rs tracks.Resource) tracks.Router {
	r.subdomains = r.subdomains.Resource(rs)
	return r
}

func (r *router) Run() {
	tracks.Serve(r, r.root.Port())
}

func (r *router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	subdomain := extractSubdomain(req.Host)
	if subdomain == "" {
		if req.Referer() != "" {
			r, err := url.Parse(req.Referer())
			if err == nil && strings.HasSuffix(r.Host, req.Host) {
				// Make sure we can redirect from the base domain to the subdomain
				w.Header().Set("Access-Control-Allow-Origin", r.Scheme+"://"+r.Host)
			}
		}

		r.root.ServeHTTP(w, req)
		return
	}

	// Find the tenant by subdomain
	tenant, err := r.tenantDB.GetTenantBySubdomain(req.Context(), subdomain)
	if err != nil {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	// Add the tenant and tenant database to the request context
	ctx := context.WithValue(req.Context(), tenantContextKey, tenant)

	// Call the next handler with the updated context
	r.subdomains.ServeHTTP(w, req.WithContext(ctx))
}

func (r *router) base() string {
	scheme := "http"
	if r.root.Secure() {
		scheme = "https"
	}

	host := fmt.Sprintf("%s://%s", scheme, r.root.BaseDomain())
	if r.root.Port() != 0 {
		host += ":" + strconv.Itoa(r.root.Port())
	}
	return host
}

// extractSubdomain extracts the subdomain from the host
func extractSubdomain(host string) string {
	host, _, err := net.SplitHostPort(host)
	if err != nil {
		return ""
	}

	// Split the host by dots
	parts := strings.Split(host, ".")

	// If we have at least 3 parts (subdomain.domain.tld), the first part is the subdomain
	if len(parts) >= 3 {
		return parts[0]
	}

	// No subdomain found
	return ""
}
