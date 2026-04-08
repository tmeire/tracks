# Feature Request: Domain-Aware Registration Logic

**Priority:** Medium  
**Status:** Open

## Description

Multi-tenant applications often have per-domain business logic and validation rules. The waitlist system demonstrates domain-aware registration with duplicate checking and conflict responses.

## Current Implementation

```go
func (h *WaitlistHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    domain := middleware.GetDomain(r.Context())
    if domain == "" {
        http.Error(w, "Could not determine domain", http.StatusBadRequest)
        return
    }

    // Check if email already registered for THIS domain
    exists, err := h.db.IsEmailRegistered(domain, req.Email)
    if err != nil {
        respondWithJSON(w, http.StatusInternalServerError, WaitlistResponse{Success: false, Message: "Database error"})
        return
    }
    if exists {
        // Return 409 Conflict for duplicate
        respondWithJSON(w, http.StatusConflict, WaitlistResponse{Success: false, Message: "Email already registered"})
        return
    }

    // Get per-domain count
    count, _ := h.db.GetWaitlistCount(domain)
    
    // Add to waitlist
    _, err = h.db.AddToWaitlist(domain, req.Email, req.Name, req.Metadata)
    // ...
}
```

## Required Functionality

1. **Domain Validation**: Ensure operations are scoped to valid domains
2. **Unique Constraints**: Per-domain uniqueness checking
3. **Conflict Responses**: HTTP 409 status for duplicate resources
4. **Domain Metrics**: Per-domain counts and statistics

## Proposed API

```go
// Define model with domain uniqueness
type WaitlistEntry struct {
    tracks.DomainModel  // Embeds domain field and methods
    ID        int64
    Email     string `validate:"unique_per_domain"`  // Tag for validation
    Name      string
    Metadata  string
}

// Repository with domain-aware validation
repo := database.NewRepository[WaitlistEntry](schema)

// Create checks for duplicates automatically
entry, err := repo.Create(r.Context(), newEntry)
if err != nil {
    if errors.Is(err, database.ErrDuplicate) {
        return tracks.Conflict("Email already registered")
    }
    return err
}

// Or explicit check
exists, err := repo.Exists(r.Context(), map[string]any{"email": email})
```

## Use Cases

- User registration per domain
- Email subscription management
- Product catalogs per tenant
- Domain-specific limits and quotas

## Acceptance Criteria

- [ ] Domain field convention for models
- [ ] Repository methods for domain-scoped uniqueness checks
- [ ] Conflict response helper (HTTP 409)
- [ ] Per-domain counting/aggregation
- [ ] Validation hooks for custom logic
- [ ] Clear error messages indicating domain context
- [ ] Documentation with examples

## Response Helpers

```go
// Convenience functions for common responses
tracks.Conflict(message string) // 409
tracks.BadRequest(err error)    // 400
tracks.NotFound(message string) // 404
tracks.Created(data any)        // 201
```
