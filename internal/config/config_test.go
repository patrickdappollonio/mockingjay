package config

import (
	"errors"
	"os"
	"strings"
	"testing"
)

func TestLoadConfig_ValidYAML(t *testing.T) {
	tests := []struct {
		name     string
		yamlData string
		wantErr  bool
	}{
		{
			name: "simple valid config",
			yamlData: `routes:
  - path: "/healthz"
    method: GET
    template: "OK"`,
			wantErr: false,
		},
		{
			name: "config with template file",
			yamlData: `routes:
  - path: "/user"
    method: POST
    template_file: "` + createTempFile(nil, "User template") + `"`,
			wantErr: false,
		},
		{
			name: "config with regex pattern",
			yamlData: `routes:
  - path: "/^/user/(?P<id>[0-9]+)$/"
    method: GET
    template: "User {{.Params.id}}"`,
			wantErr: false,
		},
		{
			name: "multiple routes",
			yamlData: `routes:
  - path: "/health"
    method: GET
    template: "healthy"
  - path: "/status"
    method: GET
    template: "running"`,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary file
			tmpFile := createTempFile(t, tt.yamlData)
			defer os.Remove(tmpFile)

			config, err := LoadConfig(tmpFile)
			if (err != nil) != tt.wantErr {
				t.Errorf("LoadConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if config == nil {
					t.Error("LoadConfig() returned nil config for valid input")
					return
				}
				if len(config.Routes) == 0 {
					t.Error("LoadConfig() returned config with no routes")
				}
			}
		})
	}
}

func TestLoadConfig_InvalidYAML(t *testing.T) {
	tests := []struct {
		name     string
		yamlData string
		wantErr  string
	}{
		{
			name:     "invalid YAML syntax",
			yamlData: `routes:\n  - path: "/test\n    method: GET`,
			wantErr:  "failed to parse YAML",
		},
		{
			name:     "malformed YAML",
			yamlData: `{invalid yaml structure`,
			wantErr:  "failed to parse YAML",
		},
		{
			name:     "wrong data structure",
			yamlData: `routes: "not an array"`,
			wantErr:  "failed to parse YAML",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpFile := createTempFile(t, tt.yamlData)
			defer os.Remove(tmpFile)

			config, err := LoadConfig(tmpFile)
			if err == nil {
				t.Error("LoadConfig() expected error but got none")
				return
			}

			if config != nil {
				t.Error("LoadConfig() should return nil config on error")
			}

			var loadErr *LoadError
			if !errors.As(err, &loadErr) {
				t.Errorf("LoadConfig() error should be LoadError, got %T", err)
			}

			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("LoadConfig() error = %v, want error containing %q", err, tt.wantErr)
			}
		})
	}
}

func TestLoadConfig_MissingRequiredFields(t *testing.T) {
	tests := []struct {
		name     string
		yamlData string
		wantErr  string
	}{
		{
			name: "missing path",
			yamlData: `routes:
  - method: GET
    template: "test"`,
			wantErr: "path cannot be empty",
		},
		{
			name: "missing method",
			yamlData: `routes:
  - path: "/test"
    template: "test"`,
			wantErr: "HTTP method cannot be empty",
		},
		{
			name: "missing template and template_file",
			yamlData: `routes:
  - path: "/test"
    method: GET`,
			wantErr: "either 'template' or 'template_file' must be specified",
		},
		{
			name:     "empty routes array",
			yamlData: `routes: []`,
			wantErr:  "at least one route must be defined",
		},
		{
			name:     "no routes key",
			yamlData: `other_key: value`,
			wantErr:  "at least one route must be defined",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpFile := createTempFile(t, tt.yamlData)
			defer os.Remove(tmpFile)

			config, err := LoadConfig(tmpFile)
			if err == nil {
				t.Error("LoadConfig() expected error but got none")
				return
			}

			if config != nil {
				t.Error("LoadConfig() should return nil config on error")
			}

			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("LoadConfig() error = %v, want error containing %q", err, tt.wantErr)
			}
		})
	}
}

