package middleware

import (
	"net/http"
	"regexp"
	"strings"
)

// BasicAuthConfig represents basic authentication middleware configuration
type BasicAuthConfig struct {
	Username string         `yaml:"username"` // Username for authentication
	Password string         `yaml:"password"` // Password for authentication
	Realm    string         `yaml:"realm"`    // Authentication realm (optional)
	Paths    BasicAuthPaths `yaml:"paths"`    // Path matching rules
}

// BasicAuthPaths defines which paths the basic auth applies to
type BasicAuthPaths struct {
	Include []string `yaml:"include"` // Paths to include (apply auth to)
	Exclude []string `yaml:"exclude"` // Paths to exclude (skip auth for)
}

// PathMatcher represents a compiled path matching rule
type PathMatcher struct {
	IsRegex bool           // Whether this is a regex or literal match
	Regex   *regexp.Regexp // Compiled regex pattern (nil for literal matches)
	Literal string         // Literal string to match (empty for regex matches)
}

// BasicAuthMiddleware implements HTTP Basic Authentication
type BasicAuthMiddleware struct {
	config          BasicAuthConfig
	includeMatcher  []*PathMatcher // Compiled include path matchers
	excludeMatchers []*PathMatcher // Compiled exclude path matchers
}

// NewBasicAuthMiddleware creates a new basic auth middleware with configuration
func NewBasicAuthMiddleware(config BasicAuthConfig) (*BasicAuthMiddleware, error) {
	// Set default realm if not specified
	if config.Realm == "" {
		config.Realm = "Restricted Area"
	}

	middleware := &BasicAuthMiddleware{
		config: config,
	}

	// Compile include path matchers
	var err error
	middleware.includeMatcher, err = compilePathMatchers(config.Paths.Include)
	if err != nil {
		return nil, err
	}

	// Compile exclude path matchers
	middleware.excludeMatchers, err = compilePathMatchers(config.Paths.Exclude)
	if err != nil {
		return nil, err
	}

	return middleware, nil
}

// compilePathMatchers compiles a list of path patterns into PathMatchers
func compilePathMatchers(paths []string) ([]*PathMatcher, error) {
	matchers := make([]*PathMatcher, len(paths))

	for i, path := range paths {
		matcher := &PathMatcher{}

		// Determine if this is a regex pattern (wrapped in /^.../$)
		if isRegexPath(path) {
			// Extract regex pattern and compile it
			pattern := extractRegexPath(path)
			regex, err := regexp.Compile(pattern)
			if err != nil {
				return nil, err
			}
			matcher.IsRegex = true
			matcher.Regex = regex
		} else {
			// Literal path match
			matcher.IsRegex = false
			matcher.Literal = path
		}

		matchers[i] = matcher
	}

	return matchers, nil
}

// isRegexPath returns true if the path should be treated as a regex pattern
func isRegexPath(path string) bool {
	return strings.HasPrefix(path, "/") && strings.HasSuffix(path, "/") && len(path) > 2
}

// extractRegexPath extracts the regex pattern from a path (removes surrounding slashes)
func extractRegexPath(path string) string {
	if isRegexPath(path) {
		return strings.TrimPrefix(strings.TrimSuffix(path, "/"), "/")
	}
	return path
}

// Name returns the middleware name
func (b *BasicAuthMiddleware) Name() string {
	return "basicauth"
}

// Handler returns the standard Go middleware handler
func (b *BasicAuthMiddleware) Handler() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check if this path should be authenticated
			if !b.shouldAuthenticate(r.URL.Path) {
				next.ServeHTTP(w, r)
				return
			}

			// Extract credentials from Authorization header
			username, password, ok := r.BasicAuth()
			if !ok {
				b.unauthorized(w)
				return
			}

			// Validate credentials
			if !b.validateCredentials(username, password) {
				b.unauthorized(w)
				return
			}

			// Authentication successful, continue to next handler
			next.ServeHTTP(w, r)
		})
	}
}

// shouldAuthenticate determines if a path should require authentication
func (b *BasicAuthMiddleware) shouldAuthenticate(path string) bool {
	// If no include patterns specified, apply to all paths
	if len(b.includeMatcher) == 0 {
		// Check excludes only
		return !b.matchesAny(path, b.excludeMatchers)
	}

	// Check if path matches any include pattern
	if !b.matchesAny(path, b.includeMatcher) {
		return false
	}

	// Check if path matches any exclude pattern (excludes take precedence)
	return !b.matchesAny(path, b.excludeMatchers)
}

// matchesAny checks if a path matches any of the provided matchers
func (b *BasicAuthMiddleware) matchesAny(path string, matchers []*PathMatcher) bool {
	for _, matcher := range matchers {
		if b.matchesPath(path, matcher) {
			return true
		}
	}
	return false
}

// matchesPath checks if a path matches a specific PathMatcher
func (b *BasicAuthMiddleware) matchesPath(path string, matcher *PathMatcher) bool {
	if matcher.IsRegex {
		return matcher.Regex != nil && matcher.Regex.MatchString(path)
	}
	return path == matcher.Literal
}

// validateCredentials checks if the provided credentials are valid
func (b *BasicAuthMiddleware) validateCredentials(username, password string) bool {
	return username == b.config.Username && password == b.config.Password
}

// unauthorized sends a 401 Unauthorized response with WWW-Authenticate header
func (b *BasicAuthMiddleware) unauthorized(w http.ResponseWriter) {
	w.Header().Set("WWW-Authenticate", `Basic realm="`+b.config.Realm+`"`)
	w.WriteHeader(http.StatusUnauthorized)
	w.Write([]byte("401 Unauthorized"))
}
