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

// Conflict returns a 409 Conflict response.
func Conflict(message string) *Response {
	return &Response{
		StatusCode: http.StatusConflict,
		Data: map[string]any{
			"success": false,
			"message": message,
		},
	}
}

// BadRequest returns a 400 Bad Request response.
func BadRequest(err error) *Response {
	return &Response{
		StatusCode: http.StatusBadRequest,
		Data: map[string]any{
			"success": false,
			"message": err.Error(),
		},
	}
}

// NotFound returns a 404 Not Found response.
func NotFound(message string) *Response {
	return &Response{
		StatusCode: http.StatusNotFound,
		Data: map[string]any{
			"success": false,
			"message": message,
		},
	}
}

// Created returns a 201 Created response.
func Created(data any) *Response {
	return &Response{
		StatusCode: http.StatusCreated,
		Data:       data,
	}
}

// Forbidden returns a 403 Forbidden response.
func Forbidden(message string) *Response {
	return &Response{
		StatusCode: http.StatusForbidden,
		Data: map[string]any{
			"success": false,
			"message": message,
		},
	}
}

// Unauthorized returns a 401 Unauthorized response.
func Unauthorized(message string) *Response {
	return &Response{
		StatusCode: http.StatusUnauthorized,
		Data: map[string]any{
			"success": false,
			"message": message,
		},
	}
}

// InternalServerError returns a 500 Internal Server Error response.
func InternalServerError(err error) *Response {
	return &Response{
		StatusCode: http.StatusInternalServerError,
		Data: map[string]any{
			"success": false,
			"message": err.Error(),
		},
	}
}
