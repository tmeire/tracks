package database

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWithTransaction(t *testing.T) {
	// Setup SQLite in-memory DB
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	defer db.Close()

	// Create a table for testing
	_, err = db.Exec("CREATE TABLE test (id INTEGER PRIMARY KEY, value TEXT)")
	require.NoError(t, err)

	ctx := WithDB(context.Background(), db)

	t.Run("Commit", func(t *testing.T) {
		err := WithTransaction(ctx, func(txCtx context.Context) error {
			txDB := FromContext(txCtx)
			_, err := txDB.ExecContext(txCtx, "INSERT INTO test (value) VALUES (?)", "commit_test")
			return err
		})
		require.NoError(t, err)

		// Verify data exists
		var count int
		err = db.QueryRow("SELECT COUNT(*) FROM test WHERE value = ?", "commit_test").Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, 1, count)
	})

	t.Run("Rollback on Error", func(t *testing.T) {
		err := WithTransaction(ctx, func(txCtx context.Context) error {
			txDB := FromContext(txCtx)
			_, err := txDB.ExecContext(txCtx, "INSERT INTO test (value) VALUES (?)", "rollback_test")
			require.NoError(t, err)
			return errors.New("intentional error")
		})
		assert.Error(t, err)

		// Verify data does not exist
		var count int
		err = db.QueryRow("SELECT COUNT(*) FROM test WHERE value = ?", "rollback_test").Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, 0, count)
	})

	t.Run("Rollback on Panic", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				// Expected panic
			}
		}()

		_ = WithTransaction(ctx, func(txCtx context.Context) error {
			txDB := FromContext(txCtx)
			_, err := txDB.ExecContext(txCtx, "INSERT INTO test (value) VALUES (?)", "panic_test")
			require.NoError(t, err)
			panic("intentional panic")
		})

		// Verify data does not exist
		var count int
		err = db.QueryRow("SELECT COUNT(*) FROM test WHERE value = ?", "panic_test").Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, 0, count)
	})

	t.Run("Nested Transaction (Reuse)", func(t *testing.T) {
		err := WithTransaction(ctx, func(txCtx context.Context) error {
			// First insert
			txDB := FromContext(txCtx)
			_, err := txDB.ExecContext(txCtx, "INSERT INTO test (value) VALUES (?)", "nested_outer")
			require.NoError(t, err)

			// Nested call
			err = WithTransaction(txCtx, func(innerCtx context.Context) error {
				innerDB := FromContext(innerCtx)
				// Ensure we are using the same wrapper/transaction
				assert.Equal(t, txDB, innerDB)

				_, err := innerDB.ExecContext(innerCtx, "INSERT INTO test (value) VALUES (?)", "nested_inner")
				return err
			})
			return err
		})
		require.NoError(t, err)

		// Verify both records exist
		var count int
		err = db.QueryRow("SELECT COUNT(*) FROM test WHERE value IN (?, ?)", "nested_outer", "nested_inner").Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, 2, count)
	})

    t.Run("Nested Transaction Rollback", func(t *testing.T) {
        // If the inner transaction fails, the whole thing should fail because we reuse the transaction.
        // NOTE: In true nested transactions (savepoints), inner could fail and outer could recover.
        // But here we implement REUSE. So if inner returns error, it doesn't rollback immediately?
        // Wait, looking at implementation:
        /*
        	if _, ok := db.(*txWrapper); ok {
		        return fn(ctx)
	        }
        */
        // It just calls fn(ctx). If fn returns error, the outer WithTransaction sees the error.
        
		err := WithTransaction(ctx, func(txCtx context.Context) error {
			txDB := FromContext(txCtx)
			_, err := txDB.ExecContext(txCtx, "INSERT INTO test (value) VALUES (?)", "nested_rollback_outer")
			require.NoError(t, err)

			err = WithTransaction(txCtx, func(innerCtx context.Context) error {
				innerDB := FromContext(innerCtx)
				_, err := innerDB.ExecContext(innerCtx, "INSERT INTO test (value) VALUES (?)", "nested_rollback_inner")
                require.NoError(t, err)
				return errors.New("inner error")
			})
            // If inner returns error, we (outer) return error, triggering rollback of the single shared transaction.
			return err
		})
		assert.Error(t, err)

		// Verify data does not exist
		var count int
		err = db.QueryRow("SELECT COUNT(*) FROM test WHERE value IN (?, ?)", "nested_rollback_outer", "nested_rollback_inner").Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, 0, count)
	})
}
