package featureflags

import (
    "fmt"
    "path/filepath"

    "github.com/spf13/cobra"
    "github.com/tmeire/tracks/database/sqlite"
)

// UnsetCmd removes an override for a flag for a given principal
func UnsetCmd() *cobra.Command {
    var (
        dbPath   string
        asGlobal bool
        tenantID string
        roleID   string
        userID   string
    )
    cmd := &cobra.Command{
        Use:   "unset <key>",
        Short: "Remove an override for a feature flag",
        Args:  cobra.ExactArgs(1),
        Run: func(cmd *cobra.Command, args []string) {
            key := args[0]
            var pType, pID string
            setCount := 0
            if asGlobal { pType = "global"; pID = ""; setCount++ }
            if tenantID != "" { pType = "tenant"; pID = tenantID; setCount++ }
            if roleID != "" { pType = "role"; pID = roleID; setCount++ }
            if userID != "" { pType = "user"; pID = userID; setCount++ }
            if setCount != 1 {
                cmd.PrintErrln("specify exactly one of --global, --tenant, --role, or --user")
                return
            }

            db, err := sqlite.New(dbPath)
            if err != nil {
                cmd.PrintErrf("failed to open db: %v\n", err)
                return
            }
            defer db.Close()

            _, err = db.Exec(`DELETE FROM feature_flag_overrides WHERE flag_key=? AND principal_type=? AND IFNULL(principal_id,'')=IFNULL(?, '')`, key, pType, nullIfEmpty(pID))
            if err != nil {
                cmd.PrintErrf("failed to unset override: %v\n", err)
                return
            }
            fmt.Fprintf(cmd.OutOrStdout(), "unset %s for %s/%s\n", key, pType, nonEmpty(pID, "-"))
        },
    }
    cmd.Flags().StringVar(&dbPath, "db", filepath.Join(".", "data", "tracks.sqlite"), "Path to central database")
    cmd.Flags().BoolVar(&asGlobal, "global", false, "Global override")
    cmd.Flags().StringVar(&tenantID, "tenant", "", "Tenant ID")
    cmd.Flags().StringVar(&roleID, "role", "", "Role ID/name")
    cmd.Flags().StringVar(&userID, "user", "", "User ID")
    return cmd
}
