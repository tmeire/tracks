package featureflags

import (
	"context"
	"path/filepath"

	"github.com/tmeire/tracks"
	"github.com/tmeire/tracks/database"
)

// Register attaches the feature flags middleware and ensures the central
// database has the feature flag schema and current code-registered flags.
func Register(r tracks.Router) tracks.Router {
	// Apply migrations for this module explicitly (lives outside default path)
	_ = database.MigrateUpDir(context.Background(), r.Database(), database.CentralDatabase, filepath.Join("modules", "featureflags", "db", "migrations", "central"))

	// Upsert registered flags into DB for operator visibility
	repo := newRepository(r.Database())
	_ = repo.UpsertFlags(context.Background(), allRegistered())

	return r.RequestMiddleware(WithFlags(r))
}
