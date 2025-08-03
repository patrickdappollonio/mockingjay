package config

import (
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/goccy/go-yaml"

	"github.com/patrickdappollonio/mockingjay/internal/middleware"
	templatepkg "github.com/patrickdappollonio/mockingjay/internal/template"
)

// Config represents the top-level configuration loaded from YAML
type Config struct {
	Routes     []RouteConfig     `yaml:"routes"`
	Middleware middleware.Config `yaml:"middleware,omitempty"`
	Server     ServerConfig      `yaml:"server,omitempty"`
	Template   TemplateConfig    `yaml:"template,omitempty"`
}

// ServerConfig represents server-level configuration options
type ServerConfig struct {
	Timeouts TimeoutConfig `yaml:"timeouts,omitempty"`
}

// TimeoutConfig represents timeout configuration options
type TimeoutConfig struct {
	Read       time.Duration `yaml:"read,omitempty"`        // ReadTimeout
	Write      time.Duration `yaml:"write,omitempty"`       // WriteTimeout
	Idle       time.Duration `yaml:"idle,omitempty"`        // IdleTimeout
	ReadHeader time.Duration `yaml:"read_header,omitempty"` // ReadHeaderTimeout
	Request    time.Duration `yaml:"request,omitempty"`     // Per-request timeout
	Shutdown   time.Duration `yaml:"shutdown,omitempty"`    // Graceful shutdown timeout
}

// TemplateConfig represents template engine configuration options
type TemplateConfig struct {
	Delimiters DelimiterConfig `yaml:"delimiters,omitempty"`
}

// DelimiterConfig represents custom template delimiter configuration
type DelimiterConfig struct {
	Left  string `yaml:"left,omitempty"`  // Left delimiter (default: "{{")
	Right string `yaml:"right,omitempty"` // Right delimiter (default: "}}")
}

// GetWithDefaults returns timeout values with sensible defaults
func (tc *TimeoutConfig) GetWithDefaults() TimeoutConfig {
	config := *tc

	// Apply default values if not set
	if config.Read == 0 {
		config.Read = 15 * time.Second
	}
	if config.Write == 0 {
		config.Write = 15 * time.Second
	}
	if config.Idle == 0 {
		config.Idle = 60 * time.Second
	}
	if config.ReadHeader == 0 {
		config.ReadHeader = 5 * time.Second
	}
	if config.Request == 0 {
		config.Request = 30 * time.Second
	}
	if config.Shutdown == 0 {
		config.Shutdown = 30 * time.Second
	}

	return config
}

// GetWithDefaults returns delimiter values with sensible defaults
func (dc *DelimiterConfig) GetWithDefaults() DelimiterConfig {
	config := *dc

	// Apply default values if not set
	if config.Left == "" {
		config.Left = "{{"
	}
	if config.Right == "" {
		config.Right = "}}"
	}

	return config
}

// RouteConfig represents a single route configuration from YAML
type RouteConfig struct {
	Path            string            `yaml:"path"`
	Method          string            `yaml:"method"`
	Template        string            `yaml:"template,omitempty"`
	TemplateFile    string            `yaml:"template_file,omitempty"`
	MatchHeaders    map[string]string `yaml:"match_headers,omitempty"`
	ResponseHeaders map[string]string `yaml:"response_headers,omitempty"`
}

// LoadConfig loads and validates a configuration from a YAML file
func LoadConfig(filename string) (*Config, error) {
	// Check if file exists and is readable
	if err := checkFileAccessibility(filename); err != nil {
		return nil, NewLoadError(filename, err)
	}

	// Read the file contents
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, NewLoadError(filename, fmt.Errorf("failed to read file: %w", err))
	}

	// Unmarshal YAML into Config struct
	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, NewLoadError(filename, fmt.Errorf("failed to parse YAML: %w", err))
	}

	// Validate the configuration
	if err := config.Validate(); err != nil {
		return nil, NewLoadError(filename, fmt.Errorf("configuration validation failed: %w", err))
	}

	return &config, nil
}

// checkFileAccessibility verifies that the file exists and is readable
func checkFileAccessibility(filename string) error {
	if strings.TrimSpace(filename) == "" {
		return fmt.Errorf("filename cannot be empty")
	}

	fileInfo, err := os.Stat(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("config file %q does not exist", filename)
		}
		return fmt.Errorf("cannot access config file %q: %w", filename, err)
	}

	if fileInfo.IsDir() {
		return fmt.Errorf("config file %q is a directory, not a file", filename)
	}

	// Try to open the file to check readability
	file, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("config file %q is not readable: %w", filename, err)
	}
	file.Close()

	return nil
}

