package shttp

// HTTPError represents an HTTP error with a message and status code
type HTTPError struct {
	Message string
	StatusCode int
}

// Error implements the error interface
func (e HTTPError) Error() string {
	return e.Message
}

// NewHTTPError creates a new HTTPError
func NewHTTPError(statusCode int, message string) error {
	return HTTPError{
		Message:    message,
		StatusCode: statusCode,
	}
}

