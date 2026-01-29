package featureflags

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/tmeire/tracks"
	"github.com/tmeire/tracks/database"
	"github.com/tmeire/tracks/modules/multitenancy"
	"github.com/tmeire/tracks/session"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// WithFlags returns middleware that computes effective feature flags for the
// request actor and stores them in the context for quick access.
func WithFlags(r tracks.Router) tracks.Middleware {
	centralDB := r.Database()
	repo := newRepository(centralDB)

	return func(next http.Handler) (http.Handler, error) {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			ctx, span := otel.GetTracerProvider().Tracer("tracks").Start(req.Context(), "featureflags")
			defer span.End()

			// Determine principals
			principals := Principals{}

			// tenant
			if vars := tracks.ViewVars(ctx); vars != nil {
				if t, ok := vars["tenant"].(*multitenancy.Tenant); ok && t != nil {
					tid := toString(t.ID)
					principals.TenantID = &tid
				}
			}

			// user and roles from session
			if s := session.FromContext(ctx); s != nil && s.IsAuthenticated() {
				if uid, ok := s.Authenticated(); ok {
					principals.UserID = &uid
					// Load roles if we have a tenant
					if principals.TenantID != nil {
						roles := loadRoles(ctx, centralDB, uid, *principals.TenantID)
						principals.RoleIDs = roles
					}
				}
			}

			keys := listKeys()
			// Use process cache first
			ck := makeCacheKey(principals)
			if cached, ok := globalCache.get(ck, now()); ok {
				ctx = withFlags(ctx, cached)
				// Add OTel attribute with enabled keys
				addSpanAttrEnabled(span, cached)
				// store in view vars
				req = tracks.AddViewVar(req, "flags", cached)
				next.ServeHTTP(w, req.WithContext(ctx))
				return
			}

			overrides, _ := repo.ListOverrides(ctx, keys, principals)

			// Compute effective values
			effective := computeEffective(keys, overrides)
			// store in cache
			globalCache.set(ck, effective, now())

			// Store in context
			ctx = withFlags(ctx, effective)

			// Add OTel attribute with enabled keys
			addSpanAttrEnabled(span, effective)

			// store in view vars
			fmt.Println("effective flags:", effective)
			req = tracks.AddViewVar(req, "flags", effective)

			next.ServeHTTP(w, req.WithContext(ctx))
		}), nil
	}
}

// computeEffective applies precedence rules to build a full map for all keys
// using the available overrides. Precedence for authenticated: user > role(any) > tenant > global > default.
// For unauthenticated: tenant > global > default. We can infer authenticated by presence of a user override capability.
func computeEffective(keys []string, overrides []Override) map[string]bool {
	// Index overrides by flag and type
	type entry struct {
		has bool
		val bool
	}
	glob := map[string]entry{}
	ten := map[string]entry{}
	role := map[string][]bool{}
	user := map[string]entry{}

	for _, o := range overrides {
		switch o.PrincipalType {
		case PrincipalGlobal:
			glob[o.FlagKey] = entry{true, o.Value}
		case PrincipalTenant:
			ten[o.FlagKey] = entry{true, o.Value}
		case PrincipalRole:
			role[o.FlagKey] = append(role[o.FlagKey], o.Value)
		case PrincipalUser:
			user[o.FlagKey] = entry{true, o.Value}
		}
	}

	out := make(map[string]bool, len(keys))
	for _, k := range keys {
		if e, ok := user[k]; ok && e.has {
			out[k] = e.val
			continue
		}
		if rs, ok := role[k]; ok && len(rs) > 0 {
			// any=true for roles
			anyTrue := false
			for _, v := range rs {
				if v {
					anyTrue = true
					break
				}
			}
			if anyTrue {
				out[k] = true
			} else {
				// all false → false
				out[k] = false
			}
			continue
		}
		if e, ok := ten[k]; ok && e.has {
			out[k] = e.val
			continue
		}
		if e, ok := glob[k]; ok && e.has {
			out[k] = e.val
			continue
		}
		def, _ := getDefault(k)
		out[k] = def
	}
	return out
}

func toString[T ~int | ~int64 | ~uint | ~uint64](n T) string {
	// simple fast itoa for limited types
	return strconvItoa(int64(n))
}

func strconvItoa(n int64) string {
	// minimal allocation; we can delegate to stdlib but avoid extra import churn
	// Implement via fmt if simplicity preferred
	// We'll just use fmt package from repo.go, but here keep light: import fmt here
	return fmtInt(n)
}

// small helper using fmt to avoid pulling strconv separately (already used elsewhere)
func fmtInt(n int64) string { return fmt.Sprintf("%d", n) }

// loadRoles returns role IDs for a user in a tenant from central DB
func loadRoles(ctx context.Context, db database.Database, userID string, tenantID string) []string {
	rows, err := db.QueryContext(ctx, `SELECT role FROM user_roles WHERE user_id=? AND tenant_id=?`, userID, tenantID)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var roles []string
	for rows.Next() {
		var role string
		if err := rows.Scan(&role); err == nil {
			roles = append(roles, role)
		}
	}
	return roles
}

// helper to assemble span attribute from a flags map
func addSpanAttrEnabled(span trace.Span, m map[string]bool) {
	enabled := make([]string, 0, len(m))
	for k, v := range m {
		if v {
			enabled = append(enabled, k)
		}
	}
	sort.Strings(enabled)
	if len(enabled) > 0 {
		span.SetAttributes(attribute.String("featureflags.enabled", strings.Join(enabled, ",")))
	}
}

// now is a testable time provider
var now = func() time.Time { return time.Now() }
