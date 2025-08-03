package router

import (
	"html/template"
	"os"
	"strings"
	"testing"

	"github.com/patrickdappollonio/mockingjay/internal/config"
)

func TestCompiler_CompileRoute_LiteralPaths(t *testing.T) {
	compiler := NewCompiler()

	tests := []struct {
		name        string
		routeConfig config.RouteConfig
		wantErr     bool
	}{
		{
			name: "simple literal path",
			routeConfig: config.RouteConfig{
				Path:     "/healthz",
				Verb:     "GET",
				Template: "OK",
			},
			wantErr: false,
		},
		{
			name: "literal path with parameters",
			routeConfig: config.RouteConfig{
				Path:     "/users/123",
				Verb:     "GET",
				Template: "User 123",
			},
			wantErr: false,
		},
		{
			name: "root path",
			routeConfig: config.RouteConfig{
				Path:     "/",
				Verb:     "GET",
				Template: "Root",
			},
			wantErr: false,
		},
		{
			name: "nested path",
			routeConfig: config.RouteConfig{
				Path:     "/api/v1/users",
				Verb:     "POST",
				Template: "Create user",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			route, err := compiler.CompileRoute(tt.routeConfig)
			if (err != nil) != tt.wantErr {
				t.Errorf("CompileRoute() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if route == nil {
					t.Error("CompileRoute() returned nil route for valid input")
					return
				}

				// Verify route properties
				if route.Pattern != tt.routeConfig.Path {
					t.Errorf("CompileRoute() Pattern = %v, want %v", route.Pattern, tt.routeConfig.Path)
				}

				if route.Verb != tt.routeConfig.GetNormalizedVerb() {
					t.Errorf("CompileRoute() Verb = %v, want %v", route.Verb, tt.routeConfig.GetNormalizedVerb())
				}

				if route.IsRegexp {
					t.Error("CompileRoute() IsRegexp should be false for literal paths")
				}

				if route.Regex != nil {
					t.Error("CompileRoute() Regex should be nil for literal paths")
				}

				if route.Tmpl == nil {
					t.Error("CompileRoute() Template should not be nil")
				}

				if route.TemplateSource != "inline" {
					t.Errorf("CompileRoute() TemplateSource = %v, want 'inline'", route.TemplateSource)
				}
			}
		})
	}
}

func TestCompiler_CompileRoute_RegexPaths(t *testing.T) {
	compiler := NewCompiler()

	tests := []struct {
		name        string
		routeConfig config.RouteConfig
		wantErr     bool
		expectNames []string
	}{
		{
			name: "simple regex pattern",
			routeConfig: config.RouteConfig{
				Path:     "/^/user/[0-9]+$/",
				Verb:     "GET",
				Template: "User found",
			},
			wantErr:     false,
			expectNames: []string{},
		},
		{
			name: "regex with single named group",
			routeConfig: config.RouteConfig{
				Path:     "/^/user/(?P<id>[0-9]+)$/",
				Verb:     "GET",
				Template: "User {{.Params.id}}",
			},
			wantErr:     false,
			expectNames: []string{"", "id"},
		},
		{
			name: "regex with multiple named groups",
			routeConfig: config.RouteConfig{
				Path:     "/^/user/(?P<id>[0-9]+)/posts/(?P<postId>[0-9]+)$/",
				Verb:     "GET",
				Template: "User {{.Params.id}} Post {{.Params.postId}}",
			},
			wantErr:     false,
			expectNames: []string{"", "id", "postId"},
		},
		{
			name: "regex with optional parts",
			routeConfig: config.RouteConfig{
				Path:     "/^/api/v(?P<version>[12])/users(/(?P<id>[0-9]+))?$/",
				Verb:     "GET",
				Template: "API v{{.Params.version}}",
			},
			wantErr:     false,
			expectNames: []string{"", "version", "", "id"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			route, err := compiler.CompileRoute(tt.routeConfig)
			if (err != nil) != tt.wantErr {
				t.Errorf("CompileRoute() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if route == nil {
					t.Error("CompileRoute() returned nil route for valid input")
					return
				}

				// Verify route properties
				if route.Pattern != tt.routeConfig.Path {
					t.Errorf("CompileRoute() Pattern = %v, want %v", route.Pattern, tt.routeConfig.Path)
				}

				if !route.IsRegexp {
					t.Error("CompileRoute() IsRegexp should be true for regex paths")
				}

				if route.Regex == nil {
					t.Error("CompileRoute() Regex should not be nil for regex paths")
				}

				// Test that the regex was compiled correctly
				expectedPattern := tt.routeConfig.GetRegexPattern()
				actualPattern := route.Regex.String()
				if actualPattern != expectedPattern {
					t.Errorf("CompileRoute() Regex pattern = %v, want %v", actualPattern, expectedPattern)
				}

				// Test named capture groups
				if len(tt.expectNames) > 0 {
					actualNames := route.Regex.SubexpNames()
					if len(actualNames) != len(tt.expectNames) {
						t.Errorf("CompileRoute() SubexpNames length = %v, want %v", len(actualNames), len(tt.expectNames))
					} else {
						for i, expected := range tt.expectNames {
							if i < len(actualNames) && actualNames[i] != expected {
								t.Errorf("CompileRoute() SubexpNames[%d] = %v, want %v", i, actualNames[i], expected)
							}
						}
					}
				}

				if route.Tmpl == nil {
					t.Error("CompileRoute() Template should not be nil")
				}
			}
		})
	}
}

