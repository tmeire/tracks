package featureflags

import (
    "database/sql"
    "fmt"
    "path/filepath"

    "github.com/spf13/cobra"
    "github.com/tmeire/tracks/database/sqlite"
)

// GetCmd shows a flag and all overrides
func GetCmd() *cobra.Command {
    var dbPath string
    cmd := &cobra.Command{
        Use:   "get <key>",
        Short: "Show a feature flag and its overrides",
        Args:  cobra.ExactArgs(1),
        Run: func(cmd *cobra.Command, args []string) {
            key := args[0]
            db, err := sqlite.New(dbPath)
            if err != nil {
                cmd.PrintErrf("failed to open db: %v\n", err)
                return
            }
            defer db.Close()

            var desc string
            var def bool
            err = db.QueryRow(`SELECT description, default_value FROM feature_flags WHERE key=?`, key).Scan(&desc, &def)
            if err != nil {
                if err == sql.ErrNoRows {
                    fmt.Fprintf(cmd.OutOrStdout(), "flag %q not found in DB (did you register it in code?)\n", key)
                    return
                }
                cmd.PrintErrf("query error: %v\n", err)
                return
            }
            fmt.Fprintf(cmd.OutOrStdout(), "%s\n  description: %s\n  default: %t\n", key, desc, def)

            rows, err := db.Query(`SELECT principal_type, IFNULL(principal_id,''), value FROM feature_flag_overrides WHERE flag_key=? ORDER BY principal_type, principal_id`, key)
            if err != nil {
                cmd.PrintErrf("overrides query error: %v\n", err)
                return
            }
            defer rows.Close()
            fmt.Fprintln(cmd.OutOrStdout(), "  overrides:")
            empty := true
            for rows.Next() {
                empty = false
                var pt, pid string
                var val bool
                if err := rows.Scan(&pt, &pid, &val); err != nil {
                    cmd.PrintErrf("scan error: %v\n", err)
                    return
                }
                if pt == "global" { pid = "-" }
                fmt.Fprintf(cmd.OutOrStdout(), "    %s/%s: %t\n", pt, pid, val)
            }
            if empty {
                fmt.Fprintln(cmd.OutOrStdout(), "    (none)")
            }
        },
    }
    cmd.Flags().StringVar(&dbPath, "db", filepath.Join(".", "data", "tracks.sqlite"), "Path to central database")
    return cmd
}
