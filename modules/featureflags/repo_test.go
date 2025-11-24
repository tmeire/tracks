package featureflags

import (
    "context"
    "os"
    "path/filepath"
    "testing"

    "github.com/tmeire/tracks/database"
    "github.com/tmeire/tracks/database/sqlite"
)

func TestRepositoryCRUDAndListOverrides(t *testing.T) {
    // temp db
    dir := t.TempDir()
    dbPath := filepath.Join(dir, "central.sqlite")
    sqlDB, err := sqlite.New(dbPath)
    if err != nil { t.Fatalf("sqlite.New: %v", err) }
    defer sqlDB.Close()

    // apply module migrations (relative to this package dir)
    migPath := filepath.Join("db", "migrations", "central")
    if err := database.MigrateUpDir(context.Background(), sqlDB, database.CentralDatabase, migPath); err != nil {
        t.Fatalf("migrate: %v", err)
    }

    // registry
    resetRegistry()
    RegisterFlags(Flag{Key: "alpha", Description: "Alpha", Default: false})
    repo := newRepository(sqlDB)
    if err := repo.UpsertFlags(context.Background(), allRegistered()); err != nil {
        t.Fatalf("UpsertFlags: %v", err)
    }

    // verify row exists
    var cnt int
    if err := sqlDB.QueryRow("SELECT COUNT(1) FROM feature_flags WHERE key='alpha'").Scan(&cnt); err != nil {
        t.Fatalf("query: %v", err)
    }
    if cnt != 1 { t.Fatalf("expected 1 feature flag, got %d", cnt) }

    // set overrides for all principal types
    ctx := context.Background()
    must := func(err error) { if err != nil { t.Fatalf("err: %v", err) } }
    must(repo.SetOverride(ctx, "alpha", Principal{Type: PrincipalGlobal}, true))
    must(repo.SetOverride(ctx, "alpha", Principal{Type: PrincipalTenant, ID: "t1"}, false))
    must(repo.SetOverride(ctx, "alpha", Principal{Type: PrincipalRole, ID: "admin"}, true))
    must(repo.SetOverride(ctx, "alpha", Principal{Type: PrincipalUser, ID: "u1"}, false))

    // list overrides for a principal set
    p := Principals{UserID: strPtr("u1"), TenantID: strPtr("t1"), RoleIDs: []string{"member", "admin"}}
    ovs, err := repo.ListOverrides(ctx, []string{"alpha"}, p)
    if err != nil { t.Fatalf("ListOverrides: %v", err) }
    if len(ovs) != 4 {
        t.Fatalf("expected 4 overrides returned, got %d", len(ovs))
    }

    // delete one override and ensure count decreases
    must(repo.DeleteOverride(ctx, "alpha", Principal{Type: PrincipalRole, ID: "admin"}))
    ovs, err = repo.ListOverrides(ctx, []string{"alpha"}, p)
    if err != nil { t.Fatalf("ListOverrides: %v", err) }
    if len(ovs) != 3 {
        t.Fatalf("expected 3 overrides after delete, got %d", len(ovs))
    }

    // update via SetOverride upsert
    must(repo.SetOverride(ctx, "alpha", Principal{Type: PrincipalTenant, ID: "t1"}, true))
    // compute should now prioritize user over tenant etc. Validate through computeEffective
    vals := computeEffective([]string{"alpha"}, ovs) // ovs currently from before tenant update; refresh
    ovs, _ = repo.ListOverrides(ctx, []string{"alpha"}, p)
    vals = computeEffective([]string{"alpha"}, ovs)
    if vals["alpha"] != false {
        t.Fatalf("expected user override false to win, got %v", vals["alpha"])
    }

    _ = os.Unsetenv("GO_ENV")
}
