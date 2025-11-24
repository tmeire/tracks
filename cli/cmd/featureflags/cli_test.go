package featureflags

import (
    "bytes"
    "context"
    "path/filepath"
    "strings"
    "testing"

    "github.com/spf13/cobra"
    "github.com/tmeire/tracks/database"
    "github.com/tmeire/tracks/database/sqlite"
)

// prepareDB applies featureflags migrations and inserts one sample flag
func prepareDB(t *testing.T) (dbPath string) {
    t.Helper()
    dir := t.TempDir()
    dbPath = filepath.Join(dir, "central.sqlite")
    db, err := sqlite.New(dbPath)
    if err != nil { t.Fatalf("sqlite.New: %v", err) }
    defer db.Close()
    // apply migrations for featureflags
    migPath := filepath.Join("..", "..", "..", "modules", "featureflags", "db", "migrations", "central")
    if err := database.MigrateUpDir(context.Background(), db, database.CentralDatabase, migPath); err != nil {
        t.Fatalf("migrate: %v", err)
    }
    // insert a sample flag
    if _, err := db.Exec(`INSERT INTO feature_flags(key, description, default_value) VALUES('cli.sample', 'sample', 1)`); err != nil {
        t.Fatalf("insert flag: %v", err)
    }
    return dbPath
}

func execCmd(t *testing.T, cmd *cobra.Command) (string, string) {
    t.Helper()
    var outBuf, errBuf bytes.Buffer
    cmd.SetOut(&outBuf)
    cmd.SetErr(&errBuf)
    if err := cmd.Execute(); err != nil {
        // include buffers for easier debugging
        t.Fatalf("execute: %v, stdout=%s, stderr=%s", err, outBuf.String(), errBuf.String())
    }
    return outBuf.String(), errBuf.String()
}

func TestCLIListAndGet(t *testing.T) {
    dbPath := prepareDB(t)
    // list
    list := ListCmd()
    list.Flags().Set("db", dbPath)
    stdout, _ := execCmd(t, list)
    if !strings.Contains(stdout, "cli.sample") {
        t.Fatalf("expected list to contain cli.sample, got: %s", stdout)
    }

    // get
    get := GetCmd()
    get.SetArgs([]string{"cli.sample"})
    get.Flags().Set("db", dbPath)
    stdout, _ = execCmd(t, get)
    if !strings.Contains(stdout, "default: true") {
        t.Fatalf("expected get to show default true, got: %s", stdout)
    }
}

func TestCLISetAndUnset(t *testing.T) {
    dbPath := prepareDB(t)

    // set global off
    set := SetCmd()
    set.Flags().Set("db", dbPath)
    set.Flags().Set("global", "true")
    set.SetArgs([]string{"cli.sample", "off"})
    _, _ = execCmd(t, set)

    // verify override written directly in DB
    db, err := sqlite.New(dbPath)
    if err != nil { t.Fatalf("open db: %v", err) }
    defer db.Close()
    var c int
    if err := db.QueryRow(`SELECT COUNT(1) FROM feature_flag_overrides WHERE flag_key='cli.sample' AND principal_type='global' AND principal_id IS NULL AND value=0`).Scan(&c); err != nil {
        t.Fatalf("query override: %v", err)
    }
    if c != 1 {
        t.Fatalf("expected one global override row, got %d", c)
    }

    // get should show override listed under overrides
    get := GetCmd()
    get.Flags().Set("db", dbPath)
    get.SetArgs([]string{"cli.sample"})
    stdout, _ := execCmd(t, get)
    if !strings.Contains(stdout, "global/-: false") {
        t.Fatalf("expected global override false, got: %s", stdout)
    }

    // unset global
    unset := UnsetCmd()
    unset.Flags().Set("db", dbPath)
    unset.Flags().Set("global", "true")
    unset.SetArgs([]string{"cli.sample"})
    _, _ = execCmd(t, unset)

    // verify override removed
    get = GetCmd()
    get.Flags().Set("db", dbPath)
    get.SetArgs([]string{"cli.sample"})
    stdout, _ = execCmd(t, get)
    if !strings.Contains(stdout, "overrides:\n    (none)") && strings.Contains(stdout, "global/-:") {
        t.Fatalf("expected no overrides after unset, got: %s", stdout)
    }

    // and DB should have 0 rows
    if err := db.QueryRow(`SELECT COUNT(1) FROM feature_flag_overrides WHERE flag_key='cli.sample'`).Scan(&c); err != nil {
        t.Fatalf("count after unset: %v", err)
    }
    if c != 0 {
        t.Fatalf("expected 0 overrides after unset, got %d", c)
    }
}
