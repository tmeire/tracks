package multitenancy

import (
	"context"
	"github.com/tmeire/tracks/database"
	"time"
)

// Tenant represents a tenant in the system
type Tenant struct {
	ID        int
	Name      string
	Subdomain string
	DBPath    string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// TableName returns the name of the database table for this model
func (t *Tenant) TableName() string {
	return "Tenants"
}

// Fields returns the list of field names for this model
func (t *Tenant) Fields() []string {
	return []string{"name", "subdomain", "db_path", "created_at", "updated_at"}
}

// Values returns the values of the fields in the same order as Fields()
func (t *Tenant) Values() []any {
	return []any{t.Name, t.Subdomain, t.DBPath, t.CreatedAt, t.UpdatedAt}
}

// Scan scans the values from a row into this model
func (t *Tenant) Scan(_ context.Context, _ *Schema, row database.Scanner) (*Tenant, error) {
	var ret = Tenant{}
	err := row.Scan(&ret.ID, &ret.Name, &ret.Subdomain, &ret.DBPath, &ret.CreatedAt, &ret.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &ret, nil
}

// HasAutoIncrementID returns true if the ID is auto-incremented by the database
func (t *Tenant) HasAutoIncrementID() bool {
	return true
}

// GetID returns the ID of the model
func (t *Tenant) GetID() any {
	return t.ID
}

// UserRole represents a user's role within a tenant
type UserRole struct {
	ID        int64
	UserID    string
	TenantID  int64
	Role      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// TableName returns the name of the database table for this model
func (ur *UserRole) TableName() string {
	return "user_roles"
}

// Fields returns the list of field names for this model
func (ur *UserRole) Fields() []string {
	return []string{"user_id", "tenant_id", "role", "created_at", "updated_at"}
}

// Values returns the values of the fields in the same order as Fields()
func (ur *UserRole) Values() []any {
	return []any{ur.UserID, ur.TenantID, ur.Role, ur.CreatedAt, ur.UpdatedAt}
}

// Scan scans the values from a row into this model
func (ur *UserRole) Scan(_ context.Context, _ *Schema, row database.Scanner) (*UserRole, error) {
	var res UserRole
	err := row.Scan(&res.ID, &res.UserID, &res.TenantID, &res.Role, &res.CreatedAt, &res.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &res, nil
}

// HasAutoIncrementID returns true if the ID is auto-incremented by the database
func (ur *UserRole) HasAutoIncrementID() bool {
	return true
}

// GetID returns the ID of the model
func (ur *UserRole) GetID() any {
	return ur.ID
}
