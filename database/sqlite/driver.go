package sqlite

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"

	_ "github.com/mattn/go-sqlite3" // Import SQLite driver

	"github.com/XSAM/otelsql"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
)

// DefaultBusyTimeoutMs is the busy_timeout applied to every connection unless configured otherwise.
const DefaultBusyTimeoutMs = 5000

// Options controls the per-connection pragmas that are applied to every pooled SQLite connection.
type Options struct {
	// BusyTimeoutMs is the busy_timeout in milliseconds. 0 means DefaultBusyTimeoutMs, a negative
	// value disables the pragma.
	BusyTimeoutMs int
	// ForeignKeys enables PRAGMA foreign_keys=ON on every connection. It defaults to off because
	// table-rebuild migrations (DROP TABLE + RENAME) run inside a transaction, where
	// PRAGMA foreign_keys=OFF is a no-op, and would cascade-delete or fail on child rows.
	ForeignKeys bool
}

var (
	defaultsMu sync.RWMutex
	defaults   Options
)

// SetDefaults sets the options used by New. It is called when the database is created from the
// application config so that every database opened afterwards (e.g. one per tenant) uses the same options.
func SetDefaults(o Options) {
	defaultsMu.Lock()
	defer defaultsMu.Unlock()
	defaults = o
}

func currentDefaults() Options {
	defaultsMu.RLock()
	defer defaultsMu.RUnlock()
	return defaults
}

// dsn appends the connection pragmas as go-sqlite3 DSN parameters, so that they are applied to every
// connection the pool opens, not only the first one.
func dsn(dbPath string, o Options) string {
	var params []string
	switch {
	case o.BusyTimeoutMs == 0:
		params = append(params, fmt.Sprintf("_busy_timeout=%d", DefaultBusyTimeoutMs))
	case o.BusyTimeoutMs > 0:
		params = append(params, fmt.Sprintf("_busy_timeout=%d", o.BusyTimeoutMs))
	}
	if o.ForeignKeys {
		params = append(params, "_foreign_keys=ON")
	}
	if len(params) == 0 {
		return dbPath
	}
	sep := "?"
	if strings.Contains(dbPath, "?") {
		sep = "&"
	}
	return dbPath + sep + strings.Join(params, "&")
}

// New creates a new SQLite database connection with OpenTelemetry tracing, using the default options.
func New(dbPath string) (*sql.DB, error) {
	return NewWithOptions(dbPath, currentDefaults())
}

// NewWithOptions creates a new SQLite database connection with OpenTelemetry tracing and the given options.
func NewWithOptions(dbPath string, opts Options) (*sql.DB, error) {
	// The file: prefix is not a path; only create directories for plain paths.
	fsPath := strings.TrimPrefix(dbPath, "file:")
	if i := strings.Index(fsPath, "?"); i >= 0 {
		fsPath = fsPath[:i]
	}
	if fsPath == ":memory:" || fsPath == "" {
		fsPath = ""
	}
	// Ensure the directory exists
	if fsPath != "" {
		if err := os.MkdirAll(filepath.Dir(fsPath), 0755); err != nil {
			return nil, fmt.Errorf("failed to create database directory: %w", err)
		}
	}

	// Connect to database
	sqlDB, err := otelsql.Open("sqlite3", dsn(dbPath, opts), otelsql.WithAttributes(
		semconv.DBSystemSqlite,
	))
	if err != nil {
		log.Fatal(err)
	}

	// Test the connection
	if err := sqlDB.Ping(); err != nil {
		sqlDB.Close()
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Always enable WAL so we can replicate with litestream if needed
	if _, err := sqlDB.Exec("PRAGMA journal_mode=WAL;"); err != nil {
		return nil, fmt.Errorf("failed to enable WAL mode: %w", err)
	}

	// Register DB stats to meter
	_, err = otelsql.RegisterDBStatsMetrics(sqlDB, otelsql.WithAttributes(
		semconv.DBSystemSqlite,
	))
	if err != nil {
		log.Fatal(err)
	}

	return sqlDB, nil
}
