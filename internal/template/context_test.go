package template

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

func TestNewTemplateContext_Basic(t *testing.T) {
	// Create a basic request
	req, err := http.NewRequest("GET", "/test?debug=true&name=world", strings.NewReader(`{"message":"hello"}`))
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", "12345")
	req.Header.Set("Authorization", "Bearer token123")

	params := map[string]string{
		"id":   "123",
		"name": "test",
	}

	ctx, err := NewTemplateContext(req, params)
	if err != nil {
		t.Errorf("NewTemplateContext() error = %v, expected no error", err)
		return
	}

	if ctx == nil {
		t.Error("NewTemplateContext() should return non-nil context")
		return
	}

	// Verify request is set
	if ctx.Request != req {
		t.Error("NewTemplateContext() context Request should match input request")
	}

	// Verify headers are extracted
	if ctx.Headers.Get("Content-Type") != "application/json" {
		t.Errorf("NewTemplateContext() context Headers.Get(Content-Type) = %v, want application/json", ctx.Headers.Get("Content-Type"))
	}
	if ctx.Headers.Get("X-User-Id") != "12345" {
		t.Errorf("NewTemplateContext() context Headers.Get(X-User-Id) = %q, want 12345", ctx.Headers.Get("X-User-Id"))
	}

	// Verify query params are extracted
	if ctx.Query.Get("debug") != "true" {
		t.Errorf("NewTemplateContext() context Query.Get(debug) = %v, want true", ctx.Query.Get("debug"))
	}
	if ctx.Query.Get("name") != "world" {
		t.Errorf("NewTemplateContext() context Query.Get(name) = %v, want world", ctx.Query.Get("name"))
	}

	// Verify params are set
	if ctx.Params["id"] != "123" {
		t.Errorf("NewTemplateContext() context Params[id] = %v, want 123", ctx.Params["id"])
	}
	if ctx.Params["name"] != "test" {
		t.Errorf("NewTemplateContext() context Params[name] = %v, want test", ctx.Params["name"])
	}

	// Verify body is parsed as JSON
	jsonBody, ok := ctx.Body.(map[string]interface{})
	if !ok {
		t.Errorf("NewTemplateContext() context Body should be map[string]interface{}, got %T", ctx.Body)
	} else {
		if message, exists := jsonBody["message"]; !exists || message != "hello" {
			t.Errorf("NewTemplateContext() context Body[message] = %v, want hello", message)
		}
	}
}

func TestParseRequestBody_JSON(t *testing.T) {
	tests := []struct {
		name        string
		body        string
		contentType string
		wantErr     bool
		wantType    string
	}{
		{
			name:        "valid JSON object",
			body:        `{"name":"test","value":123}`,
			contentType: "application/json",
			wantErr:     false,
			wantType:    "map[string]interface{}",
		},
		{
			name:        "valid JSON array",
			body:        `[1,2,3]`,
			contentType: "application/json",
			wantErr:     false,
			wantType:    "[]interface{}",
		},
		{
			name:        "valid JSON with charset",
			body:        `{"message":"hello"}`,
			contentType: "application/json; charset=utf-8",
			wantErr:     false,
			wantType:    "map[string]interface{}",
		},
		{
			name:        "text/json content type",
			body:        `{"test":true}`,
			contentType: "text/json",
			wantErr:     false,
			wantType:    "map[string]interface{}",
		},
		{
			name:        "custom JSON content type",
			body:        `{"custom":true}`,
			contentType: "application/vnd.api+json",
			wantErr:     false,
			wantType:    "map[string]interface{}",
		},
		{
			name:        "invalid JSON with JSON content type",
			body:        `{invalid json}`,
			contentType: "application/json",
			wantErr:     false,
			wantType:    "map[string]interface{}",
		},
		{
			name:        "empty body with JSON content type",
			body:        "",
			contentType: "application/json",
			wantErr:     false,
			wantType:    "<nil>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest("POST", "/test", strings.NewReader(tt.body))
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}
			req.Header.Set("Content-Type", tt.contentType)

			result, err := parseRequestBody(req)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseRequestBody() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// Check the type of the result
			var actualType string
			if result == nil {
				actualType = "<nil>"
			} else {
				switch res := result.(type) {
				case map[string]interface{}:
					actualType = "map[string]interface{}"
					// For invalid JSON, check if it contains parse_error
					if _, hasError := res["parse_error"]; hasError && tt.body == `{invalid json}` {
						// This is expected for invalid JSON
					} else if tt.body == `{"name":"test","value":123}` {
						if name, exists := res["name"]; !exists || name != "test" {
							t.Errorf("parseRequestBody() parsed JSON name = %v, want test", name)
						}
					}
				case []interface{}:
					actualType = "[]interface{}"
				case string:
					actualType = "string"
				default:
					actualType = "unknown"
				}
			}

			if actualType != tt.wantType {
				t.Errorf("parseRequestBody() result type = %v, want %v", actualType, tt.wantType)
			}
		})
	}
}

