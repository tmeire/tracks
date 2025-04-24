package tracks

// Response represents a standardized API response structure
type Response struct {
	// StatusCode is the HTTP status code to be returned
	StatusCode int
	// Location is the URL to redirect to after the request is completed.
	Location string
	// Data is the payload to be returned to the client
	Data interface{}
}
