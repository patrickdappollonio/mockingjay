package middleware

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewBasicAuthMiddleware(t *testing.T) {
	tests := []struct {
		name      string
		config    BasicAuthConfig
		wantError bool
		errMsg    string
	}{
		{
			name: "valid basic config",
			config: BasicAuthConfig{
				Username: "admin",
				Password: "secret",
				Realm:    "Test Realm",
			},
			wantError: false,
		},
		{
			name: "default realm",
			config: BasicAuthConfig{
				Username: "admin",
				Password: "secret",
			},
			wantError: false,
		},
		{
			name: "with valid literal paths",
			config: BasicAuthConfig{
				Username: "admin",
				Password: "secret",
				Paths: BasicAuthPaths{
					Include: []string{"/admin", "/api/private"},
					Exclude: []string{"/admin/health"},
				},
			},
			wantError: false,
		},
		{
			name: "with valid regex paths",
			config: BasicAuthConfig{
				Username: "admin",
				Password: "secret",
				Paths: BasicAuthPaths{
					Include: []string{"/^/admin/.*$/", "/^/api/v\\d+/private$/"},
				},
			},
			wantError: false,
		},
		{
			name: "invalid regex pattern",
			config: BasicAuthConfig{
				Username: "admin",
				Password: "secret",
				Paths: BasicAuthPaths{
					Include: []string{"/^[invalid$/"},
				},
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			middleware, err := NewBasicAuthMiddleware(tt.config)

			if tt.wantError {
				if err == nil {
					t.Errorf("NewBasicAuthMiddleware() expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("NewBasicAuthMiddleware() unexpected error: %v", err)
				return
			}

			if middleware == nil {
				t.Errorf("NewBasicAuthMiddleware() returned nil middleware")
				return
			}

			if middleware.Name() != "basicauth" {
				t.Errorf("Name() = %v, want 'basicauth'", middleware.Name())
			}

			// Check default realm is set
			if tt.config.Realm == "" && middleware.config.Realm != "Restricted Area" {
				t.Errorf("Default realm not set correctly, got %v", middleware.config.Realm)
			}
		})
	}
}

func TestBasicAuthMiddleware_Handle(t *testing.T) {
	tests := []struct {
		name           string
		config         BasicAuthConfig
		requestPath    string
		authHeader     string
		expectedStatus int
		shouldCallNext bool
	}{
		{
			name: "valid credentials",
			config: BasicAuthConfig{
				Username: "admin",
				Password: "secret",
				Realm:    "Test Realm",
			},
			requestPath:    "/admin",
			authHeader:     makeBasicAuthHeader("admin", "secret"),
			expectedStatus: 0, // Next should be called, no status set
			shouldCallNext: true,
		},
		{
			name: "invalid username",
			config: BasicAuthConfig{
				Username: "admin",
				Password: "secret",
			},
			requestPath:    "/admin",
			authHeader:     makeBasicAuthHeader("wronguser", "secret"),
			expectedStatus: http.StatusUnauthorized,
			shouldCallNext: false,
		},
		{
			name: "invalid password",
			config: BasicAuthConfig{
				Username: "admin",
				Password: "secret",
			},
			requestPath:    "/admin",
			authHeader:     makeBasicAuthHeader("admin", "wrongpass"),
			expectedStatus: http.StatusUnauthorized,
			shouldCallNext: false,
		},
		{
			name: "missing auth header",
			config: BasicAuthConfig{
				Username: "admin",
				Password: "secret",
			},
			requestPath:    "/admin",
			authHeader:     "",
			expectedStatus: http.StatusUnauthorized,
			shouldCallNext: false,
		},
		{
			name: "path not included in auth - no include patterns",
			config: BasicAuthConfig{
				Username: "admin",
				Password: "secret",
				Paths: BasicAuthPaths{
					Exclude: []string{"/public"},
				},
			},
			requestPath:    "/admin",
			authHeader:     "",
			expectedStatus: http.StatusUnauthorized,
			shouldCallNext: false,
		},
		{
			name: "path excluded from auth",
			config: BasicAuthConfig{
				Username: "admin",
				Password: "secret",
				Paths: BasicAuthPaths{
					Exclude: []string{"/public"},
				},
			},
			requestPath:    "/public",
			authHeader:     "",
			expectedStatus: 0, // Next should be called, no auth required
			shouldCallNext: true,
		},
		{
			name: "path not in include list",
			config: BasicAuthConfig{
				Username: "admin",
				Password: "secret",
				Paths: BasicAuthPaths{
					Include: []string{"/admin"},
				},
			},
			requestPath:    "/public",
			authHeader:     "",
			expectedStatus: 0, // Next should be called, not in include list
			shouldCallNext: true,
		},
		{
			name: "path in include list, valid auth",
			config: BasicAuthConfig{
				Username: "admin",
				Password: "secret",
				Paths: BasicAuthPaths{
					Include: []string{"/admin"},
				},
			},
			requestPath:    "/admin",
			authHeader:     makeBasicAuthHeader("admin", "secret"),
			expectedStatus: 0, // Next should be called
			shouldCallNext: true,
		},
		{
			name: "path in include list, invalid auth",
			config: BasicAuthConfig{
				Username: "admin",
				Password: "secret",
				Paths: BasicAuthPaths{
					Include: []string{"/admin"},
				},
			},
			requestPath:    "/admin",
			authHeader:     makeBasicAuthHeader("admin", "wrong"),
			expectedStatus: http.StatusUnauthorized,
			shouldCallNext: false,
		},
		{
			name: "regex include pattern matches",
			config: BasicAuthConfig{
				Username: "admin",
				Password: "secret",
				Paths: BasicAuthPaths{
					Include: []string{"/^/admin/.*$/"},
				},
			},
			requestPath:    "/admin/users",
			authHeader:     makeBasicAuthHeader("admin", "secret"),
			expectedStatus: 0, // Next should be called
			shouldCallNext: true,
		},
		{
			name: "exclude takes precedence over include",
			config: BasicAuthConfig{
				Username: "admin",
				Password: "secret",
				Paths: BasicAuthPaths{
					Include: []string{"/admin"},
					Exclude: []string{"/admin"},
				},
			},
			requestPath:    "/admin",
			authHeader:     "",
			expectedStatus: 0, // Next should be called, exclude takes precedence
			shouldCallNext: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			middleware, err := NewBasicAuthMiddleware(tt.config)
			if err != nil {
				t.Fatalf("NewBasicAuthMiddleware() error: %v", err)
			}

			// Create request
			req := httptest.NewRequest(http.MethodGet, tt.requestPath, nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}

			// Create response recorder
			w := httptest.NewRecorder()

			// Track if next was called
			nextCalled := false
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				nextCalled = true
			})

			// Execute middleware using Handler pattern
			handler := middleware.Handler()(next)
			handler.ServeHTTP(w, req)

			// Check if next was called as expected
			if nextCalled != tt.shouldCallNext {
				t.Errorf("next() called = %v, want %v", nextCalled, tt.shouldCallNext)
			}

			// Check status code if expected
			if tt.expectedStatus != 0 {
				if w.Code != tt.expectedStatus {
					t.Errorf("Status code = %v, want %v", w.Code, tt.expectedStatus)
				}

				// Check WWW-Authenticate header for 401 responses
				if tt.expectedStatus == http.StatusUnauthorized {
					authHeader := w.Header().Get("WWW-Authenticate")
					expectedRealm := tt.config.Realm
					if expectedRealm == "" {
						expectedRealm = "Restricted Area"
					}
					expectedAuth := fmt.Sprintf(`Basic realm="%s"`, expectedRealm)
					if authHeader != expectedAuth {
						t.Errorf("WWW-Authenticate = %v, want %v", authHeader, expectedAuth)
					}
				}
			}
		})
	}
}

