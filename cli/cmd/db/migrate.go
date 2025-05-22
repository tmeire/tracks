package db

import (
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/tmeire/tracks/database"
)

// MigrateCmd returns a cobra.Command for the db command
func MigrateCmd() *cobra.Command {
	migrateCmd := &cobra.Command{
		Use:   "db",
		Short: "Manage database migrations",
		Long:  `Manage database migrations using Goose.`,
	}

	// Add subcommands
	migrateCmd.AddCommand(UpCmd())
	migrateCmd.AddCommand(DownCmd())
	migrateCmd.AddCommand(StatusCmd())

	return migrateCmd
}

// UpCmd returns a cobra.Command for the db up command
func UpCmd() *cobra.Command {
	var dbType string
	var dbPath string
	upCmd := &cobra.Command{
		Use:   "up",
		Short: "Apply pending migrations",
		Long:  `Apply all pending database migrations or migrations from a specific source.`,
		Run: func(cmd *cobra.Command, args []string) {
			var databaseType database.DatabaseType
			switch dbType {
			case "central":
				databaseType = database.CentralDatabase
			case "tenant":
				databaseType = database.TenantDatabase
			default:
				cmd.PrintErrf("Invalid database type: %s. Must be 'central' or 'tenant'.\n", dbType)
				return
			}
			database.RunGooseMigration(cmd.Context(), "up", databaseType, dbPath)
		},
	}
	upCmd.Flags().StringVar(&dbType, "type", "central", "Database type (central or tenant)")
	upCmd.Flags().StringVar(&dbPath, "db", filepath.Join(".", "data", "tracks.sqlite"), "Database path")
	return upCmd
}

// DownCmd returns a cobra.Command for the db down command
func DownCmd() *cobra.Command {
	var dbType string
	var dbPath string
	downCmd := &cobra.Command{
		Use:   "down",
		Short: "Revert migrations",
		Long:  `Revert the most recent migration.`,
		Run: func(cmd *cobra.Command, args []string) {
			var databaseType database.DatabaseType
			switch dbType {
			case "central":
				databaseType = database.CentralDatabase
			case "tenant":
				databaseType = database.TenantDatabase
			default:
				cmd.PrintErrf("Invalid database type: %s. Must be 'central' or 'tenant'.\n", dbType)
				return
			}
			database.RunGooseMigration(cmd.Context(), "down", databaseType, dbPath)
		},
	}
	downCmd.Flags().StringVar(&dbType, "type", "central", "Database type (central or tenant)")
	downCmd.Flags().StringVar(&dbPath, "db", filepath.Join(".", "data", "tracks.sqlite"), "Database path")
	return downCmd
}

// StatusCmd returns a cobra.Command for the db status command
func StatusCmd() *cobra.Command {
	var dbType string
	var dbPath string
	statusCmd := &cobra.Command{
		Use:   "status",
		Short: "Show migration status",
		Long:  `Show the status of all migrations.`,
		Run: func(cmd *cobra.Command, args []string) {
			var databaseType database.DatabaseType
			switch dbType {
			case "central":
				databaseType = database.CentralDatabase
			case "tenant":
				databaseType = database.TenantDatabase
			default:
				cmd.PrintErrf("Invalid database type: %s. Must be 'central' or 'tenant'.\n", dbType)
				return
			}
			database.RunGooseMigration(cmd.Context(), "status", databaseType, dbPath)
		},
	}
	statusCmd.Flags().StringVar(&dbType, "type", "central", "Database type (central or tenant)")
	statusCmd.Flags().StringVar(&dbPath, "db", filepath.Join(".", "data", "tracks.sqlite"), "Database path")
	return statusCmd
}