func TestCompiler_CompileRoute_InvalidRegexPatterns(t *testing.T) {
	compiler := NewCompiler()

	tests := []struct {
		name        string
		routeConfig config.RouteConfig
		wantErrMsg  string
	}{
		{
			name: "unclosed bracket",
			routeConfig: config.RouteConfig{
				Path:     "/[invalid/",
				Verb:     "GET",
				Template: "test",
			},
			wantErrMsg: "failed to compile regex pattern",
		},
		{
			name: "invalid group syntax",
			routeConfig: config.RouteConfig{
				Path:     "/[)/",
				Verb:     "GET",
				Template: "test",
			},
			wantErrMsg: "failed to compile regex pattern",
		},
		{
			name: "unbalanced parentheses",
			routeConfig: config.RouteConfig{
				Path:     "/user/((id/",
				Verb:     "GET",
				Template: "test",
			},
			wantErrMsg: "failed to compile regex pattern",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			route, err := compiler.CompileRoute(tt.routeConfig)
			if err == nil {
				t.Error("CompileRoute() expected error but got none")
				return
			}

			if route != nil {
				t.Error("CompileRoute() should return nil route on error")
			}

			if !strings.Contains(err.Error(), tt.wantErrMsg) {
				t.Errorf("CompileRoute() error = %v, want error containing %q", err, tt.wantErrMsg)
			}
		})
	}
}

func TestCompiler_CompileRoute_TemplateFile(t *testing.T) {
	compiler := NewCompiler()

	// Create a temporary template file
	tmpFile := createTempTemplateFile(t, "Hello {{.Params.name}}")
	defer removeFile(tmpFile)

	routeConfig := config.RouteConfig{
		Path:         "/user/(?P<name>[a-zA-Z]+)/",
		Verb:         "GET",
		TemplateFile: tmpFile,
	}

	route, err := compiler.CompileRoute(routeConfig)
	if err != nil {
		t.Errorf("CompileRoute() error = %v, expected no error", err)
		return
	}

	if route == nil {
		t.Error("CompileRoute() returned nil route")
		return
	}

	if route.TemplateSource != tmpFile {
		t.Errorf("CompileRoute() TemplateSource = %v, want %v", route.TemplateSource, tmpFile)
	}

	if route.Tmpl == nil {
		t.Error("CompileRoute() Template should not be nil")
	}
}

