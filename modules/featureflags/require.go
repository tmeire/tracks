package featureflags

import (
	"net/http"

	"github.com/tmeire/tracks"
)

// RequireFlag returns a middleware that responds with 404 when the given
// feature flag is disabled for the current request context.
//
// Use this to hide routes behind feature flags while keeping controllers simple.
func RequireFlag(key string) tracks.MiddlewareBuilder {
	return func(r tracks.Router) tracks.Middleware {
		return func(next http.Handler) (http.Handler, error) {
			return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				if !Enabled(req.Context(), key) {
					http.NotFound(w, req)
					return
				}
				next.ServeHTTP(w, req)
			}), nil
		}
	}
}
