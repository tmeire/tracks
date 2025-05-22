package sqlite

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3" // Import SQLite driver

	"github.com/XSAM/otelsql"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"

	"github.com/tmeire/floral_crm/internal/tracks/database"
)

// New creates a new SQLite database connection with OpenTelemetry tracing
func New(dbPath string) (database.Database, error) {
	// Ensure the directory exists
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	// Connect to database
	sqlDB, err := otelsql.Open("sqlite3", dbPath, otelsql.WithAttributes(
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

	// Register DB stats to meter
	err = otelsql.RegisterDBStatsMetrics(sqlDB, otelsql.WithAttributes(
		semconv.DBSystemSqlite,
	))
	if err != nil {
		log.Fatal(err)
	}

	return sqlDB, nil
}