// Validate validates the Config and all its RouteConfigs
func (c *Config) Validate() error {
	if len(c.Routes) == 0 {
		return &ValidationError{
			Field:   "routes",
			Message: "at least one route must be defined",
		}
	}

	for i, route := range c.Routes {
		if err := route.Validate(); err != nil {
			return fmt.Errorf("route[%d]: %w", i, err)
		}
	}

	// Validate template configuration
	if err := c.Template.Validate(); err != nil {
		return fmt.Errorf("template configuration: %w", err)
	}

	// Validate templates by attempting to compile them
	if err := c.ValidateTemplates(); err != nil {
		return fmt.Errorf("template validation failed: %w", err)
	}

	return nil
}

// Validate validates a single RouteConfig
func (r *RouteConfig) Validate() error {
	// Validate path is not empty
	if strings.TrimSpace(r.Path) == "" {
		return &ValidationError{
			Field:   "path",
			Message: "path cannot be empty",
		}
	}

	// Validate HTTP method
	if err := r.validateHTTPMethod(); err != nil {
		return err
	}

	// Validate exactly one of template or template_file is provided
	if err := r.validateTemplateSource(); err != nil {
		return err
	}

	// Validate template file exists if template_file is specified
	if r.TemplateFile != "" {
		if err := r.validateTemplateFileExists(); err != nil {
			return err
		}
	}

	// Validate regex pattern if path appears to be a regex
	if err := r.validateRegexPattern(); err != nil {
		return err
	}

	// Validate header matching patterns
	if err := r.validateMatchHeaders(); err != nil {
		return err
	}

	// Validate response headers
	if err := r.validateResponseHeaders(); err != nil {
		return err
	}

	return nil
}

// validateHTTPMethod checks if the HTTP method is valid
func (r *RouteConfig) validateHTTPMethod() error {
	if strings.TrimSpace(r.Method) == "" {
		return &ValidationError{
			Field:   "method",
			Message: "HTTP method cannot be empty",
		}
	}

	method := strings.ToUpper(strings.TrimSpace(r.Method))
	validMethods := []string{
		http.MethodGet,
		http.MethodPost,
		http.MethodPut,
		http.MethodPatch,
		http.MethodDelete,
		http.MethodHead,
		http.MethodOptions,
		http.MethodConnect,
		http.MethodTrace,
	}

	for _, validMethod := range validMethods {
		if method == validMethod {
			return nil
		}
	}

	return &ValidationError{
		Field:   "method",
		Message: fmt.Sprintf("invalid HTTP method %q, must be one of: %s", method, strings.Join(validMethods, ", ")),
	}
}

// validateTemplateSource ensures exactly one of template or template_file is provided
func (r *RouteConfig) validateTemplateSource() error {
	hasTemplate := strings.TrimSpace(r.Template) != ""
	hasTemplateFile := strings.TrimSpace(r.TemplateFile) != ""

	if !hasTemplate && !hasTemplateFile {
		return &ValidationError{
			Field:   "template",
			Message: "either 'template' or 'template_file' must be specified",
		}
	}

	if hasTemplate && hasTemplateFile {
		return &ValidationError{
			Field:   "template",
			Message: "only one of 'template' or 'template_file' can be specified, not both",
		}
	}

	return nil
}

// validateTemplateFileExists checks if the template file exists and is readable
func (r *RouteConfig) validateTemplateFileExists() error {
	if _, err := os.Stat(r.TemplateFile); err != nil {
		if os.IsNotExist(err) {
			return &ValidationError{
				Field:   "template_file",
				Message: fmt.Sprintf("template file %q does not exist", r.TemplateFile),
			}
		}
		return &ValidationError{
			Field:   "template_file",
			Message: fmt.Sprintf("cannot access template file %q: %v", r.TemplateFile, err),
		}
	}
	return nil
}

// validateRegexPattern validates regex syntax if the path appears to be a regex
func (r *RouteConfig) validateRegexPattern() error {
	if r.IsRegexPattern() {
		// Extract the regex pattern (remove surrounding slashes)
		pattern := strings.TrimPrefix(strings.TrimSuffix(r.Path, "/"), "/")
		if _, err := regexp.Compile(pattern); err != nil {
			return &ValidationError{
				Field:   "path",
				Message: fmt.Sprintf("invalid regex pattern %q: %v", pattern, err),
			}
		}
	}
	return nil
}

// IsRegexPattern returns true if the path should be treated as a regex pattern
func (r *RouteConfig) IsRegexPattern() bool {
	return strings.HasPrefix(r.Path, "/") && strings.HasSuffix(r.Path, "/") && len(r.Path) > 2
}

// GetRegexPattern extracts the regex pattern from the path (removes surrounding slashes)
func (r *RouteConfig) GetRegexPattern() string {
	if r.IsRegexPattern() {
		return strings.TrimPrefix(strings.TrimSuffix(r.Path, "/"), "/")
	}
	return r.Path
}

