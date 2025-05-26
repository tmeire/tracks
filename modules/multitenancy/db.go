package multitenancy

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"sync"

	"github.com/tmeire/tracks/database"
	"github.com/tmeire/tracks/database/sqlite"
)

func injectTenantDB(tenants *TenantRepository) func(handler http.Handler) http.Handler {
	return func(handler http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			ctx := req.Context()
			tenant := ctx.Value(tenantContextKey).(*Tenant)

			db, err := tenants.GetTenantDB(req.Context(), tenant.ID)
			if err != nil {
				http.Error(w, "Failed to connect to database", http.StatusInternalServerError)
				return
			}

			// Add the tenant database to the request context
			req = req.WithContext(database.WithDB(req.Context(), db))

			handler.ServeHTTP(w, req)
		})
	}
}

// TenantRepository manages database connections for tenants
type TenantRepository struct {
	centralDB     database.Database
	tenantDBs     map[int]database.Database
	tenantsMutex  sync.RWMutex
	storageDir    string
	migrationsDir string
}

// NewTenantRepository creates a new TenantRepository instance
func NewTenantRepository(centralDB database.Database, baseDir string) *TenantRepository {
	return NewTenantRepositoryWithMigrations(centralDB, baseDir, baseDir)
}

// NewTenantRepositoryWithMigrations creates a new TenantRepository instance
func NewTenantRepositoryWithMigrations(centralDB database.Database, baseDir string, migrationsDir string) *TenantRepository {
	return &TenantRepository{
		centralDB:     centralDB,
		tenantDBs:     make(map[int]database.Database),
		storageDir:    baseDir,
		migrationsDir: migrationsDir,
	}
}

// GetCentralDB returns the central database connection
func (t *TenantRepository) GetCentralDB() database.Database {
	return t.centralDB
}

// GetTenantDB returns a database connection for the specified tenant
func (t *TenantRepository) GetTenantDB(ctx context.Context, tenantID int) (database.Database, error) {
	t.tenantsMutex.RLock()
	tenantDB, exists := t.tenantDBs[tenantID]
	t.tenantsMutex.RUnlock()

	if exists {
		return tenantDB, nil
	}

	// If the tenant database connection doesn't exist, create it
	return t.createTenantDB(ctx, tenantID)
}

// createTenantDB creates a new database connection for the specified tenant
func (t *TenantRepository) createTenantDB(ctx context.Context, tenantID int) (database.Database, error) {
	t.tenantsMutex.Lock()
	defer t.tenantsMutex.Unlock()

	// Check again in case another goroutine created the connection while we were waiting
	if tenantDB, exists := t.tenantDBs[tenantID]; exists {
		return tenantDB, nil
	}

	// GetFunc tenant information from the central database
	tenantRepo := database.NewRepository[*Tenant](t.centralDB)
	tenant, err := tenantRepo.FindByID(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to find tenant: %w", err)
	}

	// Create the database connection
	tenantDB, err := sqlite.New(tenant.DBPath)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to tenant database: %w", err)
	}

	err = database.MigrateUp(ctx, tenantDB, database.TenantDatabase)
	if err != nil {
		log.Fatalf("failed to apply migrations: %v", err)
	}

	// Store the connection for future use
	t.tenantDBs[tenantID] = tenantDB

	return tenantDB, nil
}

// CreateTenant creates a new tenant with its own database
func (t *TenantRepository) CreateTenant(ctx context.Context, name, subdomain string) (*Tenant, error) {
	// Create a new tenant record
	tenant := &Tenant{
		Name:      name,
		Subdomain: subdomain,
		DBPath:    filepath.Join(t.storageDir, "tenants", subdomain, "tenant.sqlite"),
	}

	// Save the tenant to the central database
	tenantRepo := database.NewRepository[*Tenant](t.centralDB)
	tenant, err := tenantRepo.Create(ctx, tenant)
	if err != nil {
		return nil, fmt.Errorf("failed to create tenant: %w", err)
	}

	// Create the tenant database
	tenantDB, err := sqlite.New(tenant.DBPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create tenant database: %w", err)
	}

	// Apply migrations to the tenant database
	err = database.MigrateUpDir(ctx, tenantDB, database.TenantDatabase, filepath.Join(t.migrationsDir, "tenant"))
	if err != nil {
		return nil, fmt.Errorf("failed to apply migrations to tenant database: %w", err)
	}

	// Store the connection for future use
	t.tenantsMutex.Lock()
	t.tenantDBs[tenant.ID] = tenantDB
	t.tenantsMutex.Unlock()

	return tenant, nil
}

// Close closes all database connections
func (t *TenantRepository) Close() error {
	t.tenantsMutex.Lock()
	defer t.tenantsMutex.Unlock()

	// Close all tenant database connections
	for _, tenantDB := range t.tenantDBs {
		if err := tenantDB.Close(); err != nil {
			return err
		}
	}

	// Close the central database connection
	return t.centralDB.Close()
}

// GetTenantBySubdomain returns a tenant by its subdomain
func (t *TenantRepository) GetTenantBySubdomain(ctx context.Context, subdomain string) (*Tenant, error) {
	tenantRepo := database.NewRepository[*Tenant](t.centralDB)
	tenants, err := tenantRepo.FindBy(ctx, map[string]any{"subdomain": subdomain})
	if err != nil {
		return nil, fmt.Errorf("failed to find tenant: %w", err)
	}

	if len(tenants) == 0 {
		return nil, fmt.Errorf("tenant not found: %s", subdomain)
	}

	return tenants[0], nil
}
