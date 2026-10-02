package middleware

import (
	"errors"
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

func TestTimeoutMiddleware_HandlerOutput(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	t.Run("completed response is passed through", func(t *testing.T) {
		var handlerWriter http.ResponseWriter
		handler := NewChain(NewTimeoutMiddleware(TimeoutConfig{Duration: time.Second}, logger)).
			Then(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				handlerWriter = w
				w.Header().Set("X-Test", "yes")
				w.WriteHeader(http.StatusCreated)
				fmt.Fprint(w, "created")
			}))
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, httptest.NewRequest("GET", "/test", nil))

		if rec.Code != http.StatusCreated {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusCreated)
		}
		if got := rec.Header().Get("X-Test"); got != "yes" {
			t.Errorf("X-Test header = %q, want %q", got, "yes")
		}
		if got := rec.Body.String(); got != "created" {
			t.Errorf("body = %q, want %q", got, "created")
		}
		if _, ok := handlerWriter.(*ResponseWriter); !ok {
			t.Errorf("handler after the timeout middleware got %T, want *ResponseWriter so the logger can read the status", handlerWriter)
		}
	})

	t.Run("body without an explicit status is a 200", func(t *testing.T) {
		handler := NewChain(NewTimeoutMiddleware(TimeoutConfig{Duration: time.Second}, logger)).
			Then(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				fmt.Fprint(w, "ok")
			}))
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, httptest.NewRequest("GET", "/test", nil))

		if rec.Code != http.StatusOK || rec.Body.String() != "ok" {
			t.Errorf("response = %d %q, want 200 %q", rec.Code, rec.Body.String(), "ok")
		}
	})

	t.Run("handler writing after the timeout cannot reach the client", func(t *testing.T) {
		lateWriteErr := make(chan error, 1)
		handler := NewChain(NewTimeoutMiddleware(TimeoutConfig{Duration: 20 * time.Millisecond}, logger)).
			Then(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				time.Sleep(100 * time.Millisecond) // ignores the request context on purpose
				w.Header().Set("X-Late", "yes")
				w.WriteHeader(http.StatusOK)
				_, err := w.Write([]byte("late body"))
				lateWriteErr <- err
			}))
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, httptest.NewRequest("GET", "/test", nil))
		err := <-lateWriteErr

		if rec.Code != http.StatusRequestTimeout {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusRequestTimeout)
		}
		if want := "408 Request Timeout\n\nThe request exceeded the configured timeout."; rec.Body.String() != want {
			t.Errorf("body = %q, want %q", rec.Body.String(), want)
		}
		if got := rec.Header().Get("X-Late"); got != "" {
			t.Errorf("X-Late header = %q, want it absent", got)
		}
		if !errors.Is(err, http.ErrHandlerTimeout) {
			t.Errorf("late Write() error = %v, want %v", err, http.ErrHandlerTimeout)
		}
	})
}
