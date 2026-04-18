package authentication

import (
	"context"
	"crypto/rand"
	"database/sql"
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
	ID              string `json:"id"`
	Email           string `json:"email"`
	Name            string `json:"name"`
	password        string
	ActivationToken string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// TableName returns the name of the database table for this model
func (*User) TableName() string {
	return "users"
}

// Fields returns the list of field names for this model
func (*User) Fields() []string {
	return []string{"email", "name", "password", "activation_token", "created_at", "updated_at"}
}

// Values returns the values of the fields in the same order as Fields()
func (s *User) Values() []any {
	return []any{s.Email, s.Name, s.password, s.ActivationToken, s.CreatedAt, s.UpdatedAt}
}

// Scan scans the values from a row into this model
func (*User) Scan(_ context.Context, _ *Schema, row database.Scanner) (*User, error) {
	var n User
	var activationToken sql.NullString

	err := row.Scan(&n.ID, &n.Email, &n.Name, &n.password, &activationToken, &n.CreatedAt, &n.UpdatedAt)
	if err != nil {
		return nil, err
	}
	n.ActivationToken = activationToken.String
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

// GenerateActivationToken generates a secure random token for user activation/password setup.
func (s *User) GenerateActivationToken() string {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	s.ActivationToken = hex.EncodeToString(b)
	return s.ActivationToken
}

// ClearActivationToken clears the activation token.
func (s *User) ClearActivationToken() {
	s.ActivationToken = ""
}

type UsersResource struct {
	schema *Schema
}

func (u *UsersResource) Index(r *http.Request) (any, error) {
	userId, ok := session.FromRequest(r).Authenticated()
	if !ok {
		return &tracks.Response{
			StatusCode: http.StatusSeeOther,
			Location:   "/",
		}, nil
	}

	return u.schema.users.FindByID(r.Context(), userId)
}

func (u *UsersResource) New(r *http.Request) (any, error) {
	if session.FromRequest(r).IsAuthenticated() {
		return &tracks.Response{
			StatusCode: http.StatusSeeOther,
			Location:   "/users/",
		}, nil
	}

	// Return nil to use the default template rendering
	return nil, nil
}

func (u *UsersResource) Create(r *http.Request) (any, error) {
	if session.FromRequest(r).IsAuthenticated() {
		return &tracks.Response{
			StatusCode: http.StatusSeeOther,
			Location:   "/users/",
		}, nil
	}

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

	user, err := u.schema.CreateNewUser(r.Context(), name, email, password)
	if err != nil {
		session.Flash(r, "alert", err.Error())
		return &tracks.Response{
			StatusCode: http.StatusUnprocessableEntity,
			Location:   "/users/new",
		}, nil
	}

	// Trigger post-user-creation hooks
	executePostUserCreationHooks(r.Context(), user)

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

func (u *UsersResource) Activate(r *http.Request) (any, error) {
	token := r.URL.Query().Get("token")
	if token == "" {
		return &tracks.Response{StatusCode: http.StatusSeeOther, Location: "/"}, nil
	}

	users, err := u.schema.users.FindBy(r.Context(), map[string]any{"activation_token": token})
	if err != nil || len(users) == 0 {
		session.Flash(r, "error", "Invalid or expired activation token")
		return &tracks.Response{StatusCode: http.StatusSeeOther, Location: "/"}, nil
	}

	return map[string]string{"token": token}, nil
}

func (u *UsersResource) SetPasswordWithToken(r *http.Request) (any, error) {
	token := r.FormValue("token")
	password := r.FormValue("password")
	passwordConfirmation := r.FormValue("password_confirmation")

	if password == "" || password != passwordConfirmation {
		session.Flash(r, "error", "Passwords must match and cannot be empty")
		return &tracks.Response{StatusCode: http.StatusUnprocessableEntity, Location: "/users/activate?token=" + token}, nil
	}

	users, err := u.schema.users.FindBy(r.Context(), map[string]any{"activation_token": token})
	if err != nil || len(users) == 0 {
		session.Flash(r, "error", "Invalid or expired activation token")
		return &tracks.Response{StatusCode: http.StatusSeeOther, Location: "/"}, nil
	}

	user := users[0]
	if err := user.SetPassword(password); err != nil {
		return nil, err
	}
	user.ClearActivationToken()

	if err := u.schema.users.Update(r.Context(), user); err != nil {
		return nil, err
	}

	// Authenticate the user
	sess := session.FromRequest(r)
	sess.Authenticate(user.ID)

	session.Flash(r, "notice", "Password set successfully. Welcome!")
	return &tracks.Response{StatusCode: http.StatusSeeOther, Location: "/"}, nil
}
