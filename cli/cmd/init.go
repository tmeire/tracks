package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/tmeire/tracks/cli/project"
)

// InitCmd returns a cobra.Command for the init command
func InitCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init [module_name]",
		Short: "Initialize a new Tracks application",
		Long: `Initialize a new Tracks application with the basic structure.
This command:
1. Creates a new directory for the project
2. Initializes a Git repository
3. Creates a Go module with the specified module name
4. Sets up the basic directory structure
5. Creates initial configuration files
6. Adds basic controllers, models, and views`,
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			moduleName := args[0]
			// Extract project name from the module name (last part of the path)
			projectName := filepath.Base(moduleName)

			fmt.Printf("Initializing new Tracks application: %s (module: %s)\n", projectName, moduleName)

			if err := project.Init(moduleName, projectName); err != nil {
				fmt.Printf("Error initializing project: %v\n", err)
				os.Exit(1)
			}

			fmt.Printf("Successfully initialized %s\n", projectName)
		},
	}

	return cmd
}
