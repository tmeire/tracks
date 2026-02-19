package tracks

import (
	"context"
	"net"
	"net/http"
)

type domainKey struct{}

// DomainFromContext returns the full domain stored in the context, or an empty string if not found.
func DomainFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if domain, ok := ctx.Value(domainKey{}).(string); ok {
		return domain
	}
	return ""
}

// DomainMiddleware extracts the full domain from the Host header, stripping any port number.
// It stores the domain in the request context and makes it available in templates as .Domain.
func DomainMiddleware() Middleware {
	return func(next http.Handler) (http.Handler, error) {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			host := r.Host
			domain, _, err := net.SplitHostPort(host)
			if err != nil {
				// If SplitHostPort fails, it might be because there's no port.
				domain = host
			}

			// Store domain in context
			ctx := context.WithValue(r.Context(), domainKey{}, domain)
			r = r.WithContext(ctx)

			// Store domain in view variables for templates
			r = AddViewVar(r, "Domain", domain)

			next.ServeHTTP(w, r)
		}), nil
	}
}
