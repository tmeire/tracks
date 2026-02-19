package tracks

import "net/http"

// Response represents a standardized API response structure
type Response struct {
	// StatusCode is the HTTP status code to be returned
	StatusCode int
	// Location is the URL to redirect to after the request is completed.
	Location string
	// Title is the title of the page to be displayed to the user
	Title string
	// Data is the payload to be returned to the client
	Data interface{}
	// Cookies is a list of cookies to be set on the response
	Cookies []*http.Cookie
}

// ErrorData represents a standardized error response payload
type ErrorData struct {
	Success bool                `json:"success"`
	Message string              `json:"message"`
	Code    string              `json:"code,omitempty"`
	Errors  map[string][]string `json:"errors,omitempty"`
}

// AppError represents an application-specific error with HTTP status code
type AppError struct {
	StatusCode int
	Code       string
	Message    string
	Details    map[string]any
}

func (e AppError) Error() string { return e.Message }

// Conflict returns a 409 Conflict response.
func Conflict(message string) *Response {
	return &Response{
		StatusCode: http.StatusConflict,
		Data: ErrorData{
			Success: false,
			Message: message,
			Code:    "CONFLICT",
		},
	}
}

// BadRequest returns a 400 Bad Request response.
func BadRequest(err error) *Response {
	return &Response{
		StatusCode: http.StatusBadRequest,
		Data: ErrorData{
			Success: false,
			Message: err.Error(),
			Code:    "BAD_REQUEST",
		},
	}
}

// NotFound returns a 404 Not Found response.
func NotFound(message string) *Response {
	return &Response{
		StatusCode: http.StatusNotFound,
		Data: ErrorData{
			Success: false,
			Message: message,
			Code:    "NOT_FOUND",
		},
	}
}

// Forbidden returns a 403 Forbidden response.
func Forbidden(message string) *Response {
	return &Response{
		StatusCode: http.StatusForbidden,
		Data: ErrorData{
			Success: false,
			Message: message,
			Code:    "FORBIDDEN",
		},
	}
}

// Unauthorized returns a 401 Unauthorized response.
func Unauthorized(message string) *Response {
	return &Response{
		StatusCode: http.StatusUnauthorized,
		Data: ErrorData{
			Success: false,
			Message: message,
			Code:    "UNAUTHORIZED",
		},
	}
}

// UnprocessableEntity returns a 422 Unprocessable Entity response.
func UnprocessableEntity(message string, errors map[string][]string) *Response {
	return &Response{
		StatusCode: http.StatusUnprocessableEntity,
		Data: ErrorData{
			Success: false,
			Message: message,
			Code:    "VALIDATION_ERROR",
			Errors:  errors,
		},
	}
}

// InternalServerError returns a 500 Internal Server Error response.
func InternalServerError(err error) *Response {
	return &Response{
		StatusCode: http.StatusInternalServerError,
		Data: ErrorData{
			Success: false,
			Message: err.Error(),
			Code:    "INTERNAL_SERVER_ERROR",
		},
	}
}

// Created returns a 201 Created response.
