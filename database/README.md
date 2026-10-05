# Database Library

The Tracks database library provides a type-safe, generic repository pattern and SQLite driver for Tracks applications, with zero runtime reflection and native OpenTelemetry tracing.

## Key Features

- **No Reflection**: Field mapping is defined explicitly via `Fields()`, `Values()`, and `Scan()`.
- **Generic Repository**: `database.NewRepository[S, T](schema)` provides full CRUD and type-safe query building.
- **Lifecycle Hooks**: Automatic hooks for `BeforeCreate`, `AfterCreate`, `BeforeUpdate`, `AfterUpdate`, `BeforeDelete`, and `AfterDelete`.
- **Atomic Operations**: Safe concurrent increments/decrements via `repo.AtomicUpdate(ctx, id, ops...)`.
- **Domain-Aware Scoping**: Automatic multi-domain isolation when models embed `database.DomainScopedModel`.
- **Embedded Migrations**: Native Goose migration support for both central and tenant databases.

## Defining a Model

Models implement `database.Model[S, T]`:

```go
package models

import (
    "context"
    "time"

    "github.com/tmeire/tracks/database"
)

type User struct {
    ID        int64     `json:"id"`
    Name      string    `json:"name"`
    Email     string    `json:"email"`
    CreatedAt time.Time `json:"created_at"`
}

func (*User) TableName() string {
    return "users"
}

func (*User) Fields() []string {
    return []string{"id", "name", "email", "created_at"}
}

func (u *User) Values() []any {
    return []any{u.ID, u.Name, u.Email, u.CreatedAt}
}

func (*User) Scan(ctx context.Context, schema any, row database.Scanner) (*User, error) {
    var u User
    err := row.Scan(&u.ID, &u.Name, &u.Email, &u.CreatedAt)
    if err != nil {
        return nil, err
    }
    return &u, nil
}

func (*User) HasAutoIncrementID() bool {
    return true
}

func (u *User) GetID() any {
    return u.ID
}
```

## Using the Repository

```go
// Initialize generic repository
repo := database.NewRepository[any, *models.User](nil)

// Create
user, err := repo.Create(ctx, &models.User{
    Name:      "Alice",
    Email:     "alice@example.com",
    CreatedAt: time.Now(),
})

// Find by ID
user, err = repo.FindByID(ctx, user.ID)

// Query Builder
users, err := repo.Select("id", "name").
    Where("email LIKE ?", "%@example.com").
    OrderBy("created_at", database.DESC).
    Limit(20).
    Exec(ctx)

// Atomic Updates
err = repo.AtomicUpdate(ctx, user.ID, database.AtomicOp{
    Field: "login_count",
    Delta: 1,
})

// Transactions
err = database.WithTransaction(ctx, func(txCtx context.Context) error {
    user.Name = "Alice Smith"
    return repo.Update(txCtx, user)
})
```
