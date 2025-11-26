package featureflags

import (
	"database/sql"
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/tmeire/tracks/database/sqlite"
)

// ListCmd lists all feature flags registered in the central DB
func ListCmd() *cobra.Command {
	var dbPath string
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all feature flags",
		Run: func(cmd *cobra.Command, args []string) {
			db, err := sqlite.New(dbPath)
			if err != nil {
				cmd.PrintErrf("failed to open db: %v\n", err)
				return
			}
			defer db.Close()

			rows, err := db.Query(`SELECT key, description, default_value FROM feature_flags ORDER BY key`)
			if err != nil {
				// if table doesn't exist, show empty
				if err == sql.ErrNoRows {
					return
				}
				cmd.PrintErrf("query error: %v\n", err)
				return
			}
			defer rows.Close()
			for rows.Next() {
				var key, desc string
				var def bool
				if err := rows.Scan(&key, &desc, &def); err != nil {
					cmd.PrintErrf("scan error: %v\n", err)
					return
				}
				fmt.Fprintf(cmd.OutOrStdout(), "%s\t(default=%t)\t%s\n", key, def, desc)
			}
		},
	}
	cmd.Flags().StringVar(&dbPath, "db", filepath.Join(".", "data", "tracks.sqlite"), "Path to central database")
	return cmd
}
