package authentication

import (
	"github.com/tmeire/tracks/database"
	"net/http"

	"github.com/tmeire/tracks"
	"github.com/tmeire/tracks/session"
)

type SessionsResource struct{}

const (
	loginRefererKey = "login-referer"
)

func (s *SessionsResource) New(r *http.Request) (any, error) {
	if session.FromRequest(r).IsAuthenticated() {
		return &tracks.Response{
			StatusCode: http.StatusSeeOther,
			Location:   "/",
		}, nil
	}
	return nil, nil
}

func (s *SessionsResource) Create(r *http.Request) (any, error) {
	if session.FromRequest(r).IsAuthenticated() {
		return &tracks.Response{
			StatusCode: http.StatusSeeOther,
			Location:   "/",
		}, nil
	}

	if err := r.ParseForm(); err != nil {
		return nil, err
	}

	email := r.PostFormValue("email")
	password := r.PostFormValue("password")

	if email == "" || password == "" {
		session.Flash(r, "alert", "email and password are required")

		return &tracks.Response{
			StatusCode: http.StatusUnauthorized,
			Location:   "/sessions/new",
		}, nil
	}

	// Find the user by email
	users, err := database.NewRepositoryFromContext[*User](r.Context()).
		FindBy(r.Context(), map[string]any{"email": email})
	if err != nil {
		return nil, err
	}

	var user *User
	if len(users) > 0 {
		user = users[0]
	}

	if user == nil || !user.ValidatePassword(password) {
		session.Flash(r, "alert", "invalid credentials")

		return &tracks.Response{
			StatusCode: http.StatusUnauthorized,
			Location:   "/sessions/new",
		}, nil
	}

	sess := session.FromRequest(r)
	sess.Authenticate(user.ID)

	referer, ok := sess.Get(loginRefererKey)
	if !ok || referer == "" {
		referer = "/"
	}
	defer sess.Forget(loginRefererKey)

	return &tracks.Response{
		StatusCode: http.StatusNoContent,
		Location:   referer,
	}, nil
}

func (s *SessionsResource) Delete(r *http.Request) (any, error) {
	session.Invalidate(r)

	return &tracks.Response{
		StatusCode: http.StatusNoContent,
		Location:   "/",
	}, nil
}