func TestCompiler_CompileRoute_InvalidTemplate(t *testing.T) {
	compiler := NewCompiler()

	tests := []struct {
		name        string
		routeConfig config.RouteConfig
		wantErrMsg  string
	}{
		{
			name: "invalid template syntax",
			routeConfig: config.RouteConfig{
				Path:     "/test",
				Verb:     "GET",
				Template: "{{.InvalidSyntax",
			},
			wantErrMsg: "failed to compile template",
		},
		{
			name: "non-existent template file",
			routeConfig: config.RouteConfig{
				Path:         "/test",
				Verb:         "GET",
				TemplateFile: "/nonexistent/file.tmpl",
			},
			wantErrMsg: "failed to compile template",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			route, err := compiler.CompileRoute(tt.routeConfig)
			if err == nil {
				t.Error("CompileRoute() expected error but got none")
				return
			}

			if route != nil {
				t.Error("CompileRoute() should return nil route on error")
			}

			if !strings.Contains(err.Error(), tt.wantErrMsg) {
				t.Errorf("CompileRoute() error = %v, want error containing %q", err, tt.wantErrMsg)
			}
		})
	}
}

func TestCompiler_CompileRoutes_MultipleRoutes(t *testing.T) {
	compiler := NewCompiler()

	routeConfigs := []config.RouteConfig{
		{
			Path:     "/health",
			Verb:     "GET",
			Template: "OK",
		},
		{
			Path:     "/^/user/(?P<id>[0-9]+)$/",
			Verb:     "GET",
			Template: "User {{.Params.id}}",
		},
		{
			Path:     "/api/v1/posts",
			Verb:     "POST",
			Template: "Created post",
		},
	}

	routes, err := compiler.CompileRoutes(routeConfigs)
	if err != nil {
		t.Errorf("CompileRoutes() error = %v, expected no error", err)
		return
	}

	if len(routes) != len(routeConfigs) {
		t.Errorf("CompileRoutes() returned %d routes, want %d", len(routes), len(routeConfigs))
		return
	}

	// Verify each route was compiled correctly
	for i, route := range routes {
		expectedConfig := routeConfigs[i]

		if route.Pattern != expectedConfig.Path {
			t.Errorf("Route[%d] Pattern = %v, want %v", i, route.Pattern, expectedConfig.Path)
		}

		if route.Verb != expectedConfig.GetNormalizedVerb() {
			t.Errorf("Route[%d] Verb = %v, want %v", i, route.Verb, expectedConfig.GetNormalizedVerb())
		}

		if route.Tmpl == nil {
			t.Errorf("Route[%d] Template should not be nil", i)
		}

		// Verify regex compilation for regex routes
		if expectedConfig.IsRegexPattern() {
			if !route.IsRegexp {
				t.Errorf("Route[%d] should be regex route", i)
			}
			if route.Regex == nil {
				t.Errorf("Route[%d] Regex should not be nil", i)
			}
		} else {
			if route.IsRegexp {
				t.Errorf("Route[%d] should not be regex route", i)
			}
			if route.Regex != nil {
				t.Errorf("Route[%d] Regex should be nil", i)
			}
		}
	}
}

func TestCompiler_CompileRoutes_ErrorInMiddle(t *testing.T) {
	compiler := NewCompiler()

	routeConfigs := []config.RouteConfig{
		{
			Path:     "/health",
			Verb:     "GET",
			Template: "OK",
		},
		{
			Path:     "/[invalid regex/", // This will cause an error
			Verb:     "GET",
			Template: "Invalid",
		},
		{
			Path:     "/users",
			Verb:     "GET",
			Template: "Users",
		},
	}

	routes, err := compiler.CompileRoutes(routeConfigs)
	if err == nil {
		t.Error("CompileRoutes() expected error but got none")
		return
	}

	if routes != nil {
		t.Error("CompileRoutes() should return nil routes on error")
	}

	// Error should mention which route failed
	if !strings.Contains(err.Error(), "route 1") {
		t.Errorf("CompileRoutes() error should mention route index, got: %v", err)
	}
}

func TestCompiler_GetEngine(t *testing.T) {
	compiler := NewCompiler()
	engine := compiler.GetEngine()

	if engine == nil {
		t.Error("GetEngine() should not return nil")
	}

	// Verify the engine has the expected function map
	funcMap := engine.GetFuncMap()
	expectedFuncs := []string{"trimPrefix", "header", "query", "jsonBody"}

	for _, funcName := range expectedFuncs {
		if _, exists := funcMap[funcName]; !exists {
			t.Errorf("GetEngine() function map missing expected function: %s", funcName)
		}
	}
}

