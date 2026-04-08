# Feature Request: Domain-Based Template Overrides

**Priority:** High  
**Status:** Open

## Relationship to Multitenancy Module

The multitenancy module provides tenant-specific views via:

```go
// From module.go
rn := r.Clone().Views("./views/tenants")
```

However, this is limited to:
- Tenants registered in the database
- Subdomain-based routing only
- Single views directory for all tenants

This request extends template selection to work with **any domain** and **fallback mechanisms**.

## Current Multitenancy Limitation

```go
// Multitenancy module
// - Only works if tenant exists in DB
// - Only works for subdomains
// - No automatic fallback to default templates
```

## Required Functionality

1. **Domain-Specific Templates**: Check for `views/domains/{domain}/` directory first
2. **Automatic Fallback**: Fall back to default templates if domain-specific not found
3. **No Database Required**: Works with any domain, no tenant lookup needed
4. **Layout Integration**: Domain templates use same layouts as main site
5. **Hot Reloading**: Development mode file watching (optional)

## Proposed API

```go
// Enable domain-aware template loading
router.DomainViews("./views")

// Template resolution order:
// 1. views/domains/{domain}/{controller}/{action}.gohtml
// 2. views/{controller}/{action}.gohtml
```

## File Structure

```
views/
├── layouts/
│   └── application.gohtml
├── pages/
│   ├── index.gohtml      # Default
│   └── about.gohtml      # Default
└── domains/
    ├── superapp.localhost/
    │   └── pages/
    │       └── index.gohtml  # Override for superapp
    └── example.com/
        └── pages/
            └── index.gohtml  # Override for example.com
```

## Comparison

| Feature | Multitenancy Views | Domain Template Overrides |
|---------|-------------------|--------------------------|
| Directory | `./views/tenants/` | `./views/domains/{domain}/` |
| Database required | Yes (tenant lookup) | No |
| Routing | Subdomain only | Any domain |
| Fallback | No | Yes (to default templates) |
| Use case | SaaS tenants | Landing pages, microsites |

## Use Cases

- Landing pages per domain with custom branding
- White-label pages without tenant management
- A/B testing different templates per domain
- Marketing microsites
- Domain-specific error pages

## Acceptance Criteria

- [ ] Domain-specific template directory structure
- [ ] Automatic fallback to default templates
- [ ] Works with existing layout system
- [ ] Template functions available in domain templates
-- [ ] View variables accessible
- [ ] No database lookup required
- [ ] Works independently of multitenancy
- [ ] Clear error messages when templates missing
- [ ] Documentation and examples

## Template Resolution Flow

```
Request: GET / (Host: superapp.localhost)

1. Check views/domains/superapp.localhost/pages/index.gohtml
2. If exists: Render with layout
3. If not exists: Check views/pages/index.gohtml
4. Render default template with layout
```

## Integration with Multitenancy

Can be used together for advanced scenarios:

```go
// Domain overrides for any domain
router.DomainViews("./views")

// Plus tenant-specific views for registered tenants
router.Module(multitenancy.Register)

// Resolution order:
// 1. views/domains/{domain}/... (any domain)
// 2. views/tenants/... (registered tenants via multitenancy)
// 3. views/... (default)
```
