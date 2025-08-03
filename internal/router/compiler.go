package router

import (
	"fmt"
	"html/template"
	"regexp"
	"strings"

	"github.com/patrickdappollonio/mockingjay/internal/config"
	templatepkg "github.com/patrickdappollonio/mockingjay/internal/template"
)

// Compiler handles the compilation of route configurations into executable routes
type Compiler struct {
	engine *templatepkg.Engine
}

// NewCompiler creates a new route compiler with a template engine
func NewCompiler() *Compiler {
	return &Compiler{
		engine: templatepkg.NewEngine(),
	}
}

// CompileRoute compiles a RouteConfig into an executable Route
func (c *Compiler) CompileRoute(routeConfig config.RouteConfig) (*Route, error) {
	route := &Route{
		Pattern: routeConfig.Path,
		Verb:    routeConfig.GetNormalizedVerb(),
	}

	// Determine if this is a regex pattern
	route.IsRegexp = routeConfig.IsRegexPattern()

	// Compile regex if needed
	if route.IsRegexp {
		pattern := routeConfig.GetRegexPattern()
		regex, err := regexp.Compile(pattern)
		if err != nil {
			return nil, fmt.Errorf("failed to compile regex pattern %q: %w", pattern, err)
		}
		route.Regex = regex
	}

	// Compile header matching patterns
	if err := c.compileHeaderMatchers(route, routeConfig); err != nil {
		return nil, fmt.Errorf("failed to compile header matchers for route %q: %w", routeConfig.Path, err)
	}

	// Compile response header templates
	if err := c.compileResponseHeaders(route, routeConfig); err != nil {
		return nil, fmt.Errorf("failed to compile response headers for route %q: %w", routeConfig.Path, err)
	}

	// Compile the template
	tmpl, err := c.compileTemplate(routeConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to compile template for route %q: %w", routeConfig.Path, err)
	}
	route.Tmpl = tmpl

	// Set template source for debugging
	if routeConfig.Template != "" {
		route.TemplateSource = "inline"
	} else {
		route.TemplateSource = routeConfig.TemplateFile
	}

	return route, nil
}

// compileTemplate compiles the template for a route configuration
func (c *Compiler) compileTemplate(routeConfig config.RouteConfig) (*template.Template, error) {
	if routeConfig.Template != "" {
		// Inline template
		templateName := fmt.Sprintf("route_%s_%s", routeConfig.GetNormalizedVerb(), sanitizeTemplateName(routeConfig.Path))
		return c.engine.CompileInlineTemplate(templateName, routeConfig.Template)
	}

	if routeConfig.TemplateFile != "" {
		// File template
		return c.engine.CompileFileTemplate(routeConfig.TemplateFile)
	}

	return nil, fmt.Errorf("no template source specified")
}

// CompileRoutes compiles multiple route configurations
func (c *Compiler) CompileRoutes(routeConfigs []config.RouteConfig) ([]*Route, error) {
	routes := make([]*Route, 0, len(routeConfigs))

	for i, routeConfig := range routeConfigs {
		route, err := c.CompileRoute(routeConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to compile route %d (%s %s): %w", i, routeConfig.Verb, routeConfig.Path, err)
		}
		routes = append(routes, route)
	}

	return routes, nil
}

// GetEngine returns the template engine for advanced usage
func (c *Compiler) GetEngine() *templatepkg.Engine {
	return c.engine
}

// sanitizeTemplateName converts a path pattern into a safe template name
func sanitizeTemplateName(path string) string {
	// Replace problematic characters with underscores
	replacer := strings.NewReplacer(
		"/", "_",
		"?", "_",
		"*", "_",
		"+", "_",
		"-", "_",
		"(", "_",
		")", "_",
		"[", "_",
		"]", "_",
		"{", "_",
		"}", "_",
		"^", "_",
		"$", "_",
		"|", "_",
		"\\", "_",
		".", "_",
		" ", "_",
	)

	sanitized := replacer.Replace(path)

	// Remove multiple consecutive underscores
	for strings.Contains(sanitized, "__") {
		sanitized = strings.ReplaceAll(sanitized, "__", "_")
	}

	// Trim leading and trailing underscores
	sanitized = strings.Trim(sanitized, "_")

	// Ensure it's not empty
	if sanitized == "" {
		sanitized = "unnamed"
	}

	return sanitized
}

// compileHeaderMatchers compiles header matching patterns for a route
func (c *Compiler) compileHeaderMatchers(route *Route, routeConfig config.RouteConfig) error {
	if len(routeConfig.MatchHeaders) == 0 {
		route.MatchHeaders = nil
		return nil
	}

	route.MatchHeaders = make(map[string]*HeaderMatcher)

	for headerName, headerValue := range routeConfig.MatchHeaders {
		// Use canonical header name for consistent matching
		canonicalName := canonicalizeHeaderName(headerName)

		if isHeaderRegexPattern(headerValue) {
			// Compile regex pattern
			pattern := extractHeaderRegexPattern(headerValue)
			regex, err := regexp.Compile(pattern)
			if err != nil {
				return fmt.Errorf("invalid regex pattern %q for header %q: %w", pattern, headerName, err)
			}
			route.MatchHeaders[canonicalName] = &HeaderMatcher{
				IsRegex: true,
				Regex:   regex,
				Literal: "",
			}
		} else {
			// For literal strings, store the literal value
			route.MatchHeaders[canonicalName] = &HeaderMatcher{
				IsRegex: false,
				Regex:   nil,
				Literal: headerValue,
			}
		}
	}

	return nil
}

// canonicalizeHeaderName converts header names to canonical form for consistent comparison
func canonicalizeHeaderName(name string) string {
	// Go's http.CanonicalHeaderKey does the same thing but is in net/http
	// We'll implement a simple version here
	return strings.ToLower(strings.TrimSpace(name))
}

// isHeaderRegexPattern returns true if the header value should be treated as a regex pattern
func isHeaderRegexPattern(value string) bool {
	return strings.HasPrefix(value, "/") && strings.HasSuffix(value, "/") && len(value) > 2
}

// extractHeaderRegexPattern extracts the regex pattern from a header value (removes surrounding slashes)
func extractHeaderRegexPattern(value string) string {
	if isHeaderRegexPattern(value) {
		return strings.TrimPrefix(strings.TrimSuffix(value, "/"), "/")
	}
	return value
}

// compileResponseHeaders compiles response header templates for a route
func (c *Compiler) compileResponseHeaders(route *Route, routeConfig config.RouteConfig) error {
	if len(routeConfig.ResponseHeaders) == 0 {
		route.ResponseHeaders = nil
		return nil
	}

	route.ResponseHeaders = make(map[string]*template.Template)

	for headerName, headerValue := range routeConfig.ResponseHeaders {
		// Use canonical header name for consistent handling
		canonicalName := canonicalizeHeaderName(headerName)

		// Compile the header value as a template
		templateName := fmt.Sprintf("response_header_%s_%s_%s",
			routeConfig.GetNormalizedVerb(),
			sanitizeTemplateName(routeConfig.Path),
			sanitizeTemplateName(headerName))

		headerTemplate, err := c.engine.CompileInlineTemplate(templateName, headerValue)
		if err != nil {
			return fmt.Errorf("failed to compile response header template for %q: %w", headerName, err)
		}

		route.ResponseHeaders[canonicalName] = headerTemplate
	}

	return nil
}
