# Design Document: tracks Transaction Management Utility

## 1. Overview
Currently, performing multiple database operations atomically requires manually extracting and wrapping `sql.Tx`. This proposal introduces a `database.WithTransaction` utility that handles the boilerplate of beginning, committing, and rolling back transactions.

## 2. Requirements
- Handle `BeginTx`, `Commit`, and `Rollback` (on error or panic) automatically.
- Provide a "Transaction Context" to the callback that contains the active transaction.
- Support nested transactions (via Savepoints or by simply reusing the existing transaction).
- Ensure that if a transaction is already present in the context, it is reused instead of starting a new one.

## 3. Proposed API

```go
// database/transaction.go

type TxFunc func(ctx context.Context) error

func WithTransaction(ctx context.Context, fn TxFunc) error {
    // 1. Check if a transaction already exists in ctx
    // We should check if the DB currently in context is already a *sql.Tx wrapper
    if hasActiveTransaction(ctx) {
        return fn(ctx)
    }

    // 2. Get *sql.DB from ctx
    db := FromContext(ctx)
    sqlDB, ok := db.(*sql.DB)
    if !ok {
        return errors.New("no sql.DB found in context")
    }

    // 3. Start Transaction
    tx, err := sqlDB.BeginTx(ctx, nil)
    if err != nil {
        return err
    }

    // 4. Wrap Tx and inject into new Context
    // contextWithTx should create a wrapper that satisfies the database.Database interface
    txCtx := contextWithTx(ctx, tx)

    // 5. Execute Callback
    defer func() {
        if r := recover(); r != nil {
            tx.Rollback()
            panic(r) // Re-panic after rollback
        }
    }()

    if err := fn(txCtx); err != nil {
        tx.Rollback()
        return err
    }

    // 6. Commit
    return tx.Commit()
}
```

## 4. Design Considerations
- **Panic Handling:** Using `defer` with `recover()` is critical to ensure that a crashing application doesn't leave orphaned transactions or locks on the SQLite database.
- **Context Injection:** The utility must use the existing `database.WithDB` mechanism so that all repositories called within the callback automatically use the transaction without being aware of it.
- **SQLite Limitations:** SQLite handles transactions well but only allows one writer at a time. The utility should consider a small retry mechanism or a clear error message for "database is locked" errors if high concurrency is expected.

## 5. Implementation Steps
1. Implement `WithTransaction` in the `database` package.
2. Create a `txWrapper` struct that satisfies the internal `Database` interface used by repositories.
3. Update `FromContext` to prefer an active transaction wrapper over the base DB connection.
