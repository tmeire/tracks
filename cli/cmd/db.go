package cmd

import (
	"github.com/spf13/cobra"
	"github.com/tmeire/tracks/cli/cmd/db"
)

// DbCmd returns a cobra.Command for the db command
func DbCmd() *cobra.Command {
	dbCmd := &cobra.Command{
		Use:   "db",
		Short: "Manage database tasks",
		Long:  `Manage database tasks, including migrations.`,
	}

	// Add subcommands
	dbCmd.AddCommand(db.MigrateCmd())

	return dbCmd
}
