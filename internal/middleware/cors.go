package middleware

import (
	"net/http"
	"strconv"
	"strings"
)

// CORSConfig represents CORS middleware configuration
type CORSConfig struct {
	AllowOrigins     []string `yaml:"allow_origins"`
	AllowMethods     []string `yaml:"allow_methods"`
	AllowHeaders     []string `yaml:"allow_headers"`
	ExposeHeaders    []string `yaml:"expose_headers"`
	AllowCredentials bool     `yaml:"allow_credentials"`
	MaxAge           int      `yaml:"max_age"`
}

// CORSMiddleware implements CORS (Cross-Origin Resource Sharing) support
type CORSMiddleware struct {
	config CORSConfig
}

// NewCORSMiddleware creates a new CORS middleware with configuration
func NewCORSMiddleware(config CORSConfig) *CORSMiddleware {
	// Set defaults if not specified
	if len(config.AllowOrigins) == 0 {
		config.AllowOrigins = []string{"*"}
	}
	if len(config.AllowMethods) == 0 {
		config.AllowMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}
	}
	if len(config.AllowHeaders) == 0 {
		config.AllowHeaders = []string{"Content-Type", "Authorization"}
	}
	if config.MaxAge == 0 {
		config.MaxAge = 3600 // 1 hour
	}

	return &CORSMiddleware{config: config}
}

// Name returns the middleware name
func (c *CORSMiddleware) Name() string {
	return "cors"
}

// Handler returns the standard Go middleware handler
func (c *CORSMiddleware) Handler() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")

			// Check if origin is allowed
			if len(c.config.AllowOrigins) == 1 && c.config.AllowOrigins[0] == "*" {
				w.Header().Set("Access-Control-Allow-Origin", "*")
			} else if c.isOriginAllowed(origin) {
				w.Header().Set("Access-Control-Allow-Origin", origin)
			}

			// Set other CORS headers
			if len(c.config.AllowMethods) > 0 {
				w.Header().Set("Access-Control-Allow-Methods", strings.Join(c.config.AllowMethods, ", "))
			}

			if len(c.config.AllowHeaders) > 0 {
				w.Header().Set("Access-Control-Allow-Headers", strings.Join(c.config.AllowHeaders, ", "))
			}

			if len(c.config.ExposeHeaders) > 0 {
				w.Header().Set("Access-Control-Expose-Headers", strings.Join(c.config.ExposeHeaders, ", "))
			}

			if c.config.AllowCredentials {
				w.Header().Set("Access-Control-Allow-Credentials", "true")
			}

			if c.config.MaxAge > 0 {
				w.Header().Set("Access-Control-Max-Age", strconv.Itoa(c.config.MaxAge))
			}

			// Handle preflight OPTIONS requests
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			// Continue to next handler
			next.ServeHTTP(w, r)
		})
	}
}

// isOriginAllowed checks if the origin is in the allowed origins list
func (c *CORSMiddleware) isOriginAllowed(origin string) bool {
	for _, allowedOrigin := range c.config.AllowOrigins {
		if allowedOrigin == "*" || allowedOrigin == origin {
			return true
		}
	}
	return false
}
