package cli

import (
	"github.com/spf13/cobra"
	"github.com/tmeire/tracks/modules/featureflags/cli/featureflags"
)

// FeatureFlagsCmd returns a cobra.Command for the featureflags command
func FeatureFlagsCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "featureflags",
		Short: "Manage feature flags",
		Long:  `Manage feature flags: list, get, set, unset overrides in the central database`,
	}

	root.AddCommand(featureflags.ListCmd())
	root.AddCommand(featureflags.GetCmd())
	root.AddCommand(featureflags.SetCmd())
	root.AddCommand(featureflags.UnsetCmd())

	return root
}
