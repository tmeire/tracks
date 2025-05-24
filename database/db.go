package database

import (
	"context"
	"database/sql"
)

// contextKey is a type for context keys specific to the multitenancy package
type contextKey string

const (
	tenantDBContextKey contextKey = "db"
)

func WithDB(ctx context.Context, db Database) context.Context {
	return context.WithValue(ctx, tenantDBContextKey, db)
}

func FromContext(ctx context.Context) Database {
	db, ok := ctx.Value(tenantDBContextKey).(Database)
	if !ok {
		return nil
	}
	return db
}

// Database represents a connection to a database
type Database interface {
	// QueryContext executes a query that returns rows
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	// QueryRowContext executes a query that returns a single row
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
	// ExecContext executes a query that doesn't return rows
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	Close() error
}

type Scanner interface {
	Scan(dest ...any) error
}

// Model is the interface that all database models must implement
type Model[T any] interface {
	// Scan scans the values from a row into this model
	Scan(row Scanner) (T, error)
}
