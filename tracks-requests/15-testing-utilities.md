# Feature Request: Testing Utilities & Helpers

**Priority:** Medium  
**Status:** Open

## Description

The framework lacks comprehensive testing utilities, making it harder to write unit and integration tests for tracks applications.

## Current Gap

No dedicated test helpers for:
- HTTP request/response testing
- Database testing (fixtures, transactions)
- Session testing
- Mail testing (already has `driver_testing.go`)
- Authentication testing

## Required Functionality

1. **Test Router**: Create testable router instances
2. **HTTP Testing**: Helpers for making requests and assertions
3. **Database Fixtures**: Load test data from YAML/JSON files
4. **Transactional Tests**: Rollback database changes after each test
5. **Session Testing**: Create authenticated sessions for tests
6. **Mail Testing**: Assert emails sent (already exists, needs integration)
7. **Time Mocking**: Control time in tests
8. **Assert Helpers**: Common HTTP assertions

## Proposed API

```go
// Test setup
func TestWaitlistHandler(t *testing.T) {
    // Create test application
    app := tracks.NewTestApp(t, tracks.TestConfig{
        Database: "sqlite::memory:",
        Fixtures: "testdata/fixtures.yml",
    })
    
    // Make request
    resp := app.Get("/api/waitlist")
    assert.Equal(t, http.StatusOK, resp.StatusCode)
    
    // With JSON body
    resp = app.Post("/api/waitlist", tracks.JSONBody{
        "email": "test@example.com",
        "name": "Test User",
    })
    assert.Equal(t, http.StatusCreated, resp.StatusCode)
    
    // Assert database state
    var count int
    app.DB().QueryRow("SELECT COUNT(*) FROM waitlist").Scan(&count)
    assert.Equal(t, 1, count)
}

// With authentication
func TestAuthenticatedEndpoint(t *testing.T) {
    app := tracks.NewTestApp(t)
    
    // Create and authenticate user
    user := app.CreateUser("test@example.com", "password")
    app.AuthenticateAs(user)
    
    resp := app.Get("/users/profile")
    assert.Equal(t, http.StatusOK, resp.StatusCode)
}

// Transactional test - auto rollback
func TestWithTransaction(t *testing.T) {
    app := tracks.NewTestApp(t, tracks.TestConfig{
        Transactional: true, // Auto-rollback after test
    })
    
    // Make changes
    app.Post("/api/waitlist", tracks.JSONBody{"email": "test@test.com"})
    
    // Changes automatically rolled back after test
}

// Mail assertions
func TestEmailSent(t *testing.T) {
    app := tracks.NewTestApp(t)
    mailer := mail.TestingDriver()
    
    app.Post("/users", tracks.JSONBody{
        "email": "new@example.com",
        "password": "password123",
    })
    
    // Assert welcome email sent
    assert.Equal(t, 1, mailer.SentCount())
    assert.Contains(t, mailer.Last().Subject, "Welcome")
    assert.Equal(t, "new@example.com", mailer.Last().To[0])
}

// Time mocking
func TestTimeBasedLogic(t *testing.T) {
    app := tracks.NewTestApp(t)
    
    // Freeze time
    app.FreezeTime(time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC))
    
    // Run code that uses time.Now()
    
    // Travel forward
    app.Travel(24 * time.Hour)
}
```

## Use Cases

- Handler/controller testing
- Integration testing
- API endpoint testing
- Database state assertions
- Authentication flow testing
- Email sending assertions

## Acceptance Criteria

- [ ] TestApp type for application testing
- [ ] HTTP request helpers (Get, Post, Put, Delete, Patch)
- [ ] Response assertion helpers
- [ ] Database fixture loading
- [ ] Transactional test support
- [ ] Authentication helpers
- [ ] Time mocking utilities
- [ ] Mail testing integration
- [ ] Works with existing Go testing framework
- [ ] Documentation and examples

## Fixture Format (YAML)

```yaml
# testdata/fixtures.yml
users:
  - id: "user-1"
    email: "admin@example.com"
    name: "Admin User"
    created_at: "2025-01-01T00:00:00Z"

waitlist:
  - id: 1
    email: "existing@example.com"
    name: "Existing User"
    domain: "example.com"
    created_at: "2025-01-01T00:00:00Z"
```
