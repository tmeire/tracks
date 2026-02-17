package authentication

import (
	"context"
	"fmt"
	"time"

	"github.com/tmeire/tracks/database"
)

type Schema struct {
	users *database.Repository[*Schema, *User]
}

func NewSchema() *Schema {
	s := &Schema{}
	s.users = database.NewRepository[*Schema, *User](s)
	return s
}

func (s *Schema) CreateNewUser(ctx context.Context, name, email, password string) (*User, error) {
	// Check if a user with this email already exists
	existingUsers, err := s.users.FindBy(ctx, map[string]any{"email": email})
	if err != nil {
		return nil, err
	}

	if len(existingUsers) > 0 {
		return nil, fmt.Errorf("user with email %s already exists", email)
	}

	// Create a new user
	now := time.Now()
	user := &User{
		ID:        email,
		Email:     email,
		Name:      name,
		CreatedAt: now,
		UpdatedAt: now,
	}

	// Set the password, if any
	if password != "" {
		if err := user.SetPassword(password); err != nil {
			return nil, err
		}
	}

	// Save the user to the database
	return s.users.Create(ctx, user)
}