// validateMatchHeaders validates header matching patterns
func (r *RouteConfig) validateMatchHeaders() error {
	for headerName, headerValue := range r.MatchHeaders {
		// Validate header name is not empty and is a valid HTTP header name
		if err := r.validateHeaderName(headerName); err != nil {
			return err
		}

		// Validate header value pattern (regex or literal)
		if err := r.validateHeaderValuePattern(headerName, headerValue); err != nil {
			return err
		}
	}
	return nil
}

// validateHeaderName checks if a header name is valid
func (r *RouteConfig) validateHeaderName(headerName string) error {
	trimmed := strings.TrimSpace(headerName)
	if trimmed == "" {
		return &ValidationError{
			Field:   "match_headers",
			Message: "header name cannot be empty",
		}
	}

	// HTTP header names should not contain invalid characters
	// RFC 7230 defines valid characters for header names
	for _, char := range trimmed {
		if !isValidHeaderNameChar(char) {
			return &ValidationError{
				Field:   "match_headers",
				Message: fmt.Sprintf("invalid character %q in header name %q", char, headerName),
			}
		}
	}

	return nil
}

// validateHeaderValuePattern validates header value patterns (regex or literal)
func (r *RouteConfig) validateHeaderValuePattern(headerName, headerValue string) error {
	if isRegexPattern(headerValue) {
		// Extract regex pattern and validate it
		pattern := extractRegexPattern(headerValue)
		if _, err := regexp.Compile(pattern); err != nil {
			return &ValidationError{
				Field:   "match_headers",
				Message: fmt.Sprintf("invalid regex pattern %q for header %q: %v", pattern, headerName, err),
			}
		}
	}
	// Literal strings are always valid, no need to validate
	return nil
}

// isValidHeaderNameChar checks if a character is valid in an HTTP header name
func isValidHeaderNameChar(char rune) bool {
	// RFC 7230: header names consist of tokens
	// token = 1*tchar
	// tchar = "!" / "#" / "$" / "%" / "&" / "'" / "*" / "+" / "-" / "." /
	//         "^" / "_" / "`" / "|" / "~" / DIGIT / ALPHA
	return (char >= 'A' && char <= 'Z') ||
		(char >= 'a' && char <= 'z') ||
		(char >= '0' && char <= '9') ||
		char == '!' || char == '#' || char == '$' || char == '%' || char == '&' ||
		char == '\'' || char == '*' || char == '+' || char == '-' || char == '.' ||
		char == '^' || char == '_' || char == '`' || char == '|' || char == '~'
}

// isRegexPattern returns true if the value should be treated as a regex pattern
func isRegexPattern(value string) bool {
	return strings.HasPrefix(value, "/") && strings.HasSuffix(value, "/") && len(value) > 2
}

// extractRegexPattern extracts the regex pattern from a value (removes surrounding slashes)
func extractRegexPattern(value string) string {
	if isRegexPattern(value) {
		return strings.TrimPrefix(strings.TrimSuffix(value, "/"), "/")
	}
	return value
}

// GetNormalizedMethod returns the HTTP method in uppercase
func (r *RouteConfig) GetNormalizedMethod() string {
	return strings.ToUpper(strings.TrimSpace(r.Method))
}

// validateResponseHeaders validates response header templates
func (r *RouteConfig) validateResponseHeaders() error {
	for headerName, headerValue := range r.ResponseHeaders {
		// Validate header name is not empty and is a valid HTTP header name
		if err := r.validateHeaderName(headerName); err != nil {
			return err
		}

		// Validate template syntax in header value
		if err := r.validateResponseHeaderTemplate(headerName, headerValue); err != nil {
			return err
		}
	}
	return nil
}

// validateResponseHeaderTemplate validates template syntax in a response header value
func (r *RouteConfig) validateResponseHeaderTemplate(headerName, headerValue string) error {
	// Basic template syntax validation - check for common template errors
	// We do a lenient validation here to catch obvious syntax errors without
	// requiring the full function map since we don't have access to template engine here

	// Check for unclosed template actions
	if strings.Contains(headerValue, "{{") && !strings.Contains(headerValue, "}}") {
		return &ValidationError{
			Field:   "response_headers",
			Message: fmt.Sprintf("invalid template syntax in response header %q: unclosed template action", headerName),
		}
	}

	// Check for unmatched closing braces
	if strings.Contains(headerValue, "}}") && !strings.Contains(headerValue, "{{") {
		return &ValidationError{
			Field:   "response_headers",
			Message: fmt.Sprintf("invalid template syntax in response header %q: unmatched closing braces", headerName),
		}
	}

	// For more detailed validation, we'll rely on the compilation phase
	// since we don't have access to the full function map here
	return nil
}

// Validate validates template configuration
func (tc *TemplateConfig) Validate() error {
	return tc.Delimiters.Validate()
}

