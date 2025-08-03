package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// Mock middleware for testing
type mockMiddleware struct {
	name   string
	called bool
}

func (m *mockMiddleware) Name() string {
	return m.name
}

func (m *mockMiddleware) Handler() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			m.called = true
			w.Header().Set("X-Middleware-"+m.name, "called")
			next.ServeHTTP(w, r)
		})
	}
}

// Mock final handler
func mockFinalHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("X-Final-Handler", "called")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("final handler"))
}

func TestMiddlewareChain(t *testing.T) {
	// Create mock middlewares
	middleware1 := &mockMiddleware{name: "first"}
	middleware2 := &mockMiddleware{name: "second"}

	// Create chain using NewChain
	chain := NewChain(middleware1, middleware2)
	handler := chain.Then(http.HandlerFunc(mockFinalHandler))

	// Create test request
	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()

	// Execute chain
	handler.ServeHTTP(rr, req)

	// Verify response
	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	if body := rr.Body.String(); body != "final handler" {
		t.Errorf("expected 'final handler', got %s", body)
	}

	// Verify middlewares were called
	if !middleware1.called {
		t.Error("first middleware was not called")
	}

	if !middleware2.called {
		t.Error("second middleware was not called")
	}

	// Verify headers from middlewares
	if rr.Header().Get("X-Middleware-first") != "called" {
		t.Error("first middleware header not set")
	}

	if rr.Header().Get("X-Middleware-second") != "called" {
		t.Error("second middleware header not set")
	}

	if rr.Header().Get("X-Final-Handler") != "called" {
		t.Error("final handler header not set")
	}
}

func TestEmptyChain(t *testing.T) {
	// Create chain with no middleware
	chain := NewChain()
	handler := chain.Then(http.HandlerFunc(mockFinalHandler))

	// Create test request
	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()

	// Execute chain
	handler.ServeHTTP(rr, req)

	// Verify response
	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	if body := rr.Body.String(); body != "final handler" {
		t.Errorf("expected 'final handler', got %s", body)
	}

	// Verify final handler header
	if rr.Header().Get("X-Final-Handler") != "called" {
		t.Error("final handler header not set")
	}
}