func TestLoadConfig_InvalidFieldCombinations(t *testing.T) {
	tests := []struct {
		name     string
		yamlData string
		wantErr  string
	}{
		{
			name: "both template and template_file specified",
			yamlData: `routes:
  - path: "/test"
    method: GET
    template: "inline template"
    template_file: "file.tmpl"`,
			wantErr: "only one of 'template' or 'template_file' can be specified",
		},
		{
			name: "invalid HTTP method",
			yamlData: `routes:
  - path: "/test"
    method: INVALID
    template: "test"`,
			wantErr: "invalid HTTP method",
		},
		{
			name: "invalid regex pattern",
			yamlData: `routes:
  - path: "/[invalid regex/"
    method: GET
    template: "test"`,
			wantErr: "invalid regex pattern",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpFile := createTempFile(t, tt.yamlData)
			defer os.Remove(tmpFile)

			config, err := LoadConfig(tmpFile)
			if err == nil {
				t.Error("LoadConfig() expected error but got none")
				return
			}

			if config != nil {
				t.Error("LoadConfig() should return nil config on error")
			}

			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("LoadConfig() error = %v, want error containing %q", err, tt.wantErr)
			}
		})
	}
}

func TestLoadConfig_FileAccessErrors(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		wantErr  string
	}{
		{
			name:     "empty filename",
			filename: "",
			wantErr:  "filename cannot be empty",
		},
		{
			name:     "whitespace filename",
			filename: "   ",
			wantErr:  "filename cannot be empty",
		},
		{
			name:     "non-existent file",
			filename: "/path/to/nonexistent/file.yaml",
			wantErr:  "does not exist",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, err := LoadConfig(tt.filename)
			if err == nil {
				t.Error("LoadConfig() expected error but got none")
				return
			}

			if config != nil {
				t.Error("LoadConfig() should return nil config on error")
			}

			var loadErr *LoadError
			if !errors.As(err, &loadErr) {
				t.Errorf("LoadConfig() error should be LoadError, got %T", err)
			}

			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("LoadConfig() error = %v, want error containing %q", err, tt.wantErr)
			}
		})
	}
}

func TestLoadConfig_DirectoryInsteadOfFile(t *testing.T) {
	// Create a temporary directory
	tmpDir := t.TempDir()

	config, err := LoadConfig(tmpDir)
	if err == nil {
		t.Error("LoadConfig() expected error when given directory instead of file")
		return
	}

	if config != nil {
		t.Error("LoadConfig() should return nil config on error")
	}

	if !strings.Contains(err.Error(), "is a directory") {
		t.Errorf("LoadConfig() error = %v, want error containing 'is a directory'", err)
	}
}

func TestRouteConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		route   RouteConfig
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid route with inline template",
			route: RouteConfig{
				Path:     "/test",
				Method:   "GET",
				Template: "Hello World",
			},
			wantErr: false,
		},
		{
			name: "valid route with template file",
			route: RouteConfig{
				Path:         "/test",
				Method:       "POST",
				TemplateFile: createTempFile(nil, "test template"),
			},
			wantErr: false,
		},
		{
			name: "valid regex route",
			route: RouteConfig{
				Path:     "/^/user/(?P<id>[0-9]+)$/",
				Method:   "GET",
				Template: "User {{.Params.id}}",
			},
			wantErr: false,
		},
		{
			name: "empty path",
			route: RouteConfig{
				Path:     "",
				Method:   "GET",
				Template: "test",
			},
			wantErr: true,
			errMsg:  "path cannot be empty",
		},
		{
			name: "empty method",
			route: RouteConfig{
				Path:     "/test",
				Method:   "",
				Template: "test",
			},
			wantErr: true,
			errMsg:  "HTTP method cannot be empty",
		},
		{
			name: "invalid method",
			route: RouteConfig{
				Path:     "/test",
				Method:   "INVALID",
				Template: "test",
			},
			wantErr: true,
			errMsg:  "invalid HTTP method",
		},
		{
			name: "no template source",
			route: RouteConfig{
				Path:   "/test",
				Method: "GET",
			},
			wantErr: true,
			errMsg:  "either 'template' or 'template_file' must be specified",
		},
		{
			name: "both template sources",
			route: RouteConfig{
				Path:         "/test",
				Method:       "GET",
				Template:     "inline",
				TemplateFile: "file.tmpl",
			},
			wantErr: true,
			errMsg:  "only one of 'template' or 'template_file' can be specified",
		},
		{
			name: "invalid regex pattern",
			route: RouteConfig{
				Path:     "/[invalid/",
				Method:   "GET",
				Template: "test",
			},
			wantErr: true,
			errMsg:  "invalid regex pattern",
		},
		{
			name: "non-existent template file",
			route: RouteConfig{
				Path:         "/test",
				Method:       "GET",
				TemplateFile: "/nonexistent/file.tmpl",
			},
			wantErr: true,
			errMsg:  "template file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.route.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("RouteConfig.Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("RouteConfig.Validate() error = %v, want error containing %q", err, tt.errMsg)
			}
		})
	}
}

