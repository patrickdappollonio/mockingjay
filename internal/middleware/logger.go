package middleware

import (
	"log/slog"
	"net/http"
	"time"
)

// LoggerConfig represents logger middleware configuration
type LoggerConfig struct {
	Format    string   `yaml:"format"`     // "json" or "text"
	Level     string   `yaml:"level"`      // "debug", "info", "warn", "error"
	Fields    []string `yaml:"fields"`     // Additional fields to log
	SkipPaths []string `yaml:"skip_paths"` // Paths to skip logging
}

// LoggerMiddleware implements request logging
type LoggerMiddleware struct {
	logger *slog.Logger
	config LoggerConfig
}

// NewLoggerMiddleware creates a new logger middleware
func NewLoggerMiddleware(logger *slog.Logger, config LoggerConfig) *LoggerMiddleware {
	// Set defaults
	if config.Format == "" {
		config.Format = "text"
	}
	if config.Level == "" {
		config.Level = "info"
	}

	return &LoggerMiddleware{
		logger: logger,
		config: config,
	}
}

// Name returns the middleware name
func (l *LoggerMiddleware) Name() string {
	return "logger"
}

// Handler returns the standard Go middleware handler
func (l *LoggerMiddleware) Handler() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check if we should skip logging this path
			if l.shouldSkipPath(r.URL.Path) {
				next.ServeHTTP(w, r)
				return
			}

			start := time.Now()

			// Continue to next handler
			next.ServeHTTP(w, r)

			// Log the request using the wrapped ResponseWriter
			duration := time.Since(start)

			// Try to get status and size from the ResponseWriter wrapper
			status := http.StatusOK
			size := 0

			if wrapper, ok := w.(*ResponseWriter); ok {
				status = wrapper.Status()
				size = wrapper.Size()
			}

			l.logger.Info("request processed",
				"method", r.Method,
				"path", r.URL.Path,
				"status", status,
				"size", size,
				"duration_ms", duration.Milliseconds(),
				"remote_addr", r.RemoteAddr,
				"user_agent", r.UserAgent(),
			)
		})
	}
}

// shouldSkipPath checks if a path should be skipped from logging
func (l *LoggerMiddleware) shouldSkipPath(path string) bool {
	for _, skipPath := range l.config.SkipPaths {
		if path == skipPath {
			return true
		}
	}
	return false
}
