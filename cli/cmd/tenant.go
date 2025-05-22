package cmd

import (
	"github.com/spf13/cobra"
	"github.com/tmeire/floral_crm/internal/tracks/cli/cmd/tenant"
)

// TenantCmd returns a cobra.Command for tenant management
func TenantCmd() *cobra.Command {
	tenantCmd := &cobra.Command{
		Use:   "tenant",
		Short: "Manage tenants",
		Long:  `Manage tenants in the multitenancy system.`,
	}

	// Add subcommands
	tenantCmd.AddCommand(tenant.CreateCmd())

	return tenantCmd
}