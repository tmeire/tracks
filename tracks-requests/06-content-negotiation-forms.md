# Feature Request: Content Negotiation for Form Submissions

**Priority:** Medium  
**Status:** Open

## Description

API endpoints should accept both JSON and form-encoded POST data to support both AJAX requests and traditional form submissions. The existing waitlist handler demonstrates this pattern.

## Current Implementation

```go
func (h *WaitlistHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    var req WaitlistRequest
    contentType := r.Header.Get("Content-Type")

    if strings.Contains(contentType, "application/json") {
        // Parse JSON body
        if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
            respondWithJSON(w, http.StatusBadRequest, WaitlistResponse{Success: false, Message: "Invalid JSON"})
            return
        }
    } else {
        // Parse form data
        if err := r.ParseForm(); err != nil {
            respondWithJSON(w, http.StatusBadRequest, WaitlistResponse{Success: false, Message: "Invalid form data"})
            return
        }
        req.Email = r.FormValue("email")
        req.Name = r.FormValue("name")
        req.Metadata = r.FormValue("metadata")
    }
    // ... rest of handler
}
```

## Required Functionality

1. **Automatic Content Type Detection**: Parse request body based on Content-Type header
2. **Unified Struct Population**: Populate the same struct regardless of encoding
3. **Validation Support**: Consistent validation across both formats
4. **Error Handling**: Appropriate error responses for each content type

## Proposed API

```go
// Define request struct
type WaitlistRequest struct {
    Email    string `json:"email" form:"email" validate:"required,email"`
    Name     string `json:"name" form:"name"`
    Metadata string `json:"metadata" form:"metadata"`
}

// In handler - automatic parsing
func (h *Handler) Create(r *http.Request) (any, error) {
    var req WaitlistRequest
    if err := tracks.ParseRequest(r, &req); err != nil {
        return nil, tracks.BadRequest(err)
    }
    // req is populated from JSON or form data automatically
    // ...
}

// Or with explicit type
data, err := tracks.ParseJSONOrForm[WaitlistRequest](r)
```

## Use Cases

- Progressive enhancement (forms work without JS)
- API endpoints that support both AJAX and standard submissions
- Webhook endpoints receiving different formats
- Backward compatibility during API migrations

## Acceptance Criteria

- [ ] Helper function to parse request based on Content-Type
- [ ] Support for `application/json`
- [ ] Support for `application/x-www-form-urlencoded`
- [ ] Support for `multipart/form-data` (optional)
- [ ] Struct tag support (`json:` and `form:`)
- [ ] Consistent error handling
- [ ] Works with validation libraries
- [ ] Documentation and examples

## Content Type Priority

1. Check `Content-Type` header
2. If JSON, parse as JSON
3. If form-urlencoded or multipart, parse as form
4. If missing, attempt JSON first, then form as fallback
