package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
	"github.com/pressly/goose/v3"
)

// DatabaseType represents the type of database (central or tenant)
type DatabaseType string

const (
	// CentralDatabase represents the central database
	CentralDatabase DatabaseType = "central"
	// TenantDatabase represents a tenant database
	TenantDatabase DatabaseType = "tenant"
)

func MigrateUp(ctx context.Context, db Database, dbType DatabaseType) error {
	// Use the appropriate migrations directory based on the database type
	migrationsDir := filepath.Join("migrations", string(dbType))

	return MigrateUpDir(ctx, db, dbType, migrationsDir)
}

func MigrateUpDir(ctx context.Context, db Database, dbType DatabaseType, migrationsDir string) error {
	rawDB, ok := db.(*sql.DB)
	if !ok {
		return errors.New("db is not a *sql.DB")
	}

	// Set the database dialect
	err := goose.SetDialect("sqlite3")
	if err != nil {
		return fmt.Errorf("failed to set dialect: %w", err)
	}

	if err := goose.UpContext(ctx, rawDB, migrationsDir); err != nil {
		return fmt.Errorf("failed to apply migrations: %w", err)
	}
	fmt.Printf("Applied %s migrations\n", dbType)
	return nil
}

// RunGooseMigration runs a Goose migration command
func RunGooseMigration(ctx context.Context, command string, dbType DatabaseType, dbPath string) error {
	// Use the provided database path or default to floral.sqlite
	if dbPath == "" {
		dbPath = filepath.Join(".", "data", "floral.sqlite")
	}

	// Set up the database connection
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %v", err)
	}
	defer db.Close()

	migrationsDir := filepath.Join("migrations", string(dbType))

	// Set the database dialect
	err = goose.SetDialect("sqlite3")
	if err != nil {
		return fmt.Errorf("failed to set dialect: %w", err)
	}

	// Run the Goose command
	switch command {
	case "up":
		if err := goose.UpContext(ctx, db, migrationsDir); err != nil {
			return fmt.Errorf("failed to apply migrations: %w", err)
		}
		fmt.Printf("Applied %s migrations\n", dbType)
	case "down":
		if err := goose.DownContext(ctx, db, migrationsDir); err != nil {
			return fmt.Errorf("failed to revert migration: %w", err)
		}
		fmt.Printf("Reverted %s migration\n", dbType)
	case "status":
		if err := goose.StatusContext(ctx, db, migrationsDir); err != nil {
			return fmt.Errorf("failed to get migration status: %w", err)
		}
	default:
		return fmt.Errorf("unknown command: %s", command)
	}
	return nil
}
