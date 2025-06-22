package authentication

import (
	"encoding/json"
	"fmt"
	"github.com/tmeire/tracks"
	"github.com/tmeire/tracks/session"
	"net/http"
	"strings"
)

const (
	htmlMediaType = "text/html"
)

func authenticate(domain string, secure bool) tracks.Middleware {
	return func(h http.Handler) (http.Handler, error) {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !session.FromRequest(r).IsAuthenticated() {
				accept := r.Header.Get("Accept")
				if strings.Contains(accept, htmlMediaType) {
					scheme := "http"
					if secure {
						scheme = "https"
					}

					host := fmt.Sprintf("%s://%s", scheme, domain)

					sess := session.FromRequest(r)
					sess.Put(loginRefererKey, scheme+"://"+r.Host+r.URL.Path)

					w.Header().Set("Location", host+"/sessions/new")
					w.WriteHeader(http.StatusSeeOther)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
				return
			}

			h.ServeHTTP(w, r)
		}), nil
	}
}

// Register sets up authentication-related routes and middleware for the application.
// It configures endpoints for user sessions including login, logout, and applies
// authentication middleware to protected routes.
//
// Parameters:
//   - r: A pointer to tracks.Router instance to register routes on
//
// Returns:
//   - tracks.Router: The modified router with authentication routes and middleware
func Register(r tracks.Router) tracks.Router {
	sr := SessionsResource{}
	ur := UsersResource{
		schema: newSchema(),
	}

	return r.
		// Login screen
		GetFunc("/sessions/new", "sessions", "new", sr.New).
		// Login action
		PostFunc("/sessions/", "sessions", "create", sr.Create).
		GetFunc("/users/", "users", "index", ur.Index).
		// Registration page
		GetFunc("/users/new", "users", "new", ur.New).
		// Registration action
		PostFunc("/users/", "users", "create", ur.Create).
		RequestMiddleware(authenticate(r.BaseDomain(), r.Secure())).
		// Logout action
		DeleteFunc("/sessions/", "sessions", "destroy", sr.Delete)
}