func TestRouteConfig_IsRegexPattern(t *testing.T) {
	tests := []struct {
		name string
		path string
		want bool
	}{
		{
			name: "regex pattern with named groups",
			path: "/^/user/(?P<id>[0-9]+)$/",
			want: true,
		},
		{
			name: "simple regex pattern",
			path: "/^/test$/",
			want: true,
		},
		{
			name: "literal path",
			path: "/user/123",
			want: false,
		},
		{
			name: "path starting with slash but not ending",
			path: "/user/test",
			want: false,
		},
		{
			name: "path ending with slash but not starting",
			path: "user/test/",
			want: false,
		},
		{
			name: "just slashes",
			path: "/",
			want: false,
		},
		{
			name: "empty path",
			path: "",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			route := RouteConfig{Path: tt.path}
			if got := route.IsRegexPattern(); got != tt.want {
				t.Errorf("RouteConfig.IsRegexPattern() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRouteConfig_GetRegexPattern(t *testing.T) {
	tests := []struct {
		name string
		path string
		want string
	}{
		{
			name: "regex pattern with slashes",
			path: "/^/user/(?P<id>[0-9]+)$/",
			want: "^/user/(?P<id>[0-9]+)$",
		},
		{
			name: "literal path",
			path: "/user/123",
			want: "/user/123",
		},
		{
			name: "empty path",
			path: "",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			route := RouteConfig{Path: tt.path}
			if got := route.GetRegexPattern(); got != tt.want {
				t.Errorf("RouteConfig.GetRegexPattern() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRouteConfig_GetNormalizedMethod(t *testing.T) {
	tests := []struct {
		name   string
		method string
		want   string
	}{
		{
			name:   "lowercase method",
			method: "get",
			want:   "GET",
		},
		{
			name:   "uppercase method",
			method: "GET",
			want:   "GET",
		},
		{
			name:   "mixed case method",
			method: "PoSt",
			want:   "POST",
		},
		{
			name:   "method with spaces",
			method: "  PUT  ",
			want:   "PUT",
		},
		{
			name:   "empty method",
			method: "",
			want:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			route := RouteConfig{Method: tt.method}
			if got := route.GetNormalizedMethod(); got != tt.want {
				t.Errorf("RouteConfig.GetNormalizedMethod() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Helper function to create temporary files for testing
func createTempFile(t *testing.T, content string) string {
	if t != nil {
		tmpFile, err := os.CreateTemp("", "config_test_*.yaml")
		if err != nil {
			t.Fatalf("Failed to create temp file: %v", err)
		}
		defer tmpFile.Close()

		if _, err := tmpFile.WriteString(content); err != nil {
			t.Fatalf("Failed to write to temp file: %v", err)
		}

		return tmpFile.Name()
	}

	// For non-test cases, create a real temp file
	tmpFile, err := os.CreateTemp("", "config_*.yaml")
	if err != nil {
		return ""
	}
	defer tmpFile.Close()

	tmpFile.WriteString(content)
	return tmpFile.Name()
}

// Test validation errors implement error interface properly
func TestValidationError_Interface(t *testing.T) {
	err := &ValidationError{
		Field:   "test_field",
		Message: "test message",
	}

	// Test Error() method
	expected := `validation error in field "test_field": test message`
	if got := err.Error(); got != expected {
		t.Errorf("ValidationError.Error() = %v, want %v", got, expected)
	}

	// Test Unwrap() method
	if unwrapped := err.Unwrap(); unwrapped != nil {
		t.Errorf("ValidationError.Unwrap() = %v, want nil", unwrapped)
	}
}

// Test load errors implement error interface properly
func TestLoadError_Interface(t *testing.T) {
	cause := errors.New("underlying error")
	err := &LoadError{
		Filename: "test.yaml",
		Cause:    cause,
	}

	// Test Error() method
	expected := `failed to load config from "test.yaml": underlying error`
	if got := err.Error(); got != expected {
		t.Errorf("LoadError.Error() = %v, want %v", got, expected)
	}

	// Test Unwrap() method
	if unwrapped := err.Unwrap(); unwrapped != cause {
		t.Errorf("LoadError.Unwrap() = %v, want %v", unwrapped, cause)
	}
}

func TestRouteConfig_ValidateMatchHeaders(t *testing.T) {
	tests := []struct {
		name         string
		matchHeaders map[string]string
		wantErr      bool
		errContains  string
	}{
		{
			name:         "no headers - valid",
			matchHeaders: nil,
			wantErr:      false,
		},
		{
			name:         "empty headers - valid",
			matchHeaders: map[string]string{},
			wantErr:      false,
		},
		{
			name: "literal header match - valid",
			matchHeaders: map[string]string{
				"Content-Type": "application/json",
				"X-API-Key":    "secret123",
			},
			wantErr: false,
		},
		{
			name: "regex header match - valid",
			matchHeaders: map[string]string{
				"Authorization": "/Bearer .+/",
				"User-Agent":    "/Mozilla.*/",
			},
			wantErr: false,
		},
		{
			name: "mixed literal and regex - valid",
			matchHeaders: map[string]string{
				"Content-Type":  "application/json",
				"Authorization": "/Bearer .+/",
			},
			wantErr: false,
		},
		{
			name: "empty header name - invalid",
			matchHeaders: map[string]string{
				"": "some-value",
			},
			wantErr:     true,
			errContains: "header name cannot be empty",
		},
		{
			name: "whitespace header name - invalid",
			matchHeaders: map[string]string{
				"   ": "some-value",
			},
			wantErr:     true,
			errContains: "header name cannot be empty",
		},
		{
			name: "invalid character in header name - invalid",
			matchHeaders: map[string]string{
				"Content@Type": "application/json",
			},
			wantErr:     true,
			errContains: "invalid character",
		},
		{
			name: "invalid regex pattern - invalid",
			matchHeaders: map[string]string{
				"Authorization": "/[unclosed/",
			},
			wantErr:     true,
			errContains: "invalid regex pattern",
		},
		{
			name: "valid regex with special characters",
			matchHeaders: map[string]string{
				"X-Request-ID": "/^[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}$/",
			},
			wantErr: false,
		},
		{
			name: "regex without slashes - treated as literal",
			matchHeaders: map[string]string{
				"Authorization": "Bearer token123",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			route := &RouteConfig{
				Path:         "/test",
				Method:       "GET",
				Template:     "test template",
				MatchHeaders: tt.matchHeaders,
			}

			err := route.Validate()
			hasErr := err != nil

			if hasErr != tt.wantErr {
				t.Errorf("RouteConfig.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.wantErr && tt.errContains != "" {
				if err == nil || !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("RouteConfig.Validate() error = %v, want error containing %q", err, tt.errContains)
				}
			}
		})
	}
}

func TestRouteConfig_ValidateResponseHeaders(t *testing.T) {
	tests := []struct {
		name            string
		responseHeaders map[string]string
		wantErr         bool
		errContains     string
	}{
		{
			name:            "no response headers - valid",
			responseHeaders: nil,
			wantErr:         false,
		},
		{
			name:            "empty response headers - valid",
			responseHeaders: map[string]string{},
			wantErr:         false,
		},
		{
			name: "simple literal headers - valid",
			responseHeaders: map[string]string{
				"Content-Type":  "application/json",
				"X-API-Version": "v1",
			},
			wantErr: false,
		},
		{
			name: "template headers - valid",
			responseHeaders: map[string]string{
				"X-Request-ID": "{{ index .Headers \"X-Request-ID\" }}",
				"X-User-Agent": "{{ .Request.Header.Get \"User-Agent\" }}",
				"Content-Type": "{{ if eq .Request.Method \"POST\" }}application/json{{ else }}text/html{{ end }}",
			},
			wantErr: false,
		},
		{
			name: "complex template with functions - valid",
			responseHeaders: map[string]string{
				"X-Custom": "{{ .Params.name | upper }}",
				"X-Query":  "{{ query \"debug\" .Request }}",
				"X-Header": "{{ header \"Authorization\" .Request }}",
			},
			wantErr: false,
		},
		{
			name: "mixed literal and template headers - valid",
			responseHeaders: map[string]string{
				"Content-Type":  "application/json",
				"X-Request-ID":  "{{ index .Headers \"X-Request-ID\" }}",
				"Cache-Control": "no-cache",
			},
			wantErr: false,
		},
		{
			name: "empty header name - invalid",
			responseHeaders: map[string]string{
				"": "some-value",
			},
			wantErr:     true,
			errContains: "header name cannot be empty",
		},
		{
			name: "whitespace header name - invalid",
			responseHeaders: map[string]string{
				"   ": "some-value",
			},
			wantErr:     true,
			errContains: "header name cannot be empty",
		},
		{
			name: "invalid character in header name - invalid",
			responseHeaders: map[string]string{
				"Content@Type": "application/json",
			},
			wantErr:     true,
			errContains: "invalid character",
		},
		{
			name: "invalid template syntax - unclosed action",
			responseHeaders: map[string]string{
				"X-Custom": "{{ .Headers.Test",
			},
			wantErr:     true,
			errContains: "invalid template syntax",
		},
		{
			name: "invalid template syntax - undefined function (allowed in validation)",
			responseHeaders: map[string]string{
				"X-Custom": "{{ undefinedFunc }}",
			},
			wantErr: false, // We allow this in validation, actual error will occur during compilation
		},
		{
			name: "invalid template syntax - malformed control structure (allowed in validation)",
			responseHeaders: map[string]string{
				"X-Custom": "{{ if .Test }}unclosed if",
			},
			wantErr: false, // We allow this in validation, actual error will occur during compilation
		},
		{
			name: "valid template with sprig functions",
			responseHeaders: map[string]string{
				"X-UUID": "{{ uuidv4 }}",
				"X-Time": "{{ now | date \"2006-01-02\" }}",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			route := &RouteConfig{
				Path:            "/test",
				Method:          "GET",
				Template:        "test template",
				ResponseHeaders: tt.responseHeaders,
			}

			err := route.Validate()
			hasErr := err != nil

			if hasErr != tt.wantErr {
				t.Errorf("RouteConfig.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.wantErr && tt.errContains != "" {
				if err == nil || !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("RouteConfig.Validate() error = %v, want error containing %q", err, tt.errContains)
				}
			}
		})
	}
}

func TestSanitizeTemplateNameForValidation(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple alphanumeric",
			input:    "hello123",
			expected: "hello123",
		},
		{
			name:     "with underscores",
			input:    "hello_world_123",
			expected: "hello_world_123",
		},
		{
			name:     "path with slashes",
			input:    "/api/v1/users",
			expected: "api_v1_users",
		},
		{
			name:     "regex pattern",
			input:    "/user/(?P<id>\\d+)",
			expected: "user_P_id_d",
		},
		{
			name:     "special characters",
			input:    "route-with.dots?and*wildcards",
			expected: "route_with_dots_and_wildcards",
		},
		{
			name:     "parentheses and brackets",
			input:    "func(param)[index]{key}",
			expected: "func_param_index_key",
		},
		{
			name:     "spaces and tabs",
			input:    "hello world\twith\tspaces",
			expected: "hello_world_with_spaces",
		},
		{
			name:     "consecutive special chars",
			input:    "multiple!!!special???chars",
			expected: "multiple_special_chars",
		},
		{
			name:     "leading and trailing special chars",
			input:    "___hello___world___",
			expected: "hello___world",
		},
		{
			name:     "only special characters",
			input:    "!@#$%^&*()",
			expected: "unnamed",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "unnamed",
		},
		{
			name:     "only underscores",
			input:    "____",
			expected: "unnamed",
		},
		{
			name:     "mixed case with numbers",
			input:    "MyRoute123_API",
			expected: "MyRoute123_API",
		},
		{
			name:     "URL with query params",
			input:    "/api/users?id=123&name=john",
			expected: "api_users_id_123_name_john",
		},
		{
			name:     "HTTP method style",
			input:    "GET_/api/v1/users/{id}",
			expected: "GET__api_v1_users_id",
		},
		{
			name:     "unicode characters",
			input:    "café_résumé_naïve",
			expected: "caf__r_sum__na_ve",
		},
		{
			name:     "long consecutive separators",
			input:    "route---with___many...separators",
			expected: "route_with___many_separators",
		},
		{
			name:     "colon separated",
			input:    "namespace:service:method",
			expected: "namespace_service_method",
		},
		{
			name:     "single character",
			input:    "a",
			expected: "a",
		},
		{
			name:     "single special character",
			input:    "@",
			expected: "unnamed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeTemplateNameForValidation(tt.input)
			if result != tt.expected {
				t.Errorf("sanitizeTemplateNameForValidation(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestDelimiterConfig_GetWithDefaults(t *testing.T) {
	tests := []struct {
		name      string
		config    DelimiterConfig
		wantLeft  string
		wantRight string
	}{
		{
			name:      "empty config uses defaults",
			config:    DelimiterConfig{},
			wantLeft:  "{{",
			wantRight: "}}",
		},
		{
			name: "custom delimiters preserved",
			config: DelimiterConfig{
				Left:  "<%",
				Right: "%>",
			},
			wantLeft:  "<%",
			wantRight: "%>",
		},
		{
			name: "partial config uses defaults for missing",
			config: DelimiterConfig{
				Left: "<%%",
			},
			wantLeft:  "<%%",
			wantRight: "}}",
		},
		{
			name: "only right delimiter set",
			config: DelimiterConfig{
				Right: "%%>",
			},
			wantLeft:  "{{",
			wantRight: "%%>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.config.GetWithDefaults()
			if result.Left != tt.wantLeft {
				t.Errorf("GetWithDefaults().Left = %q, want %q", result.Left, tt.wantLeft)
			}
			if result.Right != tt.wantRight {
				t.Errorf("GetWithDefaults().Right = %q, want %q", result.Right, tt.wantRight)
			}
		})
	}
}

func TestDelimiterConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  DelimiterConfig
		wantErr bool
		errMsg  string
	}{
		{
			name:    "empty config is valid",
			config:  DelimiterConfig{},
			wantErr: false,
		},
		{
			name: "valid custom delimiters",
			config: DelimiterConfig{
				Left:  "<%",
				Right: "%>",
			},
			wantErr: false,
		},
		{
			name: "same left and right delimiters",
			config: DelimiterConfig{
				Left:  "%%",
				Right: "%%",
			},
			wantErr: true,
			errMsg:  "left and right delimiters cannot be the same",
		},
		{
			name: "empty left delimiter when right is set",
			config: DelimiterConfig{
				Left:  "",
				Right: "%>",
			},
			wantErr: true,
			errMsg:  "delimiter cannot be empty if specified",
		},
		{
			name: "empty right delimiter when left is set",
			config: DelimiterConfig{
				Left:  "<%",
				Right: "",
			},
			wantErr: true,
			errMsg:  "delimiter cannot be empty if specified",
		},
		{
			name: "delimiter with newline",
			config: DelimiterConfig{
				Left:  "<%\n",
				Right: "%>",
			},
			wantErr: true,
			errMsg:  "delimiter cannot contain newline, carriage return, or tab characters",
		},
		{
			name: "delimiter with tab",
			config: DelimiterConfig{
				Left:  "<%",
				Right: "%>\t",
			},
			wantErr: true,
			errMsg:  "delimiter cannot contain newline, carriage return, or tab characters",
		},
		{
			name: "delimiter too long",
			config: DelimiterConfig{
				Left:  "<%",
				Right: "verylongdelimiter",
			},
			wantErr: true,
			errMsg:  "delimiter cannot be longer than 10 characters",
		},
		{
			name: "valid bracket style",
			config: DelimiterConfig{
				Left:  "[[[",
				Right: "]]]",
			},
			wantErr: false,
		},
		{
			name: "valid double character style",
			config: DelimiterConfig{
				Left:  "<<",
				Right: ">>",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("DelimiterConfig.Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("DelimiterConfig.Validate() error = %v, want error containing %q", err, tt.errMsg)
			}
		})
	}
}

func TestTemplateConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  TemplateConfig
		wantErr bool
	}{
		{
			name:    "empty template config is valid",
			config:  TemplateConfig{},
			wantErr: false,
		},
		{
			name: "valid delimiter config",
			config: TemplateConfig{
				Delimiters: DelimiterConfig{
					Left:  "<%",
					Right: "%>",
				},
			},
			wantErr: false,
		},
		{
			name: "invalid delimiter config propagates error",
			config: TemplateConfig{
				Delimiters: DelimiterConfig{
					Left:  "%%",
					Right: "%%",
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("TemplateConfig.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestConfig_ValidateWithCustomDelimiters(t *testing.T) {
	tests := []struct {
		name     string
		yamlData string
		wantErr  bool
		errMsg   string
	}{
		{
			name: "valid config with custom delimiters",
			yamlData: `
template:
  delimiters:
    left: "<%"
    right: "%>"
routes:
  - path: "/test"
    method: GET
    template: "Hello <% .Name %>!"`,
			wantErr: false,
		},
		{
			name: "invalid config with same delimiters",
			yamlData: `
template:
  delimiters:
    left: "%"
    right: "%"
routes:
  - path: "/test"
    method: GET
    template: "Hello % .Name %!"`,
			wantErr: true,
			errMsg:  "template configuration",
		},
		{
			name: "config validates template syntax with custom delimiters",
			yamlData: `
template:
  delimiters:
    left: "<%"
    right: "%>"
routes:
  - path: "/test"
    method: GET
    template: "Hello <% .Name %>! Invalid: {{ .Other }}"`,
			wantErr: false, // {{ .Other }} should be treated as literal text, not template
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempFile := createTempFile(nil, tt.yamlData)
			defer os.Remove(tempFile)

			_, err := LoadConfig(tempFile)
			if (err != nil) != tt.wantErr {
				t.Errorf("LoadConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("LoadConfig() error = %v, want error containing %q", err, tt.errMsg)
			}
		})
	}
}
