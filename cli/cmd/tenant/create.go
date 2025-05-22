package tenant

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/tmeire/tracks/database/sqlite"
	"github.com/tmeire/tracks/modules/multitenancy"
)

// blacklistedSubdomains is a list of common subdomain names that can be easily abused
var blacklistedSubdomains = []string{
	"admin",
	"api",
	"app",
	"blog",
	"dashboard",
	"dev",
	"docs",
	"help",
	"mail",
	"secure",
	"shop",
	"staging",
	"status",
	"support",
	"test",
	"www",
}

// CreateCmd returns a cobra.Command for creating a new tenant
func CreateCmd() *cobra.Command {
	createCmd := &cobra.Command{
		Use:   "create [name] [subdomain]",
		Short: "Create a new tenant",
		Long:  `Create a new tenant with the specified name and subdomain.`,
		Args:  cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			name := args[0]
			subdomain := strings.ToLower(args[1])

			// Check if the subdomain is in the blacklist
			for _, blacklisted := range blacklistedSubdomains {
				if subdomain == blacklisted {
					fmt.Printf("Error: Subdomain '%s' is not allowed as it is a reserved name.\n", subdomain)
					return
				}
			}

			// Open a connection to the central database
			centralDB, err := sqlite.New(filepath.Join(".", "data", "tracks.sqlite"))
			if err != nil {
				fmt.Printf("Failed to connect to database: %v\n", err)
				return
			}
			defer centralDB.Close()

			// Create a TenantRepository instance
			tenantDB := multitenancy.NewTenantRepositoryWithMigrations(centralDB, filepath.Join(".", "data"), filepath.Join(".", "migrations"))

			// Create the tenant
			tenant, err := tenantDB.CreateTenant(context.Background(), name, subdomain)
			if err != nil {
				fmt.Printf("Failed to create tenant: %v\n", err)
				return
			}

			fmt.Printf("Tenant created successfully:\n")
			fmt.Printf("  ID: %d\n", tenant.ID)
			fmt.Printf("  Name: %s\n", tenant.Name)
			fmt.Printf("  Subdomain: %s\n", tenant.Subdomain)
			fmt.Printf("  Database: %s\n", tenant.DBPath)
		},
	}

	return createCmd
}
