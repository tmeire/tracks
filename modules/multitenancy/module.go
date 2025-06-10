package multitenancy

import (
	"path/filepath"

	"github.com/tmeire/tracks"
)

// Register registers the multitenancy functionality with the router
func Register(r tracks.Router) tracks.Router {
	tenantDB := NewTenantRepository(r.Database(), filepath.Join(".", "data"))

	rn := &router{
		tenantDB,
		r,
		r.Clone().Views("./views/Tenants"),
	}

	return rn.Middleware(injectTenantDB(tenantDB))
}
