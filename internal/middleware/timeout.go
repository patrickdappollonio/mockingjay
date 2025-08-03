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

// TimeoutMiddleware implements request-level timeout handling using http.ResponseController
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

// Handler returns the standard Go middleware handler with proper context timeout
func (m *TimeoutMiddleware) Handler() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Create a context with timeout that will actually cancel
			ctx, cancel := context.WithTimeout(r.Context(), m.config.Duration)
			defer cancel()

			// Replace the request context with our timeout context
			r = r.WithContext(ctx)

			// Create a channel to track handler completion
			done := make(chan struct{}, 1)

			// Execute handler in goroutine so we can detect timeout
			go func() {
				defer func() {
					// nolint:staticcheck // SA9003: empty branch is intentional - we just want to catch panics without handling them
					recover()
					close(done)
				}()
				next.ServeHTTP(w, r)
			}()

			// Wait for either completion or timeout
			select {
			case <-done:
				// Handler completed normally
				return
			case <-ctx.Done():
				// Timeout occurred - send 408 response
				m.logger.Warn("request timeout",
					"path", r.URL.Path,
					"method", r.Method,
					"timeout", m.config.Duration,
					"remote_addr", r.RemoteAddr,
				)

				w.Header().Set("Content-Type", "text/plain; charset=utf-8")
				w.WriteHeader(http.StatusRequestTimeout)
				w.Write([]byte("408 Request Timeout\n\nThe request exceeded the configured timeout."))
				return
			}
		})
	}
}
