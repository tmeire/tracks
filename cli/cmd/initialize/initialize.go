package initialize

import (
	"fmt"
	"github.com/spf13/cobra"
	"github.com/tmeire/tracks/cli/project"
	"os"
)

// Run is the entry point for the init command
func Run(cmd *cobra.Command, args []string) {
	// Check if go.mod exists in the current directory
	if _, err := os.Stat("go.mod"); os.IsNotExist(err) {
		fmt.Println("Error: go.mod file not found in the current directory.")
		fmt.Println("Please run 'go mod init <module-name>' to initialize a Go module first.")
		os.Exit(1)
	}

	p, err := project.Load()
	if err != nil {
		fmt.Printf("Error loading project: %v\n", err)
		os.Exit(1)
	}

	err = p.Initialize()
	if err != nil {
		fmt.Printf("Error initializing project: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Initialization complete! Your Tracks application is ready for development.")
}
