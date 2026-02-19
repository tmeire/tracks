package database

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
)

type schema struct {
	products Repository[*schema, TestProduct]
}

// TestProduct is a test model that implements the Model interface
type TestProduct struct {
	ID    int     `json:"id"`
	Name  string  `json:"name"`
	Price float64 `json:"price"`
}

// TableName returns the name of the database table for this model
func (p TestProduct) TableName() string {
	return "products"
}

// Fields returns the list of field names for this model
func (p TestProduct) Fields() []string {
	return []string{"name", "price"}
}

// Values returns the values of the fields in the same order as Fields()
func (p TestProduct) Values() []any {
	return []any{p.Name, p.Price}
}

// Scan scans the values from a row into this model
func (p TestProduct) Scan(ctx context.Context, schema *schema, row Scanner) (TestProduct, error) {
	var product TestProduct
	err := row.Scan(&product.ID, &product.Name, &product.Price)
	if err != nil {
		return TestProduct{}, err
	}
	return product, nil
}

// HasAutoIncrementID returns true if the ID is auto-incremented by the database
func (p TestProduct) HasAutoIncrementID() bool {
	return true
}

// GetID returns the ID of the model
func (p TestProduct) GetID() any {
	return p.ID
}

// MockDB is a mock implementation of the Database interface for testing
type MockDB struct {
	// QueryFunc is a function that will be called by QueryContext
	QueryFunc func(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	// ExecFunc is a function that will be called by ExecContext
	ExecFunc func(ctx context.Context, query string, args ...any) (sql.Result, error)
}

// QueryContext executes a query that returns rows
func (m *MockDB) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	if m.QueryFunc != nil {
		return m.QueryFunc(ctx, query, args...)
	}
	return nil, fmt.Errorf("QueryFunc not implemented")
}

// ExecContext executes a query that doesn't return rows
func (m *MockDB) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	if m.ExecFunc != nil {
		return m.ExecFunc(ctx, query, args...)
	}
	return nil, fmt.Errorf("ExecFunc not implemented")
}

// Close closes the database connection
func (m *MockDB) Close() error {
	return nil
}

// TestQueryBuilding tests that the query building functions correctly generate SQL strings and arguments
func TestQueryBuilding(t *testing.T) {
	tests := []struct {
		name         string
		setupQuery   func(*Repository[*schema, TestProduct]) Query
		expectedSQL  string
		expectedArgs []any
	}{
		{
			name: "Simple all fields",
			setupQuery: func(repo *Repository[*schema, TestProduct]) Query {
				return repo.Select()
			},
			expectedSQL:  "SELECT id, name, price FROM products",
			expectedArgs: []any{},
		},
		{
			name: "Simple Select",
			setupQuery: func(repo *Repository[*schema, TestProduct]) Query {
				return repo.Select("name", "price")
			},
			expectedSQL:  "SELECT id, name, price FROM products",
			expectedArgs: []any{},
		},
		{
			name: "Select with single Where",
			setupQuery: func(repo *Repository[*schema, TestProduct]) Query {
				return repo.Select("name", "price").Where("price > ?", 15.0)
			},
			expectedSQL:  "SELECT id, name, price FROM products WHERE price > ?",
			expectedArgs: []any{15.0},
		},
		{
			name: "Select with single composited Where",
			setupQuery: func(repo *Repository[*schema, TestProduct]) Query {
				return repo.Select("name", "price").Where("price > ? AND name = ?", 15.0, "John")
			},
			expectedSQL:  "SELECT id, name, price FROM products WHERE price > ? AND name = ?",
			expectedArgs: []any{15.0, "John"},
		},
		{
			name: "Select with Where",
			setupQuery: func(repo *Repository[*schema, TestProduct]) Query {
				return repo.Select("name", "price").Where("price > ?", 15.0).Where("name = ?", "John")
			},
			expectedSQL:  "SELECT id, name, price FROM products WHERE price > ? AND name = ?",
			expectedArgs: []any{15.0, "John"},
		},
		{
			name: "Select with Order",
			setupQuery: func(repo *Repository[*schema, TestProduct]) Query {
				return repo.Select("name", "price").Order("price", DESC)
			},
			expectedSQL:  "SELECT id, name, price FROM products ORDER BY price DESC",
			expectedArgs: []any{},
		},
		{
			name: "Select with multiple Order",
			setupQuery: func(repo *Repository[*schema, TestProduct]) Query {
				return repo.Select("name", "price").Order("name", ASC).Order("price", DESC)
			},
			expectedSQL:  "SELECT id, name, price FROM products ORDER BY name ASC, price DESC",
			expectedArgs: []any{},
		},
		{
			name: "Select with Limit",
			setupQuery: func(repo *Repository[*schema, TestProduct]) Query {
				return repo.Select("name", "price").Limit(1)
			},
			expectedSQL:  "SELECT id, name, price FROM products LIMIT 1",
			expectedArgs: []any{},
		},
		// Test proper chaining of methods
		{
			name: "Select with all clauses chained",
			setupQuery: func(repo *Repository[*schema, TestProduct]) Query {
				return repo.Select("name", "price").
					Where("price > ?", 5.0).
					Order("price", DESC).
					Limit(1).
					Offset(1)
			},
			expectedSQL:  "SELECT id, name, price FROM products WHERE price > ? ORDER BY price DESC LIMIT 1 OFFSET 1",
			expectedArgs: []any{5.0},
		},
		// Test chaining in different order
		{
			name: "Select with clauses chained in different order",
			setupQuery: func(repo *Repository[*schema, TestProduct]) Query {
				return repo.Select("name", "price").
					Where("price < ?", 100.0).
					Order("name", ASC).
					Limit(10).
					Offset(5)
			},
			expectedSQL:  "SELECT id, name, price FROM products WHERE price < ? ORDER BY name ASC LIMIT 10 OFFSET 5",
			expectedArgs: []any{100.0},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a repository with the mock database
			s := &schema{}
			repo := NewRepository[*schema, TestProduct](s)

			// Setup the query
			query := tt.setupQuery(repo)

			// Build the query and check the SQL and args
			sql, args := query.Build(context.Background())
			if sql != tt.expectedSQL {
				t.Errorf("unexpected SQL query: got %q, want %q", sql, tt.expectedSQL)
			}
			if len(args) != len(tt.expectedArgs) {
				t.Errorf("unexpected number of args: got %d, want %d", len(args), len(tt.expectedArgs))
			}
			for i, arg := range args {
				if arg != tt.expectedArgs[i] {
					t.Errorf("unexpected arg at index %d: got %v, want %v", i, arg, tt.expectedArgs[i])
				}
			}
		})
	}
}