func TestSanitizeTemplateName(t *testing.T) {
	tests := []struct {
		name string
		path string
		want string
	}{
		{
			name: "simple path",
			path: "/users",
			want: "users",
		},
		{
			name: "path with parameters",
			path: "/users/{id}",
			want: "users_id",
		},
		{
			name: "regex pattern",
			path: "/^/user/(?P<id>[0-9]+)$/",
			want: "user_P<id>_0_9",
		},
		{
			name: "complex regex",
			path: "/api/v[12]/users+",
			want: "api_v_12_users",
		},
		{
			name: "path with multiple consecutive special chars",
			path: "///***///",
			want: "unnamed",
		},
		{
			name: "empty path",
			path: "",
			want: "unnamed",
		},
		{
			name: "path with spaces",
			path: "/ user / profile /",
			want: "user_profile",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := sanitizeTemplateName(tt.path); got != tt.want {
				t.Errorf("sanitizeTemplateName() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Benchmark for route compilation performance
func BenchmarkCompiler_CompileRoute_Literal(b *testing.B) {
	compiler := NewCompiler()
	routeConfig := config.RouteConfig{
		Path:     "/api/v1/users",
		Verb:     "GET",
		Template: "Users list",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := compiler.CompileRoute(routeConfig)
		if err != nil {
			b.Fatalf("CompileRoute() error = %v", err)
		}
	}
}

func BenchmarkCompiler_CompileRoute_Regex(b *testing.B) {
	compiler := NewCompiler()
	routeConfig := config.RouteConfig{
		Path:     "/^/user/(?P<id>[0-9]+)/posts/(?P<postId>[0-9]+)$/",
		Verb:     "GET",
		Template: "User {{.Params.id}} Post {{.Params.postId}}",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := compiler.CompileRoute(routeConfig)
		if err != nil {
			b.Fatalf("CompileRoute() error = %v", err)
		}
	}
}

// Helper functions for testing
func createTempTemplateFile(t *testing.T, content string) string {
	tmpFile, err := os.CreateTemp("", "template_test_*.tmpl")
	if err != nil {
		t.Fatalf("Failed to create temp template file: %v", err)
	}
	defer tmpFile.Close()

	if _, err := tmpFile.WriteString(content); err != nil {
		t.Fatalf("Failed to write to temp template file: %v", err)
	}

	return tmpFile.Name()
}

func removeFile(filename string) {
	os.Remove(filename)
}

func TestCompiler_CompileHeaderMatchers(t *testing.T) {
	tests := []struct {
		name         string
		matchHeaders map[string]string
		wantErr      bool
		validate     func(t *testing.T, headers map[string]*HeaderMatcher)
	}{
		{
			name:         "no headers",
			matchHeaders: nil,
			wantErr:      false,
			validate: func(t *testing.T, headers map[string]*HeaderMatcher) {
				if headers != nil {
					t.Errorf("Expected nil headers, got %v", headers)
				}
			},
		},
		{
			name:         "empty headers",
			matchHeaders: map[string]string{},
			wantErr:      false,
			validate: func(t *testing.T, headers map[string]*HeaderMatcher) {
				if headers != nil {
					t.Errorf("Expected nil headers, got %v", headers)
				}
			},
		},
		{
			name: "literal header match",
			matchHeaders: map[string]string{
				"Content-Type": "application/json",
				"X-API-Key":    "secret123",
			},
			wantErr: false,
			validate: func(t *testing.T, headers map[string]*HeaderMatcher) {
				if len(headers) != 2 {
					t.Errorf("Expected 2 headers, got %d", len(headers))
				}

				// Check Content-Type header (canonicalized to lowercase)
				if matcher, ok := headers["content-type"]; ok {
					if matcher.IsRegex {
						t.Error("Expected literal match for Content-Type")
					}
					if matcher.Literal != "application/json" {
						t.Errorf("Expected literal value %q, got %q", "application/json", matcher.Literal)
					}
					if matcher.Regex != nil {
						t.Error("Expected nil regex for literal match")
					}
				} else {
					t.Error("Content-Type header not found")
				}

				// Check X-API-Key header (canonicalized to lowercase)
				if matcher, ok := headers["x-api-key"]; ok {
					if matcher.IsRegex {
						t.Error("Expected literal match for X-API-Key")
					}
					if matcher.Literal != "secret123" {
						t.Errorf("Expected literal value %q, got %q", "secret123", matcher.Literal)
					}
				} else {
					t.Error("X-API-Key header not found")
				}
			},
		},
		{
			name: "regex header match",
			matchHeaders: map[string]string{
				"Authorization": "/Bearer .+/",
				"User-Agent":    "/Mozilla.*/",
			},
			wantErr: false,
			validate: func(t *testing.T, headers map[string]*HeaderMatcher) {
				if len(headers) != 2 {
					t.Errorf("Expected 2 headers, got %d", len(headers))
				}

				// Check Authorization header
				if matcher, ok := headers["authorization"]; ok {
					if !matcher.IsRegex {
						t.Error("Expected regex match for Authorization")
					}
					if matcher.Literal != "" {
						t.Errorf("Expected empty literal for regex match, got %q", matcher.Literal)
					}
					if matcher.Regex == nil {
						t.Error("Expected non-nil regex for regex match")
					} else {
						// Test the regex
						if !matcher.Regex.MatchString("Bearer abc123") {
							t.Error("Regex should match 'Bearer abc123'")
						}
						if matcher.Regex.MatchString("Basic abc123") {
							t.Error("Regex should not match 'Basic abc123'")
						}
					}
				} else {
					t.Error("Authorization header not found")
				}
			},
		},
		{
			name: "mixed literal and regex",
			matchHeaders: map[string]string{
				"Content-Type":  "application/json",
				"Authorization": "/Bearer .+/",
			},
			wantErr: false,
			validate: func(t *testing.T, headers map[string]*HeaderMatcher) {
				if len(headers) != 2 {
					t.Errorf("Expected 2 headers, got %d", len(headers))
				}

				// Content-Type should be literal
				if matcher, ok := headers["content-type"]; ok {
					if matcher.IsRegex {
						t.Error("Expected literal match for Content-Type")
					}
				}

				// Authorization should be regex
				if matcher, ok := headers["authorization"]; ok {
					if !matcher.IsRegex {
						t.Error("Expected regex match for Authorization")
					}
				}
			},
		},
		{
			name: "invalid regex pattern",
			matchHeaders: map[string]string{
				"Authorization": "/[unclosed/",
			},
			wantErr: true,
			validate: func(t *testing.T, headers map[string]*HeaderMatcher) {
				// Should not reach here due to error
			},
		},
		{
			name: "regex without slashes treated as literal",
			matchHeaders: map[string]string{
				"Authorization": "Bearer token123",
			},
			wantErr: false,
			validate: func(t *testing.T, headers map[string]*HeaderMatcher) {
				if matcher, ok := headers["authorization"]; ok {
					if matcher.IsRegex {
						t.Error("Expected literal match for header without regex slashes")
					}
					if matcher.Literal != "Bearer token123" {
						t.Errorf("Expected literal value %q, got %q", "Bearer token123", matcher.Literal)
					}
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compiler := NewCompiler()

			routeConfig := config.RouteConfig{
				Path:         "/test",
				Verb:         "GET",
				Template:     "test template",
				MatchHeaders: tt.matchHeaders,
			}

			route, err := compiler.CompileRoute(routeConfig)
			hasErr := err != nil

			if hasErr != tt.wantErr {
				t.Errorf("Compiler.CompileRoute() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && route != nil {
				tt.validate(t, route.MatchHeaders)
			}
		})
	}
}

func TestCanonicalizeHeaderName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "already lowercase",
			input:    "content-type",
			expected: "content-type",
		},
		{
			name:     "mixed case",
			input:    "Content-Type",
			expected: "content-type",
		},
		{
			name:     "uppercase",
			input:    "AUTHORIZATION",
			expected: "authorization",
		},
		{
			name:     "with spaces",
			input:    "  X-API-Key  ",
			expected: "x-api-key",
		},
		{
			name:     "custom header",
			input:    "X-Custom-Header",
			expected: "x-custom-header",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := canonicalizeHeaderName(tt.input)
			if got != tt.expected {
				t.Errorf("canonicalizeHeaderName(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestIsHeaderRegexPattern(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "valid regex pattern",
			input:    "/Bearer .+/",
			expected: true,
		},
		{
			name:     "empty regex pattern",
			input:    "//",
			expected: false,
		},
		{
			name:     "single slash",
			input:    "/",
			expected: false,
		},
		{
			name:     "no slashes",
			input:    "Bearer token",
			expected: false,
		},
		{
			name:     "only start slash",
			input:    "/Bearer token",
			expected: false,
		},
		{
			name:     "only end slash",
			input:    "Bearer token/",
			expected: false,
		},
		{
			name:     "complex regex",
			input:    "/^Bearer [a-zA-Z0-9]+$/",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isHeaderRegexPattern(tt.input)
			if got != tt.expected {
				t.Errorf("isHeaderRegexPattern(%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestCompiler_CompileResponseHeaders(t *testing.T) {
	tests := []struct {
		name            string
		responseHeaders map[string]string
		wantErr         bool
		errContains     string
		validate        func(t *testing.T, headers map[string]*template.Template)
	}{
		{
			name:            "no response headers",
			responseHeaders: nil,
			wantErr:         false,
			validate: func(t *testing.T, headers map[string]*template.Template) {
				if headers != nil {
					t.Errorf("expected nil response headers, got %v", headers)
				}
			},
		},
		{
			name:            "empty response headers",
			responseHeaders: map[string]string{},
			wantErr:         false,
			validate: func(t *testing.T, headers map[string]*template.Template) {
				if headers != nil {
					t.Errorf("expected nil response headers, got %v", headers)
				}
			},
		},
		{
			name: "simple literal headers",
			responseHeaders: map[string]string{
				"Content-Type":  "application/json",
				"X-API-Version": "v1",
			},
			wantErr: false,
			validate: func(t *testing.T, headers map[string]*template.Template) {
				if len(headers) != 2 {
					t.Errorf("expected 2 response headers, got %d", len(headers))
				}

				if _, exists := headers["content-type"]; !exists {
					t.Error("expected Content-Type header to be compiled")
				}

				if _, exists := headers["x-api-version"]; !exists {
					t.Error("expected X-API-Version header to be compiled")
				}
			},
		},
		{
			name: "template headers",
			responseHeaders: map[string]string{
				"X-Request-ID": "{{ index .Headers \"X-Request-ID\" }}",
				"X-User-Agent": "{{ .Request.Header.Get \"User-Agent\" }}",
			},
			wantErr: false,
			validate: func(t *testing.T, headers map[string]*template.Template) {
				if len(headers) != 2 {
					t.Errorf("expected 2 response headers, got %d", len(headers))
				}

				if _, exists := headers["x-request-id"]; !exists {
					t.Error("expected X-Request-ID header to be compiled")
				}

				if _, exists := headers["x-user-agent"]; !exists {
					t.Error("expected X-User-Agent header to be compiled")
				}
			},
		},
		{
			name: "invalid template syntax",
			responseHeaders: map[string]string{
				"X-Custom": "{{ .Headers.Test",
			},
			wantErr:     true,
			errContains: "failed to compile response header template",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compiler := NewCompiler()

			routeConfig := config.RouteConfig{
				Path:            "/test",
				Verb:            "GET",
				Template:        "test",
				ResponseHeaders: tt.responseHeaders,
			}

			route, err := compiler.CompileRoute(routeConfig)
			hasErr := err != nil

			if hasErr != tt.wantErr {
				t.Errorf("CompileRoute() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errContains != "" {
				if err == nil || !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("CompileRoute() error = %v, want error containing %q", err, tt.errContains)
				}
				return
			}

			if !tt.wantErr && tt.validate != nil {
				tt.validate(t, route.ResponseHeaders)
			}
		})
	}
}
