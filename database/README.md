# Database Library

This package provides a simple database library for the Tracks framework. It is designed to meet the following requirements:

- No use of reflection for marshaling/unmarshaling data
- Minimal duplication of field lists

## Overview

The library consists of the following components:

- **Database Interface**: Defines methods for executing queries and commands against a database.
- **Model Interface**: Defines methods that models must implement to work with the database.
- **Repository**: Provides CRUD operations for models.
- **SQLite Driver**: Implements the Database interface for SQLite.

## Usage

### Defining a Model

To use the database library, models must implement the `Model` interface:

```go
type Model[T any] interface {
    // TableName returns the name of the database table for this model
    TableName() string
    // Fields returns the list of field names for this model
    Fields() []string
    // Values returns the values of the fields in the same order as Fields()
    Values() []any
    // Scan scans the values from a row into this model
    Scan(row *sql.Rows) (T, error)
}
```

Example implementation:

```go
type User struct {
    ID    int
    Name  string
    Email string
}

func (*User) TableName() string {
    return "users"
}

func (*User) Fields() []string {
    return []string{"id", "name", "email"}
}

func (u *User) Values() []any {
    return []any{u.ID, u.Name, u.Email}
}

func (*User) Scan(row *sql.Rows) (*User, error) {
    var u User
    err := row.Scan(&u.ID, &u.Name, &u.Email)
    if err != nil {
        return nil, err
    }
    return &u, nil
}
```

### Using the Repository

The `Repository` provides CRUD operations for models:

```go
// Create a new SQLite database connection
sqliteDB, err := sqlite.New("./data/database.tracks_db")
if err != nil {
    log.Fatalf("failed to connect to database: %v", err)
}
defer sqliteDB.Close()

// Create a new repository with the SQLite database
repo := db.NewRepository[*User](sqliteDB)

// Create a new user
user := &User{Name: "John", Email: "john@example.com"}
user, err = repo.Create(user)
if err != nil {
    log.Fatalf("failed to create user: %v", err)
}

// Find a user by ID
user, err = repo.FindByID(1)
if err != nil {
    log.Fatalf("failed to find user: %v", err)
}

// Update a user
user.Name = "Jane"
err = repo.Update(user)
if err != nil {
    log.Fatalf("failed to update user: %v", err)
}

// Delete a user
err = repo.Delete(user)
if err != nil {
    log.Fatalf("failed to delete user: %v", err)
}

// Find all users
users, err := repo.FindAll()
if err != nil {
    log.Fatalf("failed to find all users: %v", err)
}
```

## Benefits

This library provides several benefits:

1. **No Reflection**: All field mapping is done explicitly through the `Fields()` and `Values()` methods, avoiding the use of reflection.
2. **Minimal Duplication**: Field lists are defined once in the model and reused for all database operations.
3. **Type Safety**: The use of generics ensures type safety when working with repositories.
4. **Persistence**: The SQLite implementation provides persistent storage for your data.
5. **Extensibility**: The interface-based design makes it easy to add support for different database backends.
