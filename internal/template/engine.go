package template

import (
	"fmt"
	"html/template"
	"io"
	"net/http"
	"strings"

	"github.com/Masterminds/sprig"
)

// Engine handles template compilation and execution with custom function maps
type Engine struct {
	funcMap template.FuncMap
}

// NewEngine creates a new template engine with all available functions
func NewEngine() *Engine {
	engine := &Engine{
		funcMap: createFuncMap(),
	}
	return engine
}

// createFuncMap builds the complete function map combining Sprig functions with custom ones
func createFuncMap() template.FuncMap {
	// Start with sprig functions (provides 100+ utility functions)
	funcMap := sprig.FuncMap()

	// Add our custom template functions
	customFuncs := template.FuncMap{
		"trimPrefix": customTrimPrefix,
		"header":     customHeader,
		"query":      customQuery,
		"jsonBody":   customJsonBody,
	}

	// Merge custom functions into the sprig function map
	for name, fn := range customFuncs {
		funcMap[name] = fn
	}

	return funcMap
}

// customTrimPrefix removes a prefix from a string (arguments reversed from strings.TrimPrefix for pipeline usage)
func customTrimPrefix(prefix, s string) string {
	return strings.TrimPrefix(s, prefix)
}

// customHeader extracts an HTTP header value from the request context
// Usage in templates: {{ header "X-User-ID" .Request }} or {{ .Request | header "X-User-ID" }}
func customHeader(key string, req *http.Request) string {
	if req == nil {
		return ""
	}
	return req.Header.Get(key)
}

// customQuery extracts a query parameter value from the request context
// Usage in templates: {{ query "debug" .Request }} or {{ .Request | query "debug" }}
func customQuery(key string, req *http.Request) string {
	if req == nil || req.URL == nil {
		return ""
	}
	return req.URL.Query().Get(key)
}

// customJsonBody returns the parsed JSON body from the template context
// This function works differently - it accesses the already-parsed body from context
// Usage in templates: {{ jsonBody }} (accesses .Body from context if it's JSON)
func customJsonBody() interface{} {
	// Note: This is a placeholder. In actual template execution,
	// the JSON body is already available in the context as .Body
	// This function exists for compatibility but templates should use .Body directly
	return nil
}

// CompileInlineTemplate compiles an inline template string with the engine's function map
func (e *Engine) CompileInlineTemplate(name, content string) (*template.Template, error) {
	if strings.TrimSpace(name) == "" {
		return nil, NewCompilationError("inline", "template name cannot be empty", nil)
	}

	if strings.TrimSpace(content) == "" {
		return nil, NewCompilationError("inline", "template content cannot be empty", nil)
	}

	tmpl, err := template.New(name).Funcs(e.funcMap).Parse(content)
	if err != nil {
		return nil, NewCompilationError("inline", fmt.Sprintf("failed to parse template: %v", err), err)
	}

	return tmpl, nil
}

// CompileFileTemplate compiles a template from a file with the engine's function map
func (e *Engine) CompileFileTemplate(filename string) (*template.Template, error) {
	if strings.TrimSpace(filename) == "" {
		return nil, NewCompilationError(filename, "filename cannot be empty", nil)
	}

	tmpl, err := template.New("").Funcs(e.funcMap).ParseFiles(filename)
	if err != nil {
		return nil, NewCompilationError(filename, fmt.Sprintf("failed to parse template file: %v", err), err)
	}

	return tmpl, nil
}

// ExecuteTemplate executes a template with the given context and writes the result to the writer
func (e *Engine) ExecuteTemplate(tmpl *template.Template, w io.Writer, ctx *TemplateContext) error {
	if tmpl == nil {
		return NewExecutionError("", "template is nil", nil)
	}

	if w == nil {
		return NewExecutionError(tmpl.Name(), "writer is nil", nil)
	}

	if ctx == nil {
		return NewExecutionError(tmpl.Name(), "context is nil", nil)
	}

	// Execute the template
	err := tmpl.Execute(w, ctx)
	if err != nil {
		return NewExecutionError(tmpl.Name(), fmt.Sprintf("template execution failed: %v", err), err)
	}

	return nil
}

// GetFuncMap returns a copy of the engine's function map
func (e *Engine) GetFuncMap() template.FuncMap {
	// Return a copy to prevent external modification
	funcMapCopy := make(template.FuncMap)
	for k, v := range e.funcMap {
		funcMapCopy[k] = v
	}
	return funcMapCopy
}

// BuildTemplateContext creates a complete template context from an HTTP request and route parameters
// This is a convenience function that wraps the existing NewTemplateContext function
func (e *Engine) BuildTemplateContext(req *http.Request, params map[string]string) (*TemplateContext, error) {
	if req == nil {
		return nil, NewContextError("request", "HTTP request cannot be nil", nil)
	}

	// Use the existing context builder which already handles all the complex parsing
	ctx, err := NewTemplateContext(req, params)
	if err != nil {
		return nil, NewContextError("context", "failed to build template context", err)
	}

	return ctx, nil
}

// CompileAndExecute is a convenience function that compiles a template and executes it in one step
func (e *Engine) CompileAndExecute(templateSource, templateContent string, w io.Writer, req *http.Request, params map[string]string) error {
	// Build the template context
	ctx, err := e.BuildTemplateContext(req, params)
	if err != nil {
		return fmt.Errorf("failed to build template context: %w", err)
	}

	// Determine if this is a file template or inline template
	var tmpl *template.Template
	if strings.TrimSpace(templateContent) != "" {
		// Inline template
		tmpl, err = e.CompileInlineTemplate(templateSource, templateContent)
		if err != nil {
			return fmt.Errorf("failed to compile inline template: %w", err)
		}
	} else {
		// File template
		tmpl, err = e.CompileFileTemplate(templateSource)
		if err != nil {
			return fmt.Errorf("failed to compile file template: %w", err)
		}
	}

	// Execute the template
	err = e.ExecuteTemplate(tmpl, w, ctx)
	if err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	return nil
}