func TestBasicAuthMiddleware_PathMatching(t *testing.T) {
	tests := []struct {
		name              string
		includePaths      []string
		excludePaths      []string
		testPath          string
		shouldRequireAuth bool
	}{
		{
			name:              "no patterns - auth required",
			includePaths:      []string{},
			excludePaths:      []string{},
			testPath:          "/any/path",
			shouldRequireAuth: true,
		},
		{
			name:              "literal include match",
			includePaths:      []string{"/admin"},
			excludePaths:      []string{},
			testPath:          "/admin",
			shouldRequireAuth: true,
		},
		{
			name:              "literal include no match",
			includePaths:      []string{"/admin"},
			excludePaths:      []string{},
			testPath:          "/public",
			shouldRequireAuth: false,
		},
		{
			name:              "regex include match",
			includePaths:      []string{"/^/api/v\\d+/.*$/"},
			excludePaths:      []string{},
			testPath:          "/api/v1/users",
			shouldRequireAuth: true,
		},
		{
			name:              "regex include no match",
			includePaths:      []string{"/^/api/v\\d+/.*$/"},
			excludePaths:      []string{},
			testPath:          "/api/users",
			shouldRequireAuth: false,
		},
		{
			name:              "exclude overrides include",
			includePaths:      []string{"/admin"},
			excludePaths:      []string{"/admin"},
			testPath:          "/admin",
			shouldRequireAuth: false,
		},
		{
			name:              "exclude with regex",
			includePaths:      []string{},
			excludePaths:      []string{"/^/health.*$/"},
			testPath:          "/health/check",
			shouldRequireAuth: false,
		},
		{
			name:              "multiple includes",
			includePaths:      []string{"/admin", "/api"},
			excludePaths:      []string{},
			testPath:          "/api",
			shouldRequireAuth: true,
		},
		{
			name:              "multiple excludes",
			includePaths:      []string{},
			excludePaths:      []string{"/health", "/ping"},
			testPath:          "/ping",
			shouldRequireAuth: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := BasicAuthConfig{
				Username: "admin",
				Password: "secret",
				Paths: BasicAuthPaths{
					Include: tt.includePaths,
					Exclude: tt.excludePaths,
				},
			}

			middleware, err := NewBasicAuthMiddleware(config)
			if err != nil {
				t.Fatalf("NewBasicAuthMiddleware() error: %v", err)
			}

			shouldAuth := middleware.shouldAuthenticate(tt.testPath)
			if shouldAuth != tt.shouldRequireAuth {
				t.Errorf("shouldAuthenticate(%q) = %v, want %v", tt.testPath, shouldAuth, tt.shouldRequireAuth)
			}
		})
	}
}

// Helper function to create Basic Auth header
func makeBasicAuthHeader(username, password string) string {
	auth := username + ":" + password
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(auth))
}
