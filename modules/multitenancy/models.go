package multitenancy

import (
	"github.com/tmeire/tracks/database"
	"time"
)

// Tenant represents a tenant in the system
type Tenant struct {
	database.Model[*Tenant] `tracks:"tenants"`
	ID                      int
	Name                    string
	Subdomain               string
	DBPath                  string
	CreatedAt               time.Time
	UpdatedAt               time.Time
}

// Scan scans the values from a row into this model
func (t *Tenant) Scan(row database.Scanner) (*Tenant, error) {
	var ret = Tenant{}
	err := row.Scan(&ret.ID, &ret.Name, &ret.Subdomain, &ret.DBPath, &ret.CreatedAt, &ret.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &ret, nil
}

// UserRole represents a user's role within a tenant
type UserRole struct {
	database.Model[*UserRole] `tracks:"user_roles"`
	ID                        int64
	UserID                    string
	TenantID                  int64
	Role                      string
	CreatedAt                 time.Time
	UpdatedAt                 time.Time
}

// Scan scans the values from a row into this model
func (ur *UserRole) Scan(row database.Scanner) (*UserRole, error) {
	var res UserRole
	err := row.Scan(&res.ID, &res.UserID, &res.TenantID, &res.Role, &res.CreatedAt, &res.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &res, nil
}
