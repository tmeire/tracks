# Design Document: tracks Repository Lifecycle Hooks

## 1. Overview
The `tracks` framework currently lacks a mechanism to intercept data modifications. This leads to business logic (like updating cached balances) being duplicated across multiple controllers. This proposal introduces a "Hook" system for the `database.Repository` to allow centralized, consistent side-effects.

## 2. Requirements
- Support `Before` and `After` hooks for `Create`, `Update`, and `Delete` operations.
- Hooks must receive the same `context.Context` (carrying the active transaction) as the primary operation.
- Hooks must be able to cancel the operation by returning an error.
- Hook registration must be type-safe and integrated into the `Repository` initialization.

## 3. Proposed API

### Hook Interfaces
The Repository should look for specific interfaces on the Model structs.

```go
type BeforeCreateHook interface {
    BeforeCreate(ctx context.Context) error
}

type AfterCreateHook interface {
    AfterCreate(ctx context.Context) error
}

// Similarly for Update (BeforeUpdate, AfterUpdate) 
// and Delete (BeforeDelete, AfterDelete)
```

### Repository Extension
The `database.Repository` should be updated to check for these interfaces during execution:

```go
// Inside repository.go
func (r *Repository[S, T]) Create(ctx context.Context, model T) (T, error) {
    // 1. Check BeforeCreate
    if h, ok := any(model).(BeforeCreateHook); ok {
        if err := h.BeforeCreate(ctx); err != nil {
            return nil, err
        }
    }

    // 2. Perform DB Insert (existing logic)
    created, err := r.execInsert(ctx, model)
    if err != nil {
        return nil, err
    }

    // 3. Check AfterCreate
    if h, ok := any(created).(AfterCreateHook); ok {
        if err := h.AfterCreate(ctx); err != nil {
            return nil, err
        }
    }

    return created, nil
}
```

## 4. Design Considerations
- **Transaction Propagation:** Hooks **MUST** use the context passed to them. If the operation is inside a transaction, the hook's database actions must be part of that same transaction.
- **Recursion:** Implementers must be careful not to trigger the same hook indefinitely (e.g., an `AfterUpdate` hook calling `Repository.Update`). The documentation should warn against this, or the repository should implement a "NoHooks" flag in the context.
- **Performance:** Interface checks (`any(model).(Interface)`) in Go are highly optimized, but these checks should only occur if hooks are actually intended to be used.

## 5. Implementation Steps
1. Define the lifecycle interfaces in the `database` package.
2. Update `Repository.Create`, `Repository.Update`, and `Repository.Delete` to call these hooks at the appropriate times.
3. Ensure the `AfterCreate` hook receives the model *after* the ID has been populated by the database.
