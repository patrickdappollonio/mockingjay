package template

import "fmt"

// ContextError represents an error that occurred while building template context
type ContextError struct {
	Component string // The component that failed (e.g., "body", "headers", "query")
	Message   string // Human-readable error message
	Cause     error  // The underlying error
}

func (e *ContextError) Error() string {
	if e.Component != "" {
		return fmt.Sprintf("template context error in %s: %s", e.Component, e.Message)
	}
	return fmt.Sprintf("template context error: %s", e.Message)
}

// Unwrap allows errors.Is and errors.As to work with ContextError
func (e *ContextError) Unwrap() error {
	return e.Cause
}

// ExecutionError represents an error that occurred during template execution
type ExecutionError struct {
	TemplateName string // Name of the template that failed
	Message      string // Human-readable error message
	Cause        error  // The underlying error
}

func (e *ExecutionError) Error() string {
	if e.TemplateName != "" {
		return fmt.Sprintf("template execution error in %q: %s", e.TemplateName, e.Message)
	}
	return fmt.Sprintf("template execution error: %s", e.Message)
}

// Unwrap allows errors.Is and errors.As to work with ExecutionError
func (e *ExecutionError) Unwrap() error {
	return e.Cause
}

// CompilationError represents an error that occurred during template compilation
type CompilationError struct {
	Source  string // Template source (filename or "inline")
	Message string // Human-readable error message
	Cause   error  // The underlying error
}

func (e *CompilationError) Error() string {
	return fmt.Sprintf("template compilation error in %s: %s", e.Source, e.Message)
}

// Unwrap allows errors.Is and errors.As to work with CompilationError
func (e *CompilationError) Unwrap() error {
	return e.Cause
}

// NewContextError creates a new ContextError
func NewContextError(component, message string, cause error) *ContextError {
	return &ContextError{
		Component: component,
		Message:   message,
		Cause:     cause,
	}
}

// NewExecutionError creates a new ExecutionError
func NewExecutionError(templateName, message string, cause error) *ExecutionError {
	return &ExecutionError{
		TemplateName: templateName,
		Message:      message,
		Cause:        cause,
	}
}

// NewCompilationError creates a new CompilationError
func NewCompilationError(source, message string, cause error) *CompilationError {
	return &CompilationError{
		Source:  source,
		Message: message,
		Cause:   cause,
	}
}
