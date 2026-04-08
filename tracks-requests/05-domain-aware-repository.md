# Feature Request: Domain-Aware Repository Filtering

**Priority:** High  
**Status:** Open

## Relationship to Multitenancy Module

The multitenancy module provides per-tenant database isolation through `TenantRepository.GetTenantDB()`, which injects the tenant-specific database into the context. Queries then automatically use that database.

However, **data within** the tenant database is not automatically filtered by domain. This request addresses automatic domain scoping for queries within a database.

## Current Multitenancy Approach

```go
// Multitenancy provides DB isolation:
// tenant1.example.com -> ./data/Tenants/tenant1/tenant.sqlite
// tenant2.example.com -> ./data/Tenants/tenant2/tenant.sqlite

// But within the database, queries are not automatically scoped:
repo.FindAll(ctx)  // Returns ALL records, no domain filter
```

## The Problem

Even with separate databases, you may want domain-scoped data within a shared database:

```go
// Same database, but data should be filtered by domain
site1.com -> data/site1_com.db -> waitlist table (site1 entries only)
site2.com -> data/site2_com.db -> waitlist table (site2 entries only)

// OR within same database:
// waitlist table has 'domain' column
// Need automatic filtering: SELECT * FROM waitlist WHERE domain = ?
```

## Required Functionality

1. **Automatic Domain Filtering**: Repository queries filter by current domain
2. **Domain Field Convention**: Standard pattern for domain-scoped models
3. **Cross-Domain Queries**: Option to query across domains (admin functions)
4. **Works With or Without Multitenancy**: Independent of tenant module

## Proposed API

```go
// Option 1: Domain-scoped model interface
type WaitlistEntry struct {
    tracks.DomainScopedModel  // Embeds Domain field
    ID        int64
    Email     string
    Name      string
}

// Repository automatically adds WHERE domain = ?
repo := database.NewRepository[WaitlistEntry](schema)
entries, err := repo.FindAll(r.Context())  // SELECT * FROM waitlist WHERE domain = 'site1.com'

// Option 2: Explicit scoping
repo.WithDomain(domain).FindAll(ctx)

// Option 3: Global filter
repo.DomainScoped(true).FindAll(ctx)
```

## Use Cases

- Waitlist entries per domain
- User registrations per domain
- Domain-specific settings/configuration
- Analytics data per domain
- Works with domain DB routing (request #02) or multitenancy

## Acceptance Criteria

- [ ] Convention for domain field in models
- [ ] Repository automatically applies domain filter
- [ ] Works with all query methods (FindAll, FindBy, FindByID, etc.)
- [ ] Support for per-domain unique constraints (email unique per domain)
- [ ] Option to disable filtering for admin queries
- [ ] Clear documentation on domain scoping
- [ ] Works with or without multitenancy module
- [ ] Integration examples with domain DB routing

## Database Schema

```sql
-- Domain-scoped table
CREATE TABLE waitlist (
    id INTEGER PRIMARY KEY,
    domain TEXT NOT NULL,  -- Domain field for filtering
    email TEXT NOT NULL,
    name TEXT,
    created_at DATETIME,
    UNIQUE(domain, email)  -- Unique per domain
);

CREATE INDEX idx_waitlist_domain ON waitlist(domain);
```

## Comparison with Multitenancy

| Aspect | Multitenancy DB Isolation | Domain-Aware Repository |
|--------|--------------------------|------------------------|
| Level | Separate databases | Within database |
| Use case | Full tenant isolation | Domain-scoped data |
| Schema | Can differ per tenant | Same schema |
| Overhead | High (multiple DBs) | Low (column filter) |
| Best for | SaaS tenants | Landing pages, microsites |

## Integration

```go
// Can be combined with domain DB routing:
router.DomainDatabase(config)  // Separate DB per domain
    .DomainScopedRepositories()  // + domain filtering within DB

// Or with multitenancy:
router.Module(multitenancy.Register)  // Tenant DB isolation
    .DomainScopedRepositories()  // + domain column filtering
```
