package cmd

import (
	"github.com/spf13/cobra"
	"github.com/tmeire/floral_crm/internal/tracks/cli/cmd/assets"
)

// AssetsCmd returns a cobra.Command for the assets command
func AssetsCmd() *cobra.Command {
	assetsCmd := &cobra.Command{
		Use:   "assets",
		Short: "Manage application assets",
		Long:  `Manage application assets, including compilation and hashing.`,
	}

	// Add subcommands
	assetsCmd.AddCommand(assets.CompileCmd())

	return assetsCmd
}