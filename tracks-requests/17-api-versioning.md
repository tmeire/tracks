# Feature Request: API Versioning Support

**Priority:** Low  
**Status:** Open

## Description

The framework lacks built-in support for API versioning, which is essential for maintaining backward compatibility as APIs evolve.

## Current Gap

No standard mechanism for:
- URL path versioning (`/api/v1/users`)
- Header-based versioning (`Accept: application/vnd.api.v1+json`)
- Parameter-based versioning (`?version=1`)
- Deprecation warnings

## Required Functionality

1. **Version Routing**: Route requests to different handlers based on version
2. **URL Path Versioning**: `/api/v1/*`, `/api/v2/*`
3. **Header Versioning**: Content negotiation via Accept header
4. **Query Parameter**: `?api-version=2`
5. **Default Version**: Fallback when no version specified
6. **Deprecation**: Mark versions as deprecated with sunset dates
7. **Response Headers**: API-Version, Deprecation, Sunset headers

## Proposed API

```go
// URL path versioning
api := router.Subrouter("/api")

v1 := api.Version("v1")
v1.GetFunc("/users", "users", "index", v1IndexHandler)
v1.PostFunc("/users", "users", "create", v1CreateHandler)

v2 := api.Version("v2")
v2.GetFunc("/users", "users", "index", v2IndexHandler) // Different implementation
v2.PostFunc("/users", "users", "create", v2CreateHandler)

// Header-based versioning
api.VersionFromHeader("Accept", "application/vnd.myapp.v1+json", v1Handler)
api.VersionFromHeader("Accept", "application/vnd.myapp.v2+json", v2Handler)

// Query parameter versioning
api.VersionFromQuery("api-version", "1", v1Handler)
api.VersionFromQuery("api-version", "2", v2Handler)

// With deprecation
v1 := api.Version("v1", tracks.VersionConfig{
    Deprecated: true,
    SunsetDate: time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC),
    MigrationGuide: "https://docs.example.com/migration/v2",
})

// Response will include:
// Deprecation: true
// Sunset: Sat, 31 Dec 2025 00:00:00 GMT
// Link: <https://docs.example.com/migration/v2>; rel="migration"
```

## Version Handler Example

```go
// v1 handler
type UserV1 struct {
    ID    string `json:"id"`
    Name  string `json:"name"`
    Email string `json:"email"`
}

func v1IndexHandler(r *http.Request) (any, error) {
    users := []UserV1{
        {ID: "1", Name: "John", Email: "john@example.com"},
    }
    return users, nil
}

// v2 handler with different response format
type UserV2 struct {
    ID        string    `json:"id"`
    FirstName string    `json:"first_name"`
    LastName  string    `json:"last_name"`
    Email     string    `json:"email"`
    CreatedAt time.Time `json:"created_at"`
}

func v2IndexHandler(r *http.Request) (any, error) {
    users := []UserV2{
        {ID: "1", FirstName: "John", LastName: "Doe", Email: "john@example.com", CreatedAt: time.Now()},
    }
    return users, nil
}
```

## Use Cases

- Maintaining backward compatibility
- Gradual API migrations
- Third-party integrations
- Mobile app compatibility
- Breaking change management

## Acceptance Criteria

- [ ] URL path versioning support
- [ ] Header-based content negotiation
- [ ] Query parameter versioning
- [ ] Configurable default version
- [ ] Deprecation headers
- [ ] Sunset date headers
- [ ] Migration guide links
- [ ] Version in response headers
- [ ] Documentation and examples
- [ ] Migration guide documentation

## HTTP Headers

**Request:**
```
GET /api/users
Accept: application/vnd.myapp.v2+json
```

**Response (deprecated version):**
```
HTTP/1.1 200 OK
Content-Type: application/json
API-Version: v1
Deprecation: true
Sunset: Sat, 31 Dec 2025 00:00:00 GMT
Link: <https://docs.example.com/migration/v2>; rel="migration"

{"users": [...]}
```
