package multitenancy

import (
	"net"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/tmeire/tracks/database"

	"github.com/tmeire/tracks"
)

// Register registers the multitenancy functionality with the router
func Register(r tracks.Router) tracks.Router {
	tenantDB := NewTenantRepository(r.Database(), filepath.Join(".", "data"))

	rn := r.Clone().Views("./views/tenants")

	r.GlobalMiddleware(func(next http.Handler) (http.Handler, error) {
		h, err := rn.Handler()
		if err != nil {
			return nil, err
		}
		return &splitter{
			tenantDB:   tenantDB,
			root:       next,
			subdomains: h,
		}, nil
	})

	return rn
}

type splitter struct {
	tenantDB *TenantRepository

	root       http.Handler
	subdomains http.Handler
}

func (s *splitter) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	subdomain := extractSubdomain(req.Host)
	if subdomain == "" {
		if req.Referer() != "" {
			r, err := url.Parse(req.Referer())
			if err == nil && strings.HasSuffix(r.Host, req.Host) {
				// Make sure we can redirect from the base domain to the subdomain
				w.Header().Set("Access-Control-Allow-Origin", r.Scheme+"://"+r.Host)
			}
		}

		s.root.ServeHTTP(w, req)
		return
	}

	// Find the tenant by subdomain, add the central db to the context
	tenant, err := s.tenantDB.GetTenantBySubdomain(req.Context(), subdomain)
	if err != nil {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	req = tracks.AddViewVar(req, "tenant", tenant)

	// Get a database and add it to the context
	db, err := s.tenantDB.GetTenantDB(req.Context(), tenant.ID)
	if err != nil {
		http.Error(w, "Failed to connect to database", http.StatusInternalServerError)
		return
	}

	ctx := database.WithDB(req.Context(), db)

	// Call the next handler with the updated context
	s.subdomains.ServeHTTP(w, req.WithContext(ctx))
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
