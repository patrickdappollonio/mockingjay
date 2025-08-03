package middleware

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestTimeoutMiddleware(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	tests := []struct {
		name           string
		timeout        time.Duration
		handlerDelay   time.Duration
		expectTimeout  bool
		expectedStatus int
	}{
		{
			name:           "request completes before timeout",
			timeout:        100 * time.Millisecond,
			handlerDelay:   50 * time.Millisecond,
			expectTimeout:  false,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "request exceeds timeout",
			timeout:        50 * time.Millisecond,
			handlerDelay:   100 * time.Millisecond,
			expectTimeout:  true,
			expectedStatus: http.StatusRequestTimeout, // Middleware should return 408
		},
		{
			name:           "zero timeout uses default",
			timeout:        0, // Should use default 30s
			handlerDelay:   50 * time.Millisecond,
			expectTimeout:  false,
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create middleware
			middleware := NewTimeoutMiddleware(TimeoutConfig{
				Duration: tt.timeout,
			}, logger)

			// Create a slow handler
			finalHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Simulate work that might take time
				select {
				case <-time.After(tt.handlerDelay):
					w.WriteHeader(http.StatusOK)
					fmt.Fprintln(w, "Handler completed")
				case <-r.Context().Done():
					// Request was cancelled, don't write anything
					t.Log("Handler cancelled due to context")
					return
				}
			})

			// Create test chain
			chain := NewChain(middleware)
			handler := chain.Then(finalHandler)

			// Create test request
			req := httptest.NewRequest("GET", "/test", nil)
			rec := httptest.NewRecorder()

			// Execute request
			start := time.Now()
			handler.ServeHTTP(rec, req)
			duration := time.Since(start)

			// Check status code
			if rec.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, rec.Code)
			}

			// Verify timing behavior with our monitoring approach
			if tt.expectTimeout {
				// Request should take longer than the timeout
				if duration < tt.timeout {
					t.Errorf("Expected request to exceed timeout %v, but completed in %v", tt.timeout, duration)
				}
			} else {
				// Request should complete before timeout
				if tt.timeout > 0 && duration >= tt.timeout {
					t.Errorf("Request should have completed before timeout %v, but took %v", tt.timeout, duration)
				}
			}
		})
	}
}

func TestTimeoutMiddleware_Name(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	middleware := NewTimeoutMiddleware(TimeoutConfig{}, logger)

	if name := middleware.Name(); name != "timeout" {
		t.Errorf("Expected middleware name 'timeout', got %q", name)
	}
}
