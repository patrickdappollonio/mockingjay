package config

import (
	"fmt"
)

// ValidationError represents a configuration validation error
type ValidationError struct {
	Field   string // The field that failed validation
	Message string // Human-readable error message
}

func (e *ValidationError) Error() string {
	if e.Field != "" {
		return fmt.Sprintf("validation error in field %q: %s", e.Field, e.Message)
	}
	return fmt.Sprintf("validation error: %s", e.Message)
}

// Unwrap allows errors.Is and errors.As to work with ValidationError
func (e *ValidationError) Unwrap() error {
	return nil
}

// LoadError represents an error that occurred while loading configuration
type LoadError struct {
	Filename string // The filename that failed to load
	Cause    error  // The underlying error
}

func (e *LoadError) Error() string {
	if e.Filename != "" {
		return fmt.Sprintf("failed to load config from %q: %v", e.Filename, e.Cause)
	}
	return fmt.Sprintf("failed to load config: %v", e.Cause)
}

// Unwrap allows errors.Is and errors.As to work with LoadError
func (e *LoadError) Unwrap() error {
	return e.Cause
}

// NewValidationError creates a new ValidationError
func NewValidationError(field, message string) *ValidationError {
	return &ValidationError{
		Field:   field,
		Message: message,
	}
}

// NewLoadError creates a new LoadError
func NewLoadError(filename string, cause error) *LoadError {
	return &LoadError{
		Filename: filename,
		Cause:    cause,
	}
}