func TestParseRequestBody_NonJSON(t *testing.T) {
	tests := []struct {
		name        string
		body        string
		contentType string
		expected    string
	}{
		{
			name:        "plain text",
			body:        "Hello World",
			contentType: "text/plain",
			expected:    "Hello World",
		},
		{
			name:        "HTML content",
			body:        "<html><body>Test</body></html>",
			contentType: "text/html",
			expected:    "<html><body>Test</body></html>",
		},
		{
			name:        "form data",
			body:        "name=test&value=123",
			contentType: "application/x-www-form-urlencoded",
			expected:    "name=test&value=123",
		},
		{
			name:        "no content type",
			body:        "some data",
			contentType: "",
			expected:    "some data",
		},
		{
			name:        "XML content",
			body:        "<root><item>test</item></root>",
			contentType: "application/xml",
			expected:    "<root><item>test</item></root>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest("POST", "/test", strings.NewReader(tt.body))
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}
			if tt.contentType != "" {
				req.Header.Set("Content-Type", tt.contentType)
			}

			result, err := parseRequestBody(req)
			if err != nil {
				t.Errorf("parseRequestBody() error = %v, expected no error", err)
				return
			}

			if resultStr, ok := result.(string); !ok || resultStr != tt.expected {
				t.Errorf("parseRequestBody() result = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestParseRequestBody_EdgeCases(t *testing.T) {
	tests := []struct {
		name       string
		setupReq   func() *http.Request
		wantErr    bool
		wantResult interface{}
	}{
		{
			name: "nil body",
			setupReq: func() *http.Request {
				req, _ := http.NewRequest("GET", "/test", nil)
				return req
			},
			wantErr:    false,
			wantResult: nil,
		},
		{
			name: "empty body",
			setupReq: func() *http.Request {
				req, _ := http.NewRequest("POST", "/test", strings.NewReader(""))
				return req
			},
			wantErr:    false,
			wantResult: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := tt.setupReq()

			result, err := parseRequestBody(req)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseRequestBody() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if result != tt.wantResult {
				t.Errorf("parseRequestBody() result = %v, want %v", result, tt.wantResult)
			}
		})
	}
}

func TestIsJSONContentType(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		want        bool
	}{
		{
			name:        "application/json",
			contentType: "application/json",
			want:        true,
		},
		{
			name:        "application/json with charset",
			contentType: "application/json; charset=utf-8",
			want:        true,
		},
		{
			name:        "text/json",
			contentType: "text/json",
			want:        true,
		},
		{
			name:        "custom json type",
			contentType: "application/vnd.api+json",
			want:        true,
		},
		{
			name:        "application/hal+json",
			contentType: "application/hal+json",
			want:        true,
		},
		{
			name:        "uppercase JSON",
			contentType: "APPLICATION/JSON",
			want:        true,
		},
		{
			name:        "mixed case JSON",
			contentType: "Application/Json",
			want:        true,
		},
		{
			name:        "text/plain",
			contentType: "text/plain",
			want:        false,
		},
		{
			name:        "application/xml",
			contentType: "application/xml",
			want:        false,
		},
		{
			name:        "text/html",
			contentType: "text/html",
			want:        false,
		},
		{
			name:        "empty content type",
			contentType: "",
			want:        false,
		},
		{
			name:        "whitespace content type",
			contentType: "   ",
			want:        false,
		},
		{
			name:        "partial match not json",
			contentType: "jsonp/application",
			want:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isJSONContentType(tt.contentType); got != tt.want {
				t.Errorf("isJSONContentType() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewTemplateContext_ErrorHandling(t *testing.T) {
	tests := []struct {
		name     string
		setupReq func() (*http.Request, map[string]string)
		wantErr  bool
	}{
		{
			name: "valid request",
			setupReq: func() (*http.Request, map[string]string) {
				req, _ := http.NewRequest("GET", "/test", nil)
				params := map[string]string{"id": "123"}
				return req, params
			},
			wantErr: false,
		},
		{
			name: "nil params",
			setupReq: func() (*http.Request, map[string]string) {
				req, _ := http.NewRequest("GET", "/test", nil)
				return req, nil
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, params := tt.setupReq()

			ctx, err := NewTemplateContext(req, params)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewTemplateContext() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && ctx == nil {
				t.Error("NewTemplateContext() should return non-nil context for valid input")
			}
		})
	}
}

func TestNewTemplateContext_WithComplexJSON(t *testing.T) {
	jsonData := map[string]interface{}{
		"user": map[string]interface{}{
			"id":   123,
			"name": "John Doe",
			"tags": []string{"admin", "developer"},
		},
		"metadata": map[string]interface{}{
			"version":   "1.0",
			"timestamp": "2023-01-01T00:00:00Z",
		},
	}

	jsonBytes, err := json.Marshal(jsonData)
	if err != nil {
		t.Fatalf("Failed to marshal JSON: %v", err)
	}

	req, err := http.NewRequest("POST", "/api/users", strings.NewReader(string(jsonBytes)))
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	params := map[string]string{
		"version": "v1",
		"action":  "create",
	}

	ctx, err := NewTemplateContext(req, params)
	if err != nil {
		t.Errorf("NewTemplateContext() error = %v, expected no error", err)
		return
	}

	// Verify complex JSON body parsing
	bodyMap, ok := ctx.Body.(map[string]interface{})
	if !ok {
		t.Errorf("NewTemplateContext() body should be map[string]interface{}, got %T", ctx.Body)
		return
	}

	// Check nested user object
	user, exists := bodyMap["user"]
	if !exists {
		t.Error("NewTemplateContext() body should contain 'user' field")
		return
	}

	userMap, ok := user.(map[string]interface{})
	if !ok {
		t.Errorf("NewTemplateContext() body.user should be map[string]interface{}, got %T", user)
		return
	}

	if userMap["name"] != "John Doe" {
		t.Errorf("NewTemplateContext() body.user.name = %v, want 'John Doe'", userMap["name"])
	}

	// Check array field
	tags, exists := userMap["tags"]
	if !exists {
		t.Error("NewTemplateContext() body.user should contain 'tags' field")
		return
	}

	tagsArray, ok := tags.([]interface{})
	if !ok {
		t.Errorf("NewTemplateContext() body.user.tags should be []interface{}, got %T", tags)
		return
	}

	if len(tagsArray) != 2 || tagsArray[0] != "admin" || tagsArray[1] != "developer" {
		t.Errorf("NewTemplateContext() body.user.tags = %v, want ['admin', 'developer']", tagsArray)
	}
}

// Performance benchmarks
func BenchmarkNewTemplateContext(b *testing.B) {
	req, err := http.NewRequest("POST", "/api/users?debug=true&format=json", strings.NewReader(`{"name":"test","id":123}`))
	if err != nil {
		b.Fatalf("Failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer token123")
	req.Header.Set("X-User-ID", "456")

	params := map[string]string{
		"version": "v1",
		"action":  "create",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := NewTemplateContext(req, params)
		if err != nil {
			b.Fatalf("NewTemplateContext() error = %v", err)
		}
	}
}

func BenchmarkParseRequestBody_JSON(b *testing.B) {
	jsonData := `{"user":{"id":123,"name":"John Doe","tags":["admin","developer"]},"metadata":{"version":"1.0","timestamp":"2023-01-01T00:00:00Z"}}`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req, _ := http.NewRequest("POST", "/test", strings.NewReader(jsonData))
		req.Header.Set("Content-Type", "application/json")
		_, err := parseRequestBody(req)
		if err != nil {
			b.Fatalf("parseRequestBody() error = %v", err)
		}
	}
}

func BenchmarkParseRequestBody_Text(b *testing.B) {
	textData := "This is some plain text data that will be parsed as a string"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req, _ := http.NewRequest("POST", "/test", strings.NewReader(textData))
		req.Header.Set("Content-Type", "text/plain")
		_, err := parseRequestBody(req)
		if err != nil {
			b.Fatalf("parseRequestBody() error = %v", err)
		}
	}
}
