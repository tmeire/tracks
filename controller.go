package tracks

import "net/http"

// Action is a function that processes an HTTP request and returns either:
// 1. An opaque data object or a Response object with status and message, and
// 2. An error object that satisfies the Go error interface
//
// If the first return value is an opaque data object (not a Response), the status will be set to OK.
type Action func(r *http.Request) (any, error)
