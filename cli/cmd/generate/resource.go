package generate

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/tmeire/tracks/cli/project"
)

// ResourceCmd returns a cobra.Command for the generate resource command
func ResourceCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "resource [name]",
		Short: "Generate a resource",
		Long: `Generate a resource with controller, model, and views.

This command:
1. Creates a controller file that implements the Resource interface
2. Creates view files for each of the resource actions
3. Creates a model file for the resource
4. Updates main.go to register the resource`,
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			p, err := project.Load()
			if err != nil {
				fmt.Printf("Error loading project: %v\n", err)
				os.Exit(1)
			}

			resourceName := strings.ToLower(args[0])

			fmt.Printf("Generating resource %s\n", resourceName)

			packageName, err := p.GetPackageName()
			if err != nil {
				fmt.Printf("Error reading package name: %v\n", err)
				os.Exit(1)
			}

			// Create the resource
			err = p.AddResource(resourceName, packageName)
			if err != nil {
				fmt.Printf("Error creating resource: %v\n", err)
				os.Exit(1)
			}

			fmt.Println("Resource generated successfully!")
		},
	}

	return cmd
}
