package cmd

import (
	"github.com/spf13/cobra"
	"github.com/tmeire/floral_crm/internal/tracks/cli/cmd/generate"
)

// GenerateCmd returns a cobra.Command for the generate command
func GenerateCmd() *cobra.Command {
	generateCmd := &cobra.Command{
		Use:   "generate",
		Short: "Generate code for the application",
		Long:  `Generate code for the application, including controllers, actions, resources, and views.`,
	}

	// Add subcommands
	generateCmd.AddCommand(generate.ControllerCmd())
	generateCmd.AddCommand(generate.ResourceCmd())

	return generateCmd
}
