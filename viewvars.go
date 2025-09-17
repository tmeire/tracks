package tracks

import (
	"context"
	"net/http"
)

// viewVarsKey is an unexported type used as the context key for view variables.
// Using a distinct type avoids collisions with other context values.
type viewVarsKey struct{}

// ViewVars returns the map of variables attached to the given context. It returns
// nil if no variables map has been associated yet.
func ViewVars(ctx context.Context) map[string]any {
	if ctx == nil {
		return nil
	}
	if m, ok := ctx.Value(viewVarsKey{}).(map[string]any); ok {
		return m
	}
	return nil
}

// AddViewVar ensures the request has an initialized view vars map and adds or
// overwrites the provided key with the given value. It returns a new request
// instance that carries the updated context, so it should be used for subsequent
// handler calls: r = tracks.AddViewVar(r, "key", value)
func AddViewVar(r *http.Request, key string, value any) *http.Request {
	ctx := r.Context()

	vars := ViewVars(ctx)
	if vars == nil {
		vars = make(map[string]any)
		ctx = context.WithValue(ctx, viewVarsKey{}, vars)
	}
	vars[key] = value

	return r.WithContext(ctx)
}
