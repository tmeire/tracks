# Feature Request: Error Response Customization

**Priority:** Low  
**Status:** Open

## Description

APIs need consistent, customizable error responses that can return different formats (JSON, HTML) based on the request. The existing project returns structured JSON errors with specific status codes.

## Current Implementation

```go
// Custom error response structure
type WaitlistResponse struct {
    Success bool   `json:"success"`
    Message string `json:"message"`
}

// Helper for JSON responses
func respondWithJSON(w http.ResponseWriter, status int, data interface{}) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(status)
    json.NewEncoder(w).Encode(data)
}

// Usage examples
respondWithJSON(w, http.StatusConflict, WaitlistResponse{Success: false, Message: "Email already registered"})
respondWithJSON(w, http.StatusBadRequest, WaitlistResponse{Success: false, Message: "Email is required"})
respondWithJSON(w, http.StatusInternalServerError, WaitlistResponse{Success: false, Message: "Database error"})
```

## Current Tracks Response

```go
// response.go
type Response struct {
    StatusCode int
    Data       any
    Location   string
    Cookies    []*http.Cookie
    Title      string
}
```

## Required Functionality

1. **Error Response Helpers**: Convenience functions for common HTTP error codes
2. **Consistent Format**: Standard error response structure
3. **Content Negotiation**: JSON for API requests, HTML for browser requests
4. **Custom Error Types**: Application-specific error types with metadata

## Proposed API

```go
// Error response helpers
func Conflict(message string) *Response {
    return &Response{
        StatusCode: http.StatusConflict,
        Data: ErrorData{Success: false, Message: message},
    }
}

func BadRequest(err error) *Response {
    return &Response{
        StatusCode: http.StatusBadRequest,
        Data: ErrorData{Success: false, Message: err.Error(), Errors: validationErrors(err)},
    }
}

func NotFound(message string) *Response {
    return &Response{
        StatusCode: http.StatusNotFound,
        Data: ErrorData{Success: false, Message: message},
    }
}

// Custom error type
type AppError struct {
    Code       string
    Message    string
    StatusCode int
    Details    map[string]any
}

func (e AppError) Error() string { return e.Message }
```

## Use Cases

- RESTful API error responses
- Form validation errors
- Business logic errors (duplicates, constraints)
- Consistent error format across application

## Acceptance Criteria

- [ ] Helper functions for common HTTP errors (400, 401, 403, 404, 409, 422, 500)
- [ ] Consistent error response structure
- [ ] Content negotiation (JSON vs HTML)
- [ ] Custom error type support
- [ ] Validation error aggregation
- [ ] Works with existing Response type
- [ ] Documentation and examples

## Error Response Format

```json
{
  "success": false,
  "message": "Validation failed",
  "code": "VALIDATION_ERROR",
  "errors": {
    "email": ["Email is required", "Invalid email format"],
    "name": ["Name is too short"]
  }
}
```
