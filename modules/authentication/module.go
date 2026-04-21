package authentication

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/tmeire/tracks"
	"github.com/tmeire/tracks/session"
)

const (
	htmlMediaType = "text/html"
)

func WithAnyRole(r tracks.Router) tracks.Middleware {
	return authenticate(r.BaseDomain(), r.Secure())
}

func RequireSystemRole(role string) func(tracks.Router) tracks.Middleware {
	return func(r tracks.Router) tracks.Middleware {
		return func(h http.Handler) (http.Handler, error) {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				ctx := r.Context()
				sess := session.FromContext(ctx)
				userID, ok := sess.Authenticated()
				if !ok {
					// Redirect to login if not authenticated
					scheme := "http"
					if session.IsSecure(r) {
						scheme = "https"
					}
					host := fmt.Sprintf("%s://%s", scheme, r.Host)
					w.Header().Set("Location", host+"/sessions/new")
					w.WriteHeader(http.StatusSeeOther)
					return
				}

				auth := NewSchema()
				roles, err := auth.SystemRoles().FindBy(ctx, map[string]any{"user_id": userID, "role": role})
				if err != nil || len(roles) == 0 {
					w.WriteHeader(http.StatusForbidden)
					fmt.Fprintf(w, "Forbidden: requires system role %s", role)
					return
				}

				// Also check if they are a superadmin to set the view variable for navigation
				superRoles, err := auth.SystemRoles().FindBy(ctx, map[string]any{"user_id": userID, "role": "superadmin"})
				isSystemAdmin := err == nil && len(superRoles) > 0
				r = tracks.AddViewVar(r, "is_system_admin", isSystemAdmin)

				h.ServeHTTP(w, r)
			}), nil
		}
	}
}

func authenticate(domain string, secure bool) tracks.Middleware {
	return func(h http.Handler) (http.Handler, error) {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !session.FromRequest(r).IsAuthenticated() {
				accept := r.Header.Get("Accept")
				if strings.Contains(accept, htmlMediaType) {
					scheme := "http"
					if secure || session.IsSecure(r) {
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

func SystemRoleMiddleware(next http.Handler) (http.Handler, error) {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		sess := session.FromContext(ctx)
		userID, ok := sess.Authenticated()
		if !ok {
			next.ServeHTTP(w, r)
			return
		}

		auth := NewSchema()
		roles, err := auth.SystemRoles().FindBy(ctx, map[string]any{"user_id": userID, "role": "superadmin"})
		isSystemAdmin := err == nil && len(roles) > 0
		r = tracks.AddViewVar(r, "is_system_admin", isSystemAdmin)

		next.ServeHTTP(w, r)
	}), nil
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
		schema: NewSchema(),
	}

	return r.
		GlobalMiddleware(SystemRoleMiddleware).
		// Login screen
		GetFunc("/sessions/new", "sessions", "new", sr.New).
		// Login action
		PostFunc("/sessions/", "sessions", "create", sr.Create).
		GetFunc("/users/", "users", "index", ur.Index).
		// Registration page
		GetFunc("/users/new", "users", "new", ur.New).
		// Registration action
		PostFunc("/users/", "users", "create", ur.Create).
		// Activation page
		GetFunc("/users/activate", "users", "activate", ur.Activate).
		// Activation action
		PostFunc("/users/activate", "users", "set_password_with_token", ur.SetPasswordWithToken).
		//RequestMiddleware(authenticate(r.BaseDomain(), r.Secure())).
		// Logout action
		DeleteFunc("/sessions/", "sessions", "destroy", sr.Delete, WithAnyRole)
}
