# Feature Request: General Domain Context Middleware (Beyond Multitenancy)

**Priority:** High  
**Status:** Open

## Relationship to Multitenancy Module

The multitenancy module provides subdomain extraction (`extractSubdomain`) for tenant-based routing (e.g., `tenant.example.com`). However, this request addresses a broader need: capturing the **full domain** context for arbitrary domain-based logic, including:
- Full custom domains (e.g., `customer1.com`, `customer2.org`)
- Domain-based feature flags
- Analytics/logging per domain
- Domain-specific configuration without tenant database

## Current Implementation

The multitenancy module handles subdomain extraction:

```go
// From module.go
func extractSubdomain(host string) string {
    host, _, err := net.SplitHostPort(host)
    // ...
    if len(parts) >= 3 {
        return parts[0]  // Returns "tenant" from "tenant.example.com"
    }
    return ""  // No subdomain found
}
```

This works for subdomains but doesn't capture the full domain for custom domain scenarios.

## Required Functionality

1. **Full Domain Extraction**: Extract complete domain from Host header
2. **Port Stripping**: Remove port numbers (already done in multitenancy)
3. **Context Storage**: Store domain in request context (not just subdomain)
4. **Template Access**: Make domain available in templates via `{{ .Domain }}`
5. **Helper Function**: Simple `tracks.DomainFromContext(r.Context())` accessor

## Proposed API

```go
// Simple domain middleware (can be used with or without multitenancy)
router.DomainMiddleware()  // Extracts and stores full domain

// In handlers
domain := tracks.DomainFromContext(r.Context())  // "customer1.com" or "tenant.example.com"

// In templates
{{ .Domain }}  // Full domain
{{ .Subdomain }}  // If using multitenancy
```

## Key Differences from Multitenancy

| Feature | Multitenancy Module | Domain Context Middleware |
|---------|-------------------|--------------------------|
| Extracts | Subdomain only (first part) | Full domain |
| Database lookup | Required (tenant lookup) | Optional |
| Use case | Tenant isolation | Domain-aware logic |
| Example input | `tenant.example.com` | `tenant.example.com` or `custom.com` |
| Example output | `tenant` | `tenant.example.com` or `custom.com` |

## Use Cases

- Custom domain support (e.g., `mybrand.com` vs `mybrand.io`)
- Domain-specific configuration
- Analytics per domain
- A/B testing by domain
- Works alongside multitenancy for subdomain scenarios

## Acceptance Criteria

- [ ] Middleware extracts full domain from Host header
- [ ] Port numbers stripped automatically
- [ ] Domain available in request context
- [ ] Helper function for easy access
- [ ] Available in templates as `{{ .Domain }}`
- [ ] Works independently of multitenancy module
- [ ] Can be combined with multitenancy (both domain and subdomain available)
- [ ] Documentation and examples

## Integration with Multitenancy

```go
// Using both together
router.DomainMiddleware()  // Sets {{ .Domain }}
router.Module(multitenancy.Register)  // Sets {{ .Subdomain }}, {{ .Tenant }}, tenant DB

// In template:
// {{ .Domain }} = "tenant.example.com"
// {{ .Subdomain }} = "tenant"
// {{ .Tenant.Name }} = "Tenant Name"
```
