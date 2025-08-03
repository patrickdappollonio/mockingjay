package middleware

import (
	"context"
	"log/slog"
	"net/http"
	"time"
)

// TimeoutConfig represents timeout middleware configuration
type TimeoutConfig struct {
	Duration time.Duration `yaml:"duration,omitempty"` // Request timeout duration
}

// TimeoutMiddleware implements request-level timeout handling
type TimeoutMiddleware struct {
	config TimeoutConfig
	logger *slog.Logger
}

// NewTimeoutMiddleware creates a new timeout middleware instance
func NewTimeoutMiddleware(config TimeoutConfig, logger *slog.Logger) *TimeoutMiddleware {
	// Set default timeout if not specified
	if config.Duration == 0 {
		config.Duration = 30 * time.Second
	}

	return &TimeoutMiddleware{
		config: config,
		logger: logger,
	}
}

// Name returns the middleware name
func (m *TimeoutMiddleware) Name() string {
	return "timeout"
}

// Handler returns the standard Go middleware handler
func (m *TimeoutMiddleware) Handler() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Create a context with timeout
			ctx, cancel := context.WithTimeout(r.Context(), m.config.Duration)
			defer cancel()

			// Replace the request context
			r = r.WithContext(ctx)

			// Start timing the request
			start := time.Now()

			// Channel to signal completion and capture any panic
			done := make(chan error, 1)

			// Execute handler in goroutine so we can monitor timeout
			go func() {
				defer func() {
					if err := recover(); err != nil {
						// Convert panic to error
						done <- http.ErrAbortHandler
						return
					}
					done <- nil
				}()
				next.ServeHTTP(w, r)
			}()

			// Wait for completion or timeout
			select {
			case err := <-done:
				// Handler completed
				duration := time.Since(start)
				if err != nil {
					m.logger.Error("request handler panicked",
						"path", r.URL.Path,
						"method", r.Method,
						"duration", duration,
						"remote_addr", r.RemoteAddr,
						"error", err,
					)
					// Don't write response if already written
					if wrapper, ok := w.(*ResponseWriter); ok && !wrapper.Written() {
						http.Error(w, "internal server error", http.StatusInternalServerError)
					}
				}
				return
			case <-ctx.Done():
				// Timeout occurred
				m.logger.Warn("request timeout",
					"path", r.URL.Path,
					"method", r.Method,
					"timeout", m.config.Duration,
					"remote_addr", r.RemoteAddr,
				)

				// Send timeout response if we haven't written anything yet
				if wrapper, ok := w.(*ResponseWriter); ok && !wrapper.Written() {
					http.Error(w, "request timeout", http.StatusRequestTimeout)
				}

				// Wait for the handler to complete to avoid goroutine leaks
				go func() {
					<-done // Wait for handler completion
				}()
				return
			}
		})
	}
}
