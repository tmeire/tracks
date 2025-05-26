package cmd

import (
	"github.com/spf13/cobra"
	"github.com/tmeire/tracks/cli/cmd/initialize"
)

// InitCmd returns a cobra.Command for the init command
func InitCmd() *cobra.Command {
	initCmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize a new Tracks application",
		Long:  `Initialize a new Tracks application by setting up the necessary files and directories.`,
		Run:   initialize.Run,
	}

	return initCmd
}
