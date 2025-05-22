package tracks

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
}
