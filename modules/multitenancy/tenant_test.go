package multitenancy_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/tmeire/tracks/database"
	"github.com/tmeire/tracks/database/sqlite"
	"github.com/tmeire/tracks/modules/multitenancy"
)

func TestTenantDB(t *testing.T) {
	ctx := t.Context()

	// Create a temporary directory for the test
	tempDir, err := os.MkdirTemp("", "tenant_test")
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

	// Apply migrations to the central database
	err = database.MigrateUpDir(ctx, centralDB, database.CentralDatabase, "./testdata/migrations/central")
	if err != nil {
		t.Fatalf("Failed to apply migrations to central database: %v", err)
	}

	// Create a tenant database manager
	tenantDB := multitenancy.NewTenantRepositoryWithMigrations(centralDB, tempDir, "./testdata/migrations/")
	defer tenantDB.Close()

	// Create a tenant
	tenant, err := tenantDB.CreateTenant(ctx, "Test Tenant", "test")
	if err != nil {
		t.Fatalf("Failed to create tenant: %v", err)
	}

	// Verify the tenant was created
	if tenant.ID == 0 {
		t.Fatalf("Tenant ID should not be 0")
	}
	if tenant.Name != "Test Tenant" {
		t.Fatalf("Tenant name should be 'Test Tenant', got '%s'", tenant.Name)
	}
	if tenant.Subdomain != "test" {
		t.Fatalf("Tenant subdomain should be 'test', got '%s'", tenant.Subdomain)
	}

	// GetFunc the tenant by subdomain
	tenant2, err := tenantDB.GetTenantBySubdomain(ctx, "test")
	if err != nil {
		t.Fatalf("Failed to get tenant by subdomain: %v", err)
	}
	if tenant2.ID != tenant.ID {
		t.Fatalf("Tenant IDs should match, got %d and %d", tenant.ID, tenant2.ID)
	}

	// GetFunc the tenant database
	tenantDatabase, err := tenantDB.GetTenantDB(ctx, 1)
	if err != nil {
		t.Fatalf("Failed to get tenant database: %v", err)
	}

	// Verify the tenant database exists
	if _, err := os.Stat(tenant.DBPath); os.IsNotExist(err) {
		t.Fatalf("Tenant database file should exist at %s", tenant.DBPath)
	}

	// Verify we can execute a query on the tenant database
	_, err = tenantDatabase.ExecContext(ctx, "CREATE TABLE test (id INTEGER PRIMARY KEY, name TEXT)")
	if err != nil {
		t.Fatalf("Failed to create table in tenant database: %v", err)
	}

	_, err = tenantDatabase.ExecContext(ctx, "INSERT INTO test (name) VALUES (?)", "test")
	if err != nil {
		t.Fatalf("Failed to insert data into tenant database: %v", err)
	}

	rows, err := tenantDatabase.QueryContext(ctx, "SELECT name FROM test")
	if err != nil {
		t.Fatalf("Failed to query tenant database: %v", err)
	}
	defer rows.Close()

	if !rows.Next() {
		t.Fatalf("Expected a row in the test table")
	}

	var name string
	err = rows.Scan(&name)
	if err != nil {
		t.Fatalf("Failed to scan row: %v", err)
	}

	if name != "test" {
		t.Fatalf("Expected name to be 'test', got '%s'", name)
	}
}