// Validate validates delimiter configuration
func (dc *DelimiterConfig) Validate() error {
	// If both are empty, that's fine - we'll use defaults
	if dc.Left == "" && dc.Right == "" {
		return nil
	}

	// Validate left delimiter
	if err := dc.validateDelimiter(dc.Left, "left"); err != nil {
		return err
	}

	// Validate right delimiter
	if err := dc.validateDelimiter(dc.Right, "right"); err != nil {
		return err
	}

	// Ensure delimiters are different
	if dc.Left == dc.Right {
		return &ValidationError{
			Field:   "delimiters",
			Message: "left and right delimiters cannot be the same",
		}
	}

	return nil
}

// validateDelimiter validates a single delimiter
func (dc *DelimiterConfig) validateDelimiter(delimiter, side string) error {
	if delimiter == "" {
		return &ValidationError{
			Field:   fmt.Sprintf("delimiters.%s", side),
			Message: "delimiter cannot be empty if specified",
		}
	}

	// Check for invalid characters that could cause parsing issues
	if strings.ContainsAny(delimiter, "\n\r\t") {
		return &ValidationError{
			Field:   fmt.Sprintf("delimiters.%s", side),
			Message: "delimiter cannot contain newline, carriage return, or tab characters",
		}
	}

	// Delimiter should be reasonably short
	if len(delimiter) > 10 {
		return &ValidationError{
			Field:   fmt.Sprintf("delimiters.%s", side),
			Message: "delimiter cannot be longer than 10 characters",
		}
	}

	return nil
}

// ValidateTemplates validates all templates by attempting to compile them
func (c *Config) ValidateTemplates() error {
	// Create a template engine for validation with configured delimiters
	delimiters := c.Template.Delimiters.GetWithDefaults()
	engine := templatepkg.NewEngineWithDelimiters(delimiters.Left, delimiters.Right)

	for i, route := range c.Routes {
		if err := c.validateRouteTemplates(engine, route, i); err != nil {
			return err
		}
	}

	return nil
}

// validateRouteTemplates validates templates for a single route
func (c *Config) validateRouteTemplates(engine *templatepkg.Engine, route RouteConfig, routeIndex int) error {
	// Validate main response template
	if err := c.validateMainTemplate(engine, route, routeIndex); err != nil {
		return err
	}

	// Validate response header templates
	if err := c.validateResponseHeaderTemplates(engine, route, routeIndex); err != nil {
		return err
	}

	return nil
}

// validateMainTemplate validates the main response template for a route
func (c *Config) validateMainTemplate(engine *templatepkg.Engine, route RouteConfig, routeIndex int) error {
	if route.Template != "" {
		// Validate inline template
		templateName := fmt.Sprintf("validation_route_%d_%s_%s", routeIndex, route.GetNormalizedMethod(), sanitizeTemplateNameForValidation(route.Path))
		_, err := engine.CompileInlineTemplate(templateName, route.Template)
		if err != nil {
			return fmt.Errorf("route[%d] template compilation failed: %w", routeIndex, err)
		}
	} else if route.TemplateFile != "" {
		// Validate file template
		_, err := engine.CompileFileTemplate(route.TemplateFile)
		if err != nil {
			return fmt.Errorf("route[%d] template file %q compilation failed: %w", routeIndex, route.TemplateFile, err)
		}
	}

	return nil
}

// validateResponseHeaderTemplates validates response header templates for a route
func (c *Config) validateResponseHeaderTemplates(engine *templatepkg.Engine, route RouteConfig, routeIndex int) error {
	for headerName, headerValue := range route.ResponseHeaders {
		templateName := fmt.Sprintf("validation_header_%d_%s_%s_%s", routeIndex, route.GetNormalizedMethod(), sanitizeTemplateNameForValidation(route.Path), sanitizeTemplateNameForValidation(headerName))
		_, err := engine.CompileInlineTemplate(templateName, headerValue)
		if err != nil {
			return fmt.Errorf("route[%d] response header %q template compilation failed: %w", routeIndex, headerName, err)
		}
	}

	return nil
}

// nonAlphanumericRegex matches any character that is not alphanumeric or underscore
var nonAlphanumericRegex = regexp.MustCompile(`[^a-zA-Z0-9_]+`)

// sanitizeTemplateNameForValidation converts a string into a safe template name for validation
// using a precompiled regex to replace non-alphanumeric characters with underscores
func sanitizeTemplateNameForValidation(input string) string {
	// Replace any sequence of non-alphanumeric characters with a single underscore
	sanitized := nonAlphanumericRegex.ReplaceAllString(input, "_")

	// Trim leading and trailing underscores
	sanitized = strings.Trim(sanitized, "_")

	// Ensure it's not empty
	if sanitized == "" {
		sanitized = "unnamed"
	}

	return sanitized
}
