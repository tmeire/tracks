package featureflags

import (
    "fmt"
    "path/filepath"
    "strings"

    "github.com/spf13/cobra"
    "github.com/tmeire/tracks/database/sqlite"
)

// SetCmd sets an override for a flag for a given principal
func SetCmd() *cobra.Command {
    var (
        dbPath       string
        asGlobal     bool
        tenantID     string
        roleID       string
        userID       string
    )
    cmd := &cobra.Command{
        Use:   "set <key> <on|off>",
        Short: "Set an override for a feature flag",
        Args:  cobra.ExactArgs(2),
        Run: func(cmd *cobra.Command, args []string) {
            key := args[0]
            valStr := strings.ToLower(args[1])
            var value bool
            switch valStr {
            case "on", "true", "1", "enable", "enabled":
                value = true
            case "off", "false", "0", "disable", "disabled":
                value = false
            default:
                cmd.PrintErrf("invalid value: %s (use on/off)\n", valStr)
                return
            }

            // determine principal
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

            // Update first (handles NULL principal_id via IFNULL), then insert if no row updated
            res, err := db.Exec(`UPDATE feature_flag_overrides
                SET value=?, updated_at=CURRENT_TIMESTAMP
                WHERE flag_key=? AND principal_type=? AND IFNULL(principal_id,'')=IFNULL(?, '')`, value, key, pType, nullIfEmpty(pID))
            if err != nil {
                cmd.PrintErrf("failed to set override: %v\n", err)
                return
            }
            if n, _ := res.RowsAffected(); n == 0 {
                if _, err := db.Exec(`INSERT INTO feature_flag_overrides(flag_key, principal_type, principal_id, value) VALUES(?,?,?,?)`, key, pType, nullIfEmpty(pID), value); err != nil {
                    cmd.PrintErrf("failed to insert override: %v\n", err)
                    return
                }
            }
            fmt.Fprintf(cmd.OutOrStdout(), "set %s for %s/%s to %t\n", key, pType, nonEmpty(pID, "-"), value)
        },
    }
    cmd.Flags().StringVar(&dbPath, "db", filepath.Join(".", "data", "tracks.sqlite"), "Path to central database")
    cmd.Flags().BoolVar(&asGlobal, "global", false, "Set a global override")
    cmd.Flags().StringVar(&tenantID, "tenant", "", "Tenant ID")
    cmd.Flags().StringVar(&roleID, "role", "", "Role ID/name")
    cmd.Flags().StringVar(&userID, "user", "", "User ID")
    return cmd
}

func nullIfEmpty(s string) any { if s == "" { return nil }; return s }
func nonEmpty(s, alt string) string { if s == "" { return alt }; return s }
