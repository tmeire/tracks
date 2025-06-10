package authentication

import (
	"context"
	"encoding/hex"
	"log/slog"
	"net/http"
	"time"

	"github.com/tmeire/tracks"
	"github.com/tmeire/tracks/database"
	"github.com/tmeire/tracks/session"
	"golang.org/x/crypto/bcrypt"
)

type User struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	Name      string `json:"name"`
	password  string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// TableName returns the name of the database table for this model
func (*User) TableName() string {
	return "users"
}

// Fields returns the list of field names for this model
func (*User) Fields() []string {
	return []string{"email", "name", "password", "created_at", "updated_at"}
}

// Values returns the values of the fields in the same order as Fields()
func (s *User) Values() []any {
	return []any{s.Email, s.Name, s.password, s.CreatedAt, s.UpdatedAt}
}

// Scan scans the values from a row into this model
func (*User) Scan(_ context.Context, _ *schema, row database.Scanner) (*User, error) {
	var n User

	err := row.Scan(&n.ID, &n.Email, &n.Name, &n.password, &n.CreatedAt, &n.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &n, nil
}

// HasAutoIncrementID returns true if the ID is auto-incremented by the database
func (*User) HasAutoIncrementID() bool {
	return false
}

// GetID returns the ID of the model
func (s *User) GetID() any {
	return s.ID
}

// SetPassword encrypts the provided password using bcrypt and stores the encrypted value in the user's password field.
func (s *User) SetPassword(password string) error {
	pwEnc, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	s.password = hex.EncodeToString(pwEnc)
	return nil
}

// ValidatePassword verifies whether the given password matches the stored hashed password for the user.
func (s *User) ValidatePassword(password string) bool {
	pwEnc, err := hex.DecodeString(s.password)
	if err != nil {
		slog.Warn("user has an invalid stored password", slog.String("user", s.ID), slog.String("error", err.Error()))
		return false
	}

	return bcrypt.CompareHashAndPassword(pwEnc, []byte(password)) == nil
}

type UsersResource struct{}

func (u *UsersResource) New(r *http.Request) (any, error) {
	// Return nil to use the default template rendering
	return nil, nil
}

func (u *UsersResource) Create(r *http.Request) (any, error) {
	if err := r.ParseForm(); err != nil {
		return nil, err
	}

	email := r.PostFormValue("email")
	name := r.PostFormValue("name")
	password := r.PostFormValue("password")
	passwordConfirmation := r.PostFormValue("password_confirmation")

	// Validate input
	if email == "" || name == "" || password == "" || passwordConfirmation == "" {
		session.Flash(r, "alert", "All fields are required")
		return &tracks.Response{
			StatusCode: http.StatusUnprocessableEntity,
			Location:   "/users/new",
		}, nil
	}

	if password != passwordConfirmation {
		session.Flash(r, "alert", "Passwords do not match")
		return &tracks.Response{
			StatusCode: http.StatusUnprocessableEntity,
			Location:   "/users/new",
		}, nil
	}

	s := newSchema(database.FromContext(r.Context()))

	// Check if a user with this email already exists
	existingUsers, err := s.users.FindBy(r.Context(), map[string]any{"email": email})
	if err != nil {
		return nil, err
	}

	if len(existingUsers) > 0 {
		session.Flash(r, "alert", "A user with this email already exists")
		return &tracks.Response{
			StatusCode: http.StatusUnprocessableEntity,
			Location:   "/users/new",
		}, nil
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

	// Set the password
	if err := user.SetPassword(password); err != nil {
		return nil, err
	}

	// Save the user to the database
	user, err = s.users.Create(r.Context(), user)
	if err != nil {
		return nil, err
	}

	// Set a flash message
	session.Flash(r, "notice", "Account created successfully")

	// Authenticate the user
	sess := session.FromRequest(r)
	sess.Authenticate(user.ID)

	// Redirect to the home page
	return &tracks.Response{
		StatusCode: http.StatusCreated,
		Location:   "/",
		Data:       user,
	}, nil
}
