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
	PlanID    string
	Active    bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

// TableName returns the name of the database table for this model
func (t *Tenant) TableName() string {
	return "tenants"
}

// Fields returns the list of field names for this model
func (t *Tenant) Fields() []string {
	return []string{"name", "subdomain", "db_path", "plan_id", "active", "created_at", "updated_at"}
}

// Values returns the values of the fields in the same order as Fields()
func (t *Tenant) Values() []any {
	return []any{t.Name, t.Subdomain, t.DBPath, t.PlanID, t.Active, t.CreatedAt, t.UpdatedAt}
}

// Scan scans the values from a row into this model
func (t *Tenant) Scan(_ context.Context, _ *Schema, row database.Scanner) (*Tenant, error) {
	var ret = Tenant{}
	err := row.Scan(&ret.ID, &ret.Name, &ret.Subdomain, &ret.DBPath, &ret.PlanID, &ret.Active, &ret.CreatedAt, &ret.UpdatedAt)
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

// Profile represents a freelancer's public profile
type Profile struct {
	ID                 int
	UserID             string
	Bio                string
	PortfolioURL       string
	Specialties        string
	IsPublic           bool
	AvailabilityStatus string
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

// TableName returns the name of the database table for this model
func (p *Profile) TableName() string {
	return "profiles"
}

// Fields returns the list of field names for this model
func (p *Profile) Fields() []string {
	return []string{"user_id", "bio", "portfolio_url", "specialties", "is_public", "availability_status", "created_at", "updated_at"}
}

// Values returns the values of the fields in the same order as Fields()
func (p *Profile) Values() []any {
	return []any{p.UserID, p.Bio, p.PortfolioURL, p.Specialties, p.IsPublic, p.AvailabilityStatus, p.CreatedAt, p.UpdatedAt}
}

// Scan scans the values from a row into this model
func (p *Profile) Scan(_ context.Context, _ *Schema, row database.Scanner) (*Profile, error) {
	var ret = Profile{}
	err := row.Scan(&ret.ID, &ret.UserID, &ret.Bio, &ret.PortfolioURL, &ret.Specialties, &ret.IsPublic, &ret.AvailabilityStatus, &ret.CreatedAt, &ret.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &ret, nil
}

// HasAutoIncrementID returns true if the ID is auto-incremented by the database
func (p *Profile) HasAutoIncrementID() bool {
	return true
}

// GetID returns the ID of the model
func (p *Profile) GetID() any {
	return p.ID
}
