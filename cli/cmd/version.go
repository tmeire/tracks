package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
)

var (
	// Version is the version of the application
	Version = "0.1.0"
)

func VersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the version number of Tracks",
		Long:  `All software has versions. This is Tracks'.`,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("Tracks version %s\n", Version)
		},
	}
}
