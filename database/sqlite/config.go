package sqlite

import (
	"database/sql"
)

type Config struct {
	Path string `json:"path"`
	// BusyTimeoutMs is the busy_timeout applied to every connection. Defaults to 5000; a negative value disables it.
	BusyTimeoutMs int `json:"busy_timeout_ms"`
	// ForeignKeys turns on PRAGMA foreign_keys for every connection. Off by default, see Options.ForeignKeys.
	ForeignKeys bool `json:"foreign_keys"`
}

// Options returns the connection options described by this config.
func (c Config) Options() Options {
	return Options{BusyTimeoutMs: c.BusyTimeoutMs, ForeignKeys: c.ForeignKeys}
}

func (c Config) Create() (*sql.DB, error) {
	// Databases opened later through New (e.g. tenant databases) inherit these options.
	SetDefaults(c.Options())
	return NewWithOptions(c.Path, c.Options())
}
