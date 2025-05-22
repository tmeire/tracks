package main

import (
	"fmt"
	"github.com/tmeire/floral_crm/internal/tracks/cli/cmd"
	"os"

	"github.com/spf13/cobra"
)

func main() {
	// Used for flags
	rootCmd := &cobra.Command{
		Use:   "tracks",
		Short: "Tracks CLI",
		Long: `A command line interface for the Tracks application.
This CLI provides various utilities and commands for managing the Tracks-based systems.`,
	}
	rootCmd.AddCommand(cmd.VersionCmd())
	rootCmd.AddCommand(cmd.AssetsCmd())
	rootCmd.AddCommand(cmd.GenerateCmd())
	rootCmd.AddCommand(cmd.DbCmd())
	rootCmd.AddCommand(cmd.TenantCmd())

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
