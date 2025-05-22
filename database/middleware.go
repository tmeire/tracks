package database

import (
	"net/http"
)

func Middleware(db Database) func(handler http.Handler) http.Handler {
	return func(handler http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			req = req.WithContext(WithDB(req.Context(), db))

			handler.ServeHTTP(w, req)
		})
	}
}
