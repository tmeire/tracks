package cmd

import (
    "github.com/spf13/cobra"
    featureflagscmd "github.com/tmeire/tracks/cli/cmd/featureflags"
)

// FeatureFlagsCmd returns a cobra.Command for the featureflags command
func FeatureFlagsCmd() *cobra.Command {
    root := &cobra.Command{
        Use:   "featureflags",
        Short: "Manage feature flags",
        Long:  `Manage feature flags: list, get, set, unset overrides in the central database`,
    }

    root.AddCommand(featureflagscmd.ListCmd())
    root.AddCommand(featureflagscmd.GetCmd())
    root.AddCommand(featureflagscmd.SetCmd())
    root.AddCommand(featureflagscmd.UnsetCmd())

    return root
}
