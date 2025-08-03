package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCORSMiddleware(t *testing.T) {
	// Create CORS middleware with test config
	config := CORSConfig{
		AllowOrigins:     []string{"http://localhost:3000", "https://example.com"},
		AllowMethods:     []string{"GET", "POST", "PUT"},
		AllowHeaders:     []string{"Content-Type", "Authorization"},
		AllowCredentials: true,
		MaxAge:           3600,
	}
	corsMiddleware := NewCORSMiddleware(config)

	// Mock final handler
	finalHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("response"))
	})

	// Create middleware chain
	chain := NewChain(corsMiddleware)
	handler := chain.Then(finalHandler)

	tests := []struct {
		name             string
		method           string
		origin           string
		expectedStatus   int
		expectedOrigin   string
		shouldHaveOrigin bool
	}{
		{
			name:             "allowed origin",
			method:           "GET",
			origin:           "http://localhost:3000",
			expectedStatus:   200,
			expectedOrigin:   "http://localhost:3000",
			shouldHaveOrigin: true,
		},
		{
			name:             "unauthorized origin",
			method:           "GET",
			origin:           "http://unauthorized.com",
			expectedStatus:   200,
			expectedOrigin:   "",
			shouldHaveOrigin: false,
		},
		{
			name:             "OPTIONS preflight",
			method:           "OPTIONS",
			origin:           "http://localhost:3000",
			expectedStatus:   204,
			expectedOrigin:   "http://localhost:3000",
			shouldHaveOrigin: true,
		},
		{
			name:             "no origin header",
			method:           "GET",
			origin:           "",
			expectedStatus:   200,
			expectedOrigin:   "",
			shouldHaveOrigin: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "/test", nil)
			if tt.origin != "" {
				req.Header.Set("Origin", tt.origin)
			}
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			// Check status code
			if rr.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rr.Code)
			}

			// Check CORS headers
			if tt.shouldHaveOrigin {
				if origin := rr.Header().Get("Access-Control-Allow-Origin"); origin != tt.expectedOrigin {
					t.Errorf("expected Access-Control-Allow-Origin %s, got %s", tt.expectedOrigin, origin)
				}
			} else {
				if origin := rr.Header().Get("Access-Control-Allow-Origin"); origin != "" {
					t.Errorf("expected no Access-Control-Allow-Origin header, got %s", origin)
				}
			}

			// Check other CORS headers are present
			if methods := rr.Header().Get("Access-Control-Allow-Methods"); methods == "" {
				t.Error("Access-Control-Allow-Methods header not set")
			}

			if headers := rr.Header().Get("Access-Control-Allow-Headers"); headers == "" {
				t.Error("Access-Control-Allow-Headers header not set")
			}

			if credentials := rr.Header().Get("Access-Control-Allow-Credentials"); credentials != "true" {
				t.Errorf("expected Access-Control-Allow-Credentials true, got %s", credentials)
			}

			if maxAge := rr.Header().Get("Access-Control-Max-Age"); maxAge != "3600" {
				t.Errorf("expected Access-Control-Max-Age 3600, got %s", maxAge)
			}
		})
	}
}

func TestCORSDefaults(t *testing.T) {
	// Create CORS middleware with empty config to test defaults
	corsMiddleware := NewCORSMiddleware(CORSConfig{})

	// Mock final handler
	finalHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Create middleware chain
	chain := NewChain(corsMiddleware)
	handler := chain.Then(finalHandler)

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Origin", "http://any-origin.com")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	// Should allow any origin due to default "*"
	if origin := rr.Header().Get("Access-Control-Allow-Origin"); origin != "*" {
		t.Errorf("expected Access-Control-Allow-Origin *, got %s", origin)
	}

	// Check default methods
	methods := rr.Header().Get("Access-Control-Allow-Methods")
	expectedMethods := "GET, POST, PUT, DELETE, OPTIONS"
	if methods != expectedMethods {
		t.Errorf("expected methods %s, got %s", expectedMethods, methods)
	}
}
