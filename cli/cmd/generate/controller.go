package generate

import (
	"fmt"
	"github.com/spf13/cobra"
	"github.com/tmeire/floral_crm/internal/tracks/cli/project"
	"os"
)

// ControllerCmd returns a cobra.Command for the generate controller command
func ControllerCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "controller [method] [path]",
		Short: "Generate a controller with an action",
		Long: `Generate a controller with an action.

This command:
1. Extracts the controller and action name from the path
2. Creates a controller file if it doesn't exist
3. Creates an action function in the controller
4. Creates a view template file
5. Updates main.go to register the action`,
		Args: cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			p, err := project.Load()
			if err != nil {
				fmt.Printf("Error loading project: %v\n", err)
				os.Exit(1)
			}
			p.AddPage(args[0], args[1])
		},
	}

	return cmd
}
