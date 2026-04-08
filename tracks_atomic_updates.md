# Design Document: tracks Atomic Field Updates

## 1. Overview
The current `Repository.Update` method is unsuitable for high-frequency counter updates (like stock levels) because it reads the whole record, modifies it in memory, and writes it back. This proposal adds an `AtomicUpdate` method to the repository to perform SQL-level increments/decrements.

## 2. Requirements
- Allow updating specific numeric fields using `SET field = field + ?`.
- Support multiple field updates in a single SQL statement.
- Must support a `WHERE` clause (typically ID-based) to target the record.
- Should return the updated values if possible, or at least confirm the number of rows affected.

## 3. Proposed API

```go
// In database/repository.go

type AtomicOp struct {
    Field string
    Delta any // int, float64, etc.
}

func (r *Repository[S, T]) AtomicUpdate(ctx context.Context, id any, ops ...AtomicOp) error {
    if len(ops) == 0 {
        return nil
    }

    // Example SQL: UPDATE inventory_items SET qty = qty + ?, reserved = reserved + ? WHERE id = ?
    query := fmt.Sprintf("UPDATE %s SET ", r.tableName())
    args := []any{}

    for i, op := range ops {
        // Validation: op.Field must be a valid column name for the model
        query += fmt.Sprintf("%s = %s + ?", op.Field, op.Field)
        args = append(args, op.Delta)
        if i < len(ops)-1 {
            query += ", "
        }
    }

    query += " WHERE id = ?"
    args = append(args, id)

    db := r.getDB(ctx)
    _, err := db.ExecContext(ctx, query, args...)
    return err
}
```

## 4. Design Considerations
- **Validation:** Field names **MUST** be validated against the allowed fields of the repository to prevent SQL injection, as they are interpolated directly into the query string.
- **Hooks:** Since this bypasses the standard `Update` flow, decide if `AfterUpdate` hooks should be triggered. If so, the full model may need to be re-loaded, which might negate some performance gains. A specialized `AfterAtomicUpdate` hook could be an alternative.
- **Negative Balances:** The application logic should still check for negative balances if required by business rules, but this tool provides the safety to ensure that two simultaneous subtractions don't result in an incorrect final balance.

## 5. Implementation Steps
1. Add `AtomicOp` struct and `AtomicUpdate` method to the generic `Repository`.
2. Implement field name validation by checking the `Fields()` method of the model.
3. Update the `Inventory` logic to use this method for updating `quantity_on_hand` and `quantity_reserved`.
