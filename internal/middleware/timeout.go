package middleware

import (
	"bytes"
	"cmp"
	"context"
	"log/slog"
	"maps"
	"net/http"
	"sync"
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

			// The handler writes to tw; only this goroutine writes to w
			tw := &timeoutWriter{header: make(http.Header)}
			done := make(chan struct{}, 1)

			// Execute handler in goroutine so we can detect timeout
			go func() {
				defer func() {
					// nolint:staticcheck // SA9003: empty branch is intentional - we just want to catch panics without handling them
					recover()
					close(done)
				}()
				next.ServeHTTP(NewResponseWriter(tw), r)
			}()

			// Wait for either completion or timeout
			select {
			case <-done:
				tw.writeTo(w)
				return
			case <-ctx.Done():
				tw.expire()

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

// timeoutWriter buffers a handler's response so the middleware can send it,
// or a 408 in its place, without the handler writing to the client directly.
type timeoutWriter struct {
	header http.Header

	mu       sync.Mutex
	status   int
	body     bytes.Buffer
	timedOut bool
}

// Header returns the buffered header map; only the handler goroutine may use it.
func (tw *timeoutWriter) Header() http.Header {
	return tw.header
}

// WriteHeader records the first status code written before the timeout.
func (tw *timeoutWriter) WriteHeader(code int) {
	tw.mu.Lock()
	defer tw.mu.Unlock()

	if tw.timedOut || tw.status != 0 {
		return
	}
	tw.status = code
}

// Write buffers b, or returns http.ErrHandlerTimeout once the request has timed out.
func (tw *timeoutWriter) Write(b []byte) (int, error) {
	tw.mu.Lock()
	defer tw.mu.Unlock()

	if tw.timedOut {
		return 0, http.ErrHandlerTimeout
	}
	if tw.status == 0 {
		tw.status = http.StatusOK
	}
	return tw.body.Write(b)
}

// expire makes every later write fail with http.ErrHandlerTimeout.
func (tw *timeoutWriter) expire() {
	tw.mu.Lock()
	defer tw.mu.Unlock()

	tw.timedOut = true
}

// writeTo sends the buffered response to w; call it only after the handler returns.
func (tw *timeoutWriter) writeTo(w http.ResponseWriter) {
	maps.Copy(w.Header(), tw.header)
	w.WriteHeader(cmp.Or(tw.status, http.StatusOK))
	// A write error means the client went away; there is no one left to tell.
	_, _ = w.Write(tw.body.Bytes())
}
