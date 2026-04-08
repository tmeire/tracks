# Feature Request: Lightweight Domain-Based Database Routing

**Priority:** High  
**Status:** Open

## Relationship to Multitenancy Module

The multitenancy module provides robust per-tenant database isolation through `TenantRepository`. This is ideal for SaaS applications with registered tenants. However, this request addresses a **lighter-weight** use case:

- No tenant registration/management required
- Automatic database creation per domain
- Works with any domain (not just subdomains)
- Suitable for simple multi-domain landing pages, microsites, etc.

## Current Multitenancy Approach

```go
// Multitenancy requires explicit tenant creation
tenantDB := multitenancy.NewTenantRepository(centralDB, "./data")
tenant, _ := tenantDB.CreateTenant(ctx, "Tenant Name", "subdomain")
// Creates: ./data/Tenants/subdomain/tenant.sqlite
```

This requires:
1. Central database for tenant registry
2. Explicit tenant creation via CLI or API
3. Subdomain-based routing

## Required Functionality

1. **Automatic Database per Domain**: Create DB on first access, no registration needed
2. **Simple Path Mapping**: Domain → sanitized file path
3. **No Central Registry**: No tenant table, no central DB required
4. **Schema Initialization**: Auto-run migrations on new databases
5. **Works with Full Domains**: Not limited to subdomains

## Proposed API

```go
// Lightweight domain DB (no multitenancy overhead)
router.DomainDatabase(tracks.DomainDBConfig{
    DataDir: "./data",
    Driver: "sqlite3",
    SchemaInit: func(db *sql.DB) error {
        // Run schema creation
        return nil
    },
})

// In handlers
func (h *Handler) Create(r *http.Request) (any, error) {
    // Database is automatically domain-scoped
    db := database.FromContext(r.Context())
    // This uses data/{domain}.db automatically
    
    repo := database.NewRepository[WaitlistEntry](schema)
    return repo.Create(r.Context(), entry)
}
```

## Comparison

| Feature | Multitenancy Module | Domain DB Routing |
|---------|-------------------|-------------------|
| Setup | Central DB + tenant registry | None |
| Tenant creation | Required (CLI/API) | Automatic on first access |
| Routing | Subdomain only | Any domain |
| Best for | SaaS with registered tenants | Simple multi-domain sites |
| Example | `tenant.example.com` | `site1.com`, `site2.org` |
| Path | `./data/Tenants/{subdomain}/tenant.sqlite` | `./data/{domain}.db` |
| CLI tools | Yes (create tenant) | No (auto-created) |

## Use Cases

- Multi-domain landing pages (each domain gets own waitlist DB)
- Microsites with isolated data
- White-label apps without tenant management
- Simple SaaS where users bring their own domain
- Development/testing with per-domain isolation

## Acceptance Criteria

- [ ] Domain extraction and sanitization for safe file paths
- [ ] Automatic database creation on first access
- [ ] Per-domain database connections
- [ ] Schema initialization hook
- [ ] Works with any domain (not just subdomains)
- [ ] No central registry required
- [ ] Connection pooling per domain
- [ ] Works with existing repository pattern
- [ ] Documentation and examples

## Database Path Resolution

```go
// Input domains -> Database paths
"superapp.localhost" -> "./data/superapp_localhost.db"
"example.com" -> "./data/example_com.db"
"my-site.org" -> "./data/my-site_org.db"

// Sanitization:
// - Replace non-alphanumeric chars with underscore
// - Prevent directory traversal
// - Normalize to safe filename
```

## Integration Note

This can be used alongside multitenancy for hybrid scenarios:
- Use multitenancy for main SaaS tenant isolation
- Use domain DB routing for landing pages/marketing sites
- Or migrate from domain DB to full multitenancy as needed
