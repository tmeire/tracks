package authentication

import (
	"context"
	"time"

	"github.com/tmeire/tracks/database"
)

type SystemRole struct {
	ID        int
	UserID    string
	Role      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// TableName returns the name of the database table for this model
func (*SystemRole) TableName() string {
	return "system_roles"
}

// Fields returns the list of field names for this model
func (*SystemRole) Fields() []string {
	return []string{"user_id", "role", "created_at", "updated_at"}
}

// Values returns the values of the fields in the same order as Fields()
func (s *SystemRole) Values() []any {
	return []any{s.UserID, s.Role, s.CreatedAt, s.UpdatedAt}
}

// Scan scans the values from a row into this model
func (*SystemRole) Scan(_ context.Context, _ *Schema, row database.Scanner) (*SystemRole, error) {
	var n SystemRole

	err := row.Scan(&n.ID, &n.UserID, &n.Role, &n.CreatedAt, &n.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &n, nil
}

// HasAutoIncrementID returns true if the ID is auto-incremented by the database
func (*SystemRole) HasAutoIncrementID() bool {
	return true
}

// GetID returns the ID of the model
func (s *SystemRole) GetID() any {
	return s.ID
}
