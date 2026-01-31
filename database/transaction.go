package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

// TxFunc is a function that performs operations within a transaction.
type TxFunc func(ctx context.Context) error

// txWrapper wraps an *sql.Tx to satisfy the Database interface.
type txWrapper struct {
	*sql.Tx
}

// Close is a no-op for a transaction wrapper, as the transaction lifecycle
// is managed via Commit and Rollback.
func (t *txWrapper) Close() error {
	return nil
}

// Ensure txWrapper implements Database
var _ Database = (*txWrapper)(nil)

// WithTransaction creates a new transaction or reuses an existing one.
// It injects the transaction into the context so that repositories use it automatically.
func WithTransaction(ctx context.Context, fn TxFunc) error {
	db := FromContext(ctx)
	if db == nil {
		return errors.New("database not found in context")
	}

	// 1. Check if a transaction already exists in ctx
	// We check if the DB currently in context is already a *txWrapper
	if _, ok := db.(*txWrapper); ok {
		return fn(ctx)
	}

	// 2. Get *sql.DB from ctx
	sqlDB, ok := db.(*sql.DB)
	if !ok {
		// If the database in context is not *sql.DB and not a *txWrapper,
		// we can't start a transaction on it.
		return errors.New("database in context is not *sql.DB and not an active transaction")
	}

	// 3. Start Transaction
	tx, err := sqlDB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	// 4. Wrap Tx and inject into new Context
	txCtx := WithDB(ctx, &txWrapper{Tx: tx})

	// 5. Execute Callback
	// Panic handling: Ensure rollback on panic
	defer func() {
		if r := recover(); r != nil {
			_ = tx.Rollback()
			panic(r) // Re-panic after rollback
		}
	}()

	if err := fn(txCtx); err != nil {
		_ = tx.Rollback()
		return err
	}

	// 6. Commit
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}
