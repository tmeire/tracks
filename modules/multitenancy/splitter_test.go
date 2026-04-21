package multitenancy

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tmeire/tracks"
	"github.com/tmeire/tracks/database"
	"github.com/tmeire/tracks/database/sqlite"
)

func TestSplitter_ViewVars(t *testing.T) {
	ctx := context.Background()
	
	// Create a temporary directory for the test
	tempDir, err := os.MkdirTemp("", "splitter_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a central database
	centralDBPath := filepath.Join(tempDir, "central.sqlite")
	centralDB, err := sqlite.New(centralDBPath)
	if err != nil {
		t.Fatalf("Failed to create central database: %v", err)
	}
	defer centralDB.Close()

	// Initialize the tenant repository
	tenantDB := NewTenantRepositoryWithMigrations(centralDB, tempDir, "./testdata/migrations/")
	defer tenantDB.Close()

	// Apply migrations to the central database
	err = database.MigrateUpDir(ctx, centralDB, database.CentralDatabase, "./testdata/migrations/central")
	if err != nil {
		t.Fatalf("Failed to apply migrations: %v", err)
	}

	// Create a tenant
	tenant, err := tenantDB.CreateTenant(ctx, "Test Tenant", "test", true)
	if err != nil {
		t.Fatalf("Failed to create tenant: %v", err)
	}

	// Create a mock subdomain handler that checks for ViewVars
	subHandlerCalled := false
	subHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		subHandlerCalled = true
		
		vTenant := tracks.ViewVar(r, "tenant")
		vSubdomain := tracks.ViewVar(r, "Subdomain")
		
		assert.NotNil(t, vTenant, "tenant ViewVar should be set")
		assert.Equal(t, "test", vSubdomain, "Subdomain ViewVar should be set to 'test'")
		
		if ten, ok := vTenant.(*Tenant); ok {
			assert.Equal(t, tenant.ID, ten.ID)
		} else {
			assert.Fail(t, "tenant ViewVar is not a *Tenant")
		}
	})

	s := &splitter{
		tenantDB:         tenantDB,
		root:             http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}),
		subdomains:       subHandler,
		baseDomain:       "floralynx.com",
		secure:           false,
	}

	// Test a subdomain request
	req := httptest.NewRequest("GET", "http://test.floralynx.com/", nil)
	w := httptest.NewRecorder()
	
	s.ServeHTTP(w, req)
	
	assert.True(t, subHandlerCalled, "subdomain handler should have been called")
}
