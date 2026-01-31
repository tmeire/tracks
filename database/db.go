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

type Schema interface{}

// Model is the interface that all database models must implement
type Model[S Schema, T any] interface {
	// TableName returns the name of the database table for this model
	TableName() string
	// Fields returns the list of field names for this model
	Fields() []string
	// Values returns the values of the fields in the same order as Fields()
	Values() []any
	// Scan scans the values from a row into this model
	Scan(ctx context.Context, schema S, row Scanner) (T, error)
	// HasAutoIncrementID returns true if the ID is auto-incremented by the database
	HasAutoIncrementID() bool
	// GetID returns the ID of the model
	GetID() any
}

// BeforeCreateHook is called before a record is created
type BeforeCreateHook interface {
	BeforeCreate(ctx context.Context) error
}

// AfterCreateHook is called after a record is created
type AfterCreateHook interface {
	AfterCreate(ctx context.Context) error
}

// BeforeUpdateHook is called before a record is updated
type BeforeUpdateHook interface {
	BeforeUpdate(ctx context.Context) error
}

// AfterUpdateHook is called after a record is updated
type AfterUpdateHook interface {
	AfterUpdate(ctx context.Context) error
}

// BeforeDeleteHook is called before a record is deleted
type BeforeDeleteHook interface {
	BeforeDelete(ctx context.Context) error
}

// AfterDeleteHook is called after a record is deleted
type AfterDeleteHook interface {
	AfterDelete(ctx context.Context) error
}

type skipHooksKey struct{}

// SkipHooks returns a new context that tells the repository to skip lifecycle hooks
func SkipHooks(ctx context.Context) context.Context {
	return context.WithValue(ctx, skipHooksKey{}, true)
}

func shouldSkipHooks(ctx context.Context) bool {
	skip, _ := ctx.Value(skipHooksKey{}).(bool)
	return skip
}
