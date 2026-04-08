# Feature Request: CSRF Protection Middleware

**Priority:** High  
**Status:** Open

## Description

The framework lacks built-in CSRF (Cross-Site Request Forgery) protection, which is essential for form submissions in web applications.

## Current Gap

Forms submit data without CSRF tokens, making them vulnerable to CSRF attacks. No middleware or helper functions exist for:
- Generating CSRF tokens
- Validating tokens on POST/PUT/DELETE requests
- Excluding specific routes from CSRF checks (API endpoints)

## Required Functionality

1. **Token Generation**: Generate cryptographically secure CSRF tokens per session
2. **Token Storage**: Store tokens in session or secure cookies
3. **Template Helper**: Helper to embed tokens in forms: `{{ csrf_token }}`
4. **Validation Middleware**: Automatically validate tokens on state-changing requests
5. **Exemptions**: Allowlist certain routes/paths from CSRF validation
6. **Error Handling**: Clear error responses for missing/invalid tokens

## Proposed API

```go
// Router configuration
router.CSRFProtection(tracks.CSRFConfig{
    TokenLength: 32,
    CookieName: "csrf_token",
    HeaderName: "X-CSRF-Token",
    ExemptPaths: []string{"/api/webhooks/*", "/api/public/*"},
})

// In templates
<form method="POST" action="/waitlist">
    <input type="hidden" name="csrf_token" value="{{ csrf_token }}">
    <!-- form fields -->
</form>

// Or use a helper
{{ csrf_field }}

// JavaScript fetch helper
fetch('/api/waitlist', {
    method: 'POST',
    headers: {
        'X-CSRF-Token': document.querySelector('meta[name="csrf-token"]').content
    }
})
```

## Use Cases

- Form submissions in web applications
- AJAX requests requiring protection
- Multi-step workflows
- Admin interfaces

## Acceptance Criteria

- [ ] CSRF token generation and storage
- [ ] Middleware for automatic validation
- [ ] Template helper for forms
- [ ] Meta tag helper for JavaScript access
- [ ] Configurable exempt paths
- [ ] Configurable token lifetime
- [ ] Works with existing session system
- [ ] Documentation and examples

## Security Considerations

- Tokens must be cryptographically random
- Double-submit cookie pattern or session storage
- Secure defaults (all POST/PUT/PATCH/DELETE protected)
- Proper SameSite cookie attributes
- Token rotation on authentication
