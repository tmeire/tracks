package multitenancy

import (
	"context"
	"embed"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/pressly/goose/v3"
	"github.com/tmeire/tracks/database"

	"github.com/tmeire/tracks"
)

//go:embed migrations
var migrations embed.FS

// Register registers the multitenancy functionality with the router
func Register(r tracks.Router) tracks.Router {
	// Apply migrations for this module explicitly (lives outside default path)
	goose.SetBaseFS(migrations)
	database.MigrateUp(context.Background(), r.Database(), database.CentralDatabase)
	goose.SetBaseFS(nil)

	rn := r.Clone().Views("./views/tenants")

	r.GlobalMiddleware(func(next http.Handler) (http.Handler, error) {
		tenantDB := NewTenantRepositoryWithMigrations(r.Database(), filepath.Join(".", "data"), filepath.Join(".", "migrations"))

		h, err := rn.Handler()
		if err != nil {
			return nil, err
		}
		return &splitter{
			tenantDB:         tenantDB,
			root:             next,
			subdomains:       h,
			subdomainsRouter: rn,
			baseDomain:       r.BaseDomain(),
			secure:           r.Secure(),
		}, nil
	})

	return rn
}

type splitter struct {
	tenantDB *TenantRepository

	root             http.Handler
	subdomains       http.Handler
	subdomainsRouter tracks.Router
	baseDomain       string
	secure           bool
}

func (s *splitter) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	host := req.Host
	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	}

	subdomain := extractSubdomain(req.Host, s.baseDomain)
	fmt.Printf("DEBUG: Splitter: Request %s %s (Host: %s, Subdomain: %s)\n", req.Method, req.URL.Path, req.Host, subdomain)

	ctx := req.Context()
	ctx = WithCentralDB(ctx, s.tenantDB.GetCentralDB())

	var tenant *Tenant
	var err error

	if subdomain != "" {
		tenant, err = s.tenantDB.GetTenantBySubdomain(ctx, subdomain)
	} else {
		// It might be a custom domain, or the root domain
		tenant, err = s.tenantDB.GetTenantByCustomDomain(ctx, host)
	}

	if err != nil || tenant == nil {
		if subdomain == "" {
			if req.Referer() != "" {
				r, err := url.Parse(req.Referer())
				if err == nil && strings.HasSuffix(r.Host, req.Host) {
					// Make sure we can redirect from the base domain to the subdomain
					w.Header().Set("Access-Control-Allow-Origin", r.Scheme+"://"+r.Host)
				}
			}

			s.root.ServeHTTP(w, req.WithContext(ctx))
			return
		}
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	if !tenant.Active {
		scheme := "http"
		if s.secure {
			scheme = "https"
		}

		// Redirect to the root domain's "welcome" or landing page
		// We can use a query parameter to indicate which tenant was being accessed
		target := fmt.Sprintf("%s://%s/pending-activation", scheme, s.baseDomain)
		http.Redirect(w, req, target, http.StatusSeeOther)
		return
	}

	req = tracks.AddViewVar(req, "tenant", tenant)
	req = tracks.AddViewVar(req, "Subdomain", subdomain)

	// Get a database and add it to the context
	db, err := s.tenantDB.GetTenantDB(ctx, tenant.ID)
	if err != nil {
		slog.Error("Failed to connect to tenant database", "tenantID", tenant.ID, "dbPath", tenant.DBPath, "error", err)
		http.Error(w, "Failed to connect to database", http.StatusInternalServerError)
		return
	}

	ctx = database.WithDB(ctx, db)
	ctx = WithContext(ctx, tenant.ID)
	ctx = context.WithValue(ctx, subdomainKey{}, subdomain)

	slog.Info("Successfully connected to tenant database", "tenantID", tenant.ID, "subdomain", subdomain)

	// Call the next handler with the updated context
	s.subdomains.ServeHTTP(w, req.WithContext(ctx))
}

type subdomainKey struct{}

// SubdomainFromContext returns the subdomain stored in the context, or an empty string if not found.
func SubdomainFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if subdomain, ok := ctx.Value(subdomainKey{}).(string); ok {
		return subdomain
	}
	return ""
}

// extractSubdomain extracts the subdomain from the host
func extractSubdomain(host, baseDomain string) string {
	h, _, err := net.SplitHostPort(host)
	if err != nil {
		h = host
	}

	bh, _, err := net.SplitHostPort(baseDomain)
	if err != nil {
		bh = baseDomain
	}

	if h == bh {
		return ""
	}

	if strings.HasSuffix(h, "."+bh) {
		return strings.TrimSuffix(h, "."+bh)
	}

	return ""
}
