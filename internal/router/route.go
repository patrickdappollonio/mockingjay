package router

import (
	"html/template"
	"net/http"
	"regexp"
	"strings"
)

// HeaderMatcher represents a compiled header matching rule
type HeaderMatcher struct {
	IsRegex bool           // Whether this is a regex or literal match
	Regex   *regexp.Regexp // Compiled regex pattern (nil for literal matches)
	Literal string         // Literal string to match (empty for regex matches)
}

// Route represents a compiled route ready for matching and execution
type Route struct {
	// Original configuration
	Pattern string // The original path pattern from config
	Verb    string // HTTP verb (uppercase)

	// Compiled regex information
	IsRegexp bool           // Whether this route uses regex matching
	Regex    *regexp.Regexp // Compiled regex for pattern matching (nil if not regex)

	// Header matching
	MatchHeaders map[string]*HeaderMatcher // Compiled header matchers

	// Template
	Tmpl *template.Template // Compiled template for rendering responses

	// Response headers
	ResponseHeaders map[string]*template.Template // Compiled response header templates

	// Template source info (for debugging/logging)
	TemplateSource string // "inline" or filename
}

// RouteMatch represents the result of matching a route against a request
type RouteMatch struct {
	Route  *Route            // The matched route
	Params map[string]string // Named capture groups from regex (empty for literal matches)
}

// MatchRequest checks if this route matches the given HTTP request
func (r *Route) MatchRequest(req *http.Request) (*RouteMatch, bool) {
	// Check HTTP verb first (fail fast)
	if !r.matchesVerb(req.Method) {
		return nil, false
	}

	// Check path pattern
	var match *RouteMatch
	var pathMatches bool

	if r.IsRegexp {
		match, pathMatches = r.matchRegexPattern(req.URL.Path)
	} else {
		match, pathMatches = r.matchLiteralPattern(req.URL.Path)
	}

	if !pathMatches {
		return nil, false
	}

	// Check header matching
	if !r.matchesHeaders(req) {
		return nil, false
	}

	return match, true
}

// matchesVerb checks if the route's verb matches the request method
func (r *Route) matchesVerb(method string) bool {
	return strings.EqualFold(r.Verb, method)
}

// matchRegexPattern matches the request path against the regex pattern
func (r *Route) matchRegexPattern(path string) (*RouteMatch, bool) {
	if r.Regex == nil {
		return nil, false
	}

	matches := r.Regex.FindStringSubmatch(path)
	if matches == nil {
		return nil, false
	}

	// Extract named capture groups
	params := make(map[string]string)
	names := r.Regex.SubexpNames()

	for i, name := range names {
		if i > 0 && i < len(matches) && name != "" {
			params[name] = matches[i]
		}
	}

	return &RouteMatch{
		Route:  r,
		Params: params,
	}, true
}

// matchLiteralPattern matches the request path against the literal pattern
func (r *Route) matchLiteralPattern(path string) (*RouteMatch, bool) {
	if path == r.Pattern {
		return &RouteMatch{
			Route:  r,
			Params: make(map[string]string), // Empty params for literal matches
		}, true
	}

	return nil, false
}

// String returns a string representation of the route for debugging
func (r *Route) String() string {
	routeType := "literal"
	if r.IsRegexp {
		routeType = "regex"
	}

	return strings.Join([]string{
		r.Verb,
		r.Pattern,
		"(" + routeType + ")",
		"template=" + r.TemplateSource,
	}, " ")
}

// matchesHeaders checks if the request headers match the route's header requirements
func (r *Route) matchesHeaders(req *http.Request) bool {
	// If no header matching is configured, always match
	if len(r.MatchHeaders) == 0 {
		return true
	}

	// All configured headers must match
	for headerName, headerMatcher := range r.MatchHeaders {
		// Get the header value from the request (case-insensitive)
		headerValue := getHeaderIgnoreCase(req, headerName)

		// If the required header is missing, no match
		if headerValue == "" {
			return false
		}

		// Check if the header value matches the pattern
		if !r.matchHeaderValue(headerValue, headerMatcher) {
			return false
		}
	}

	return true
}

// matchHeaderValue checks if a header value matches the expected pattern
func (r *Route) matchHeaderValue(value string, matcher *HeaderMatcher) bool {
	if matcher.IsRegex {
		// Regex pattern matching
		return matcher.Regex.MatchString(value)
	}

	// Literal string matching (exact match)
	return value == matcher.Literal
}

// getHeaderIgnoreCase gets a header value by name, ignoring case
func getHeaderIgnoreCase(req *http.Request, name string) string {
	// Convert to lowercase for comparison
	lowerName := strings.ToLower(name)

	// Check all headers for a case-insensitive match
	for headerName, headerValues := range req.Header {
		if strings.ToLower(headerName) == lowerName && len(headerValues) > 0 {
			return headerValues[0] // Return the first value
		}
	}

	return ""
}
