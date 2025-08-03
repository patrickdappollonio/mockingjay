package server

import "fmt"

// ServerError represents an error that occurred in the HTTP server
type ServerError struct {
	StatusCode int    // HTTP status code to return
	Message    string // Human-readable error message
	Cause      error  // The underlying error
}

func (e *ServerError) Error() string {
	return fmt.Sprintf("server error (%d): %s", e.StatusCode, e.Message)
}

// Unwrap allows errors.Is and errors.As to work with ServerError
func (e *ServerError) Unwrap() error {
	return e.Cause
}

// RouteError represents an error that occurred during route processing
type RouteError struct {
	Path    string // The request path that caused the error
	Method  string // The HTTP method
	Message string // Human-readable error message
	Cause   error  // The underlying error
}

func (e *RouteError) Error() string {
	return fmt.Sprintf("route error for %s %s: %s", e.Method, e.Path, e.Message)
}

// Unwrap allows errors.Is and errors.As to work with RouteError
func (e *RouteError) Unwrap() error {
	return e.Cause
}

// NewServerError creates a new ServerError
func NewServerError(statusCode int, message string, cause error) *ServerError {
	return &ServerError{
		StatusCode: statusCode,
		Message:    message,
		Cause:      cause,
	}
}

// NewRouteError creates a new RouteError
func NewRouteError(path, method, message string, cause error) *RouteError {
	return &RouteError{
		Path:    path,
		Method:  method,
		Message: message,
		Cause:   cause,
	}
}
