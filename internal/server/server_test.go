package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/patrickdappollonio/mockingjay/internal/config"
)

// TestServer represents a test server instance with utilities for integration testing
type TestServer struct {
	*Server
	HTTPServer *httptest.Server
	BaseURL    string
	Client     *http.Client
	Logger     *slog.Logger
}

// NewTestServer creates a new test server instance for integration testing
func NewTestServer(t *testing.T, cfg *config.Config) *TestServer {
	t.Helper()

	// Create a logger that discards output during tests (unless verbose)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	if testing.Verbose() {
		logger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	}

	// Create server instance
	server, err := NewServer(cfg, "test-config.yaml", ":0", logger, "test-version")
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	// Create test HTTP server
	httpServer := httptest.NewServer(server)

	// Create HTTP client with reasonable timeout
	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	testServer := &TestServer{
		Server:     server,
		HTTPServer: httpServer,
		BaseURL:    httpServer.URL,
		Client:     client,
		Logger:     logger,
	}

	// Register cleanup function
	t.Cleanup(func() {
		testServer.Close()
	})

	return testServer
}

// Close shuts down the test server
func (ts *TestServer) Close() {
	if ts.HTTPServer != nil {
		ts.HTTPServer.Close()
	}
}

// makeRequest is a helper method to make HTTP requests to the test server
func (ts *TestServer) makeRequest(method, path string, body io.Reader, headers map[string]string) (*http.Response, error) {
	url := ts.BaseURL + path
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	return ts.Client.Do(req)
}

// readResponseBody reads and returns the response body as a string
func readResponseBody(t *testing.T, resp *http.Response) string {
	t.Helper()
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}

	return string(body)
}

// createTestConfig creates a test configuration with the given routes
func createTestConfig(routes []config.RouteConfig) *config.Config {
	return &config.Config{
		Routes: routes,
	}
}

// Integration Tests Start Here

func TestServer_Integration_SimpleStaticRoute(t *testing.T) {
	// Test simple static route responses
	cfg := createTestConfig([]config.RouteConfig{
		{
			Path:     "/healthz",
			Method:   "GET",
			Template: "OK - Server is healthy",
		},
	})

	ts := NewTestServer(t, cfg)

	// Test successful request
	resp, err := ts.makeRequest("GET", "/healthz", nil, nil)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	body := readResponseBody(t, resp)
	expected := "OK - Server is healthy"
	if body != expected {
		t.Errorf("Expected body %q, got %q", expected, body)
	}

	// Test wrong method
	resp, err = ts.makeRequest("POST", "/healthz", nil, nil)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("Expected status 404 for wrong method, got %d", resp.StatusCode)
	}
}

func TestServer_Integration_DynamicRoutesWithParameters(t *testing.T) {
	// Test dynamic routes with regex parameters
	cfg := createTestConfig([]config.RouteConfig{
		{
			Path:   "/^/user/(?P<name>[^/]+)$/",
			Method: "GET",
			Template: `Hello, {{ .Params.name }}!
User-Agent: {{ index .Headers "User-Agent" }}`,
		},
	})

	ts := NewTestServer(t, cfg)

	// Test successful request with parameter extraction
	headers := map[string]string{"User-Agent": "test-client/1.0"}
	resp, err := ts.makeRequest("GET", "/user/alice", nil, headers)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	body := readResponseBody(t, resp)
	if !strings.Contains(body, "Hello, alice!") {
		t.Errorf("Expected body to contain 'Hello, alice!', got %q", body)
	}
	if !strings.Contains(body, "User-Agent: [test-client/1.0]") {
		t.Errorf("Expected body to contain User-Agent header, got %q", body)
	}

	// Test non-matching path
	resp, err = ts.makeRequest("GET", "/user/alice/extra", nil, nil)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("Expected status 404 for non-matching path, got %d", resp.StatusCode)
	}
}

func TestServer_Integration_JSONEchoEndpoint(t *testing.T) {
	// Test JSON echo endpoints with body parsing
	cfg := createTestConfig([]config.RouteConfig{
		{
			Path:     "/api/echo",
			Method:   "POST",
			Template: `Received JSON: {{ .Body }}`,
		},
	})

	ts := NewTestServer(t, cfg)

	// Test with valid JSON
	jsonData := map[string]interface{}{
		"message": "hello world",
		"count":   42,
	}
	jsonBytes, _ := json.Marshal(jsonData)

	headers := map[string]string{
		"Content-Type": "application/json",
	}

	resp, err := ts.makeRequest("POST", "/api/echo", bytes.NewReader(jsonBytes), headers)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	body := readResponseBody(t, resp)
	if !strings.Contains(body, "hello world") {
		t.Errorf("Expected body to contain JSON data, got %q", body)
	}
	if !strings.Contains(body, "42") {
		t.Errorf("Expected body to contain JSON number, got %q", body)
	}
}

func TestServer_Integration_TemplateRenderingWithContext(t *testing.T) {
	// Test template rendering with all context data
	cfg := createTestConfig([]config.RouteConfig{
		{
			Path:   "/context-test",
			Method: "GET",
			Template: `Request Method: {{ .Request.Method }}
Path: {{ .Request.URL.Path }}
Query param 'debug': {{ .Query.debug }}
Header 'X-Custom': {{ index .Headers "X-Custom" }}
{{ if .Params.name }}Param 'name': {{ .Params.name }}{{ end }}`,
		},
	})

	ts := NewTestServer(t, cfg)

	// Make request with query parameters and headers
	headers := map[string]string{"X-Custom": "test-value"}
	resp, err := ts.makeRequest("GET", "/context-test?debug=true&other=ignored", nil, headers)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	body := readResponseBody(t, resp)
	expectedContains := []string{
		"Request Method: GET",
		"Path: /context-test",
		"Query param 'debug': [true]",
		"Header 'X-Custom': [test-value]",
	}

	for _, expected := range expectedContains {
		if !strings.Contains(body, expected) {
			t.Errorf("Expected body to contain %q, got %q", expected, body)
		}
	}
}

func TestServer_Integration_HeaderMatching(t *testing.T) {
	// Test header matching functionality
	cfg := createTestConfig([]config.RouteConfig{
		{
			Path:   "/api/secure",
			Method: "POST",
			MatchHeaders: map[string]string{
				"Authorization": "/^Bearer .+$/",
				"Content-Type":  "application/json",
			},
			Template: "Access granted",
		},
	})

	ts := NewTestServer(t, cfg)

	// Test successful header matching
	headers := map[string]string{
		"Authorization": "Bearer abc123",
		"Content-Type":  "application/json",
	}

	resp, err := ts.makeRequest("POST", "/api/secure", nil, headers)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200 with matching headers, got %d", resp.StatusCode)
	}

	body := readResponseBody(t, resp)
	if body != "Access granted" {
		t.Errorf("Expected 'Access granted', got %q", body)
	}

	// Test missing required header
	badHeaders := map[string]string{
		"Content-Type": "application/json",
		// Missing Authorization header
	}

	resp, err = ts.makeRequest("POST", "/api/secure", nil, badHeaders)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("Expected status 404 with missing header, got %d", resp.StatusCode)
	}

	// Test non-matching header value
	badHeaders = map[string]string{
		"Authorization": "Basic dXNlcjpwYXNz", // Basic auth instead of Bearer
		"Content-Type":  "application/json",
	}

	resp, err = ts.makeRequest("POST", "/api/secure", nil, badHeaders)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("Expected status 404 with non-matching header, got %d", resp.StatusCode)
	}
}

func TestServer_Integration_CustomResponseHeaders(t *testing.T) {
	// Test custom response headers with template rendering
	cfg := createTestConfig([]config.RouteConfig{
		{
			Path:     "/api/data",
			Method:   "GET",
			Template: "Response data",
			ResponseHeaders: map[string]string{
				"X-Request-ID":   "{{ .Headers.Get \"X-Request-ID\" }}",
				"X-Custom-Value": "static-value",
				"Content-Type":   "application/json",
			},
		},
	})

	ts := NewTestServer(t, cfg)

	// Make request with headers that will be echoed back
	headers := map[string]string{
		"X-Request-ID": "req-123456",
	}

	resp, err := ts.makeRequest("GET", "/api/data", nil, headers)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	// Check custom response headers
	if resp.Header.Get("X-Request-ID") != "req-123456" {
		t.Errorf("Expected X-Request-ID header 'req-123456', got %q", resp.Header.Get("X-Request-ID"))
	}

	if resp.Header.Get("X-Custom-Value") != "static-value" {
		t.Errorf("Expected X-Custom-Value header 'static-value', got %q", resp.Header.Get("X-Custom-Value"))
	}

	if resp.Header.Get("Content-Type") != "application/json" {
		t.Errorf("Expected Content-Type header 'application/json', got %q", resp.Header.Get("Content-Type"))
	}
}

// Error scenario tests

func TestServer_Integration_NotFoundResponses(t *testing.T) {
	// Test 404 responses for unmatched routes
	cfg := createTestConfig([]config.RouteConfig{
		{
			Path:     "/only-path",
			Method:   "GET",
			Template: "Found",
		},
	})

	ts := NewTestServer(t, cfg)

	// Test non-existent path
	resp, err := ts.makeRequest("GET", "/non-existent", nil, nil)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", resp.StatusCode)
	}

	body := readResponseBody(t, resp)
	if !strings.Contains(body, "404 Not Found") {
		t.Errorf("Expected 404 error message, got %q", body)
	}
	if !strings.Contains(body, "GET /non-existent") {
		t.Errorf("Expected error to mention request details, got %q", body)
	}
}

func TestServer_Integration_TemplateErrors(t *testing.T) {
	// Test 500 responses for template errors
	cfg := createTestConfig([]config.RouteConfig{
		{
			Path:     "/bad-template",
			Method:   "GET",
			Template: "{{ .NonExistentField.SubField }}",
		},
	})

	ts := NewTestServer(t, cfg)

	resp, err := ts.makeRequest("GET", "/bad-template", nil, nil)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("Expected status 500 for template error, got %d", resp.StatusCode)
	}

	body := readResponseBody(t, resp)
	if !strings.Contains(body, "500 Internal Server Error") {
		t.Errorf("Expected 500 error message, got %q", body)
	}
	if !strings.Contains(body, "response template cannot be rendered due to an error in the template") {
		t.Errorf("Expected template error message, got %q", body)
	}
}

func TestServer_Integration_InvalidRequestHandling(t *testing.T) {
	// Test handling of invalid request data
	cfg := createTestConfig([]config.RouteConfig{
		{
			Path:     "/json-endpoint",
			Method:   "POST",
			Template: "JSON Data: {{ .Body }}",
		},
	})

	ts := NewTestServer(t, cfg)

	// Test with malformed JSON
	malformedJSON := `{"incomplete": json`
	headers := map[string]string{
		"Content-Type": "application/json",
	}

	resp, err := ts.makeRequest("POST", "/json-endpoint", strings.NewReader(malformedJSON), headers)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	// Should still return 200 since the template engine handles invalid JSON gracefully
	// by setting .Body to null or handling the error appropriately
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200 (graceful handling), got %d", resp.StatusCode)
	}
}

// Edge case tests

func TestServer_Integration_MissingCaptureGroups(t *testing.T) {
	// Test regex routes with missing capture groups
	cfg := createTestConfig([]config.RouteConfig{
		{
			Path:     "/^/user/([^/]+)$/", // No named capture group
			Method:   "GET",
			Template: "User: {{ .Params.name }}", // This will be empty
		},
	})

	ts := NewTestServer(t, cfg)

	resp, err := ts.makeRequest("GET", "/user/alice", nil, nil)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	body := readResponseBody(t, resp)
	// Should contain "User: " but name will be empty since it's not a named capture
	if !strings.Contains(body, "User: ") {
		t.Errorf("Expected body to contain 'User: ', got %q", body)
	}
}

func TestServer_Integration_EmptyRequestData(t *testing.T) {
	// Test handling of empty headers, query params, and body
	cfg := createTestConfig([]config.RouteConfig{
		{
			Path:   "/empty-test",
			Method: "POST",
			Template: `Headers count: {{ len .Headers }}
Query count: {{ len .Query }}
Body: {{ .Body }}`,
		},
	})

	ts := NewTestServer(t, cfg)

	// Make request with minimal data
	resp, err := ts.makeRequest("POST", "/empty-test", nil, nil)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	body := readResponseBody(t, resp)
	if !strings.Contains(body, "Headers count:") {
		t.Errorf("Expected body to handle empty headers, got %q", body)
	}
	if !strings.Contains(body, "Query count:") {
		t.Errorf("Expected body to handle empty query params, got %q", body)
	}
}

func TestServer_Integration_MultipleRoutesWithSamePattern(t *testing.T) {
	// Test multiple routes with same pattern but different methods
	cfg := createTestConfig([]config.RouteConfig{
		{
			Path:     "/api/resource",
			Method:   "GET",
			Template: "GET response",
		},
		{
			Path:     "/api/resource",
			Method:   "POST",
			Template: "POST response",
		},
		{
			Path:     "/api/resource",
			Method:   "PUT",
			Template: "PUT response",
		},
	})

	ts := NewTestServer(t, cfg)

	// Test each method
	methods := []string{"GET", "POST", "PUT"}
	for _, method := range methods {
		resp, err := ts.makeRequest(method, "/api/resource", nil, nil)
		if err != nil {
			t.Fatalf("Request failed for %s: %v", method, err)
		}

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200 for %s, got %d", method, resp.StatusCode)
		}

		body := readResponseBody(t, resp)
		expected := fmt.Sprintf("%s response", method)
		if body != expected {
			t.Errorf("Expected %q for %s, got %q", expected, method, body)
		}
	}

	// Test unsupported method
	resp, err := ts.makeRequest("DELETE", "/api/resource", nil, nil)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("Expected status 404 for unsupported method, got %d", resp.StatusCode)
	}
}

func TestServer_Integration_HeaderTemplateExecutionErrors(t *testing.T) {
	// Test header template execution errors
	cfg := createTestConfig([]config.RouteConfig{
		{
			Path:     "/bad-header-template",
			Method:   "GET",
			Template: "Response content",
			ResponseHeaders: map[string]string{
				"X-Bad-Template": "{{ .NonExistent.Field }}",
			},
		},
	})

	ts := NewTestServer(t, cfg)

	resp, err := ts.makeRequest("GET", "/bad-header-template", nil, nil)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	// Should return 500 because header template execution failed
	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("Expected status 500 for header template error, got %d", resp.StatusCode)
	}

	body := readResponseBody(t, resp)
	if !strings.Contains(body, "500 Internal Server Error") {
		t.Errorf("Expected 500 error message, got %q", body)
	}
}

func TestServer_Integration_CustomStatusCodes(t *testing.T) {
	cfg := createTestConfig([]config.RouteConfig{
		{Path: "/created", Method: "POST", Status: "201", Template: "created"},
		{Path: "/flaky", Method: "GET", Status: `{{ if .Query.Get "fail" }}503{{ else }}200{{ end }}`, Template: "flaky"},
		{Path: "/no-content", Method: "DELETE", Status: "204", Template: "this body is never sent"},
		{Path: "/not-modified", Method: "GET", Status: "304", Template: "this body is never sent"},
		{Path: "/not-a-number", Method: "GET", Status: "{{ .Query.Get \"code\" }}", Template: "body"},
		{Path: "/teapot", Method: "GET", Status: "418", Template: "short and stout", ResponseHeaders: map[string]string{"X-Kind": "teapot"}},
	})
	ts := NewTestServer(t, cfg)

	tests := []struct {
		name       string
		method     string
		path       string
		wantStatus int
		wantBody   string
	}{
		{name: "static status with body", method: "POST", path: "/created", wantStatus: 201, wantBody: "created"},
		{name: "templated status takes the error branch", method: "GET", path: "/flaky?fail=1", wantStatus: 503, wantBody: "flaky"},
		{name: "templated status takes the success branch", method: "GET", path: "/flaky", wantStatus: 200, wantBody: "flaky"},
		{name: "204 sends no body", method: "DELETE", path: "/no-content", wantStatus: 204, wantBody: ""},
		{name: "304 sends no body", method: "GET", path: "/not-modified", wantStatus: 304, wantBody: ""},
		{name: "templated status rendering a number", method: "GET", path: "/not-a-number?code=404", wantStatus: 404, wantBody: "body"},
		{name: "templated status rendering text is a server error", method: "GET", path: "/not-a-number?code=abc", wantStatus: 500, wantBody: "500 Internal Server Error: response template cannot be rendered due to an error in the template\n"},
		{name: "templated status rendering nothing is a server error", method: "GET", path: "/not-a-number", wantStatus: 500, wantBody: "500 Internal Server Error: response template cannot be rendered due to an error in the template\n"},
		{name: "templated status out of range is a server error", method: "GET", path: "/not-a-number?code=700", wantStatus: 500, wantBody: "500 Internal Server Error: response template cannot be rendered due to an error in the template\n"},
		{name: "templated status in the informational range is a server error", method: "GET", path: "/not-a-number?code=100", wantStatus: 500, wantBody: "500 Internal Server Error: response template cannot be rendered due to an error in the template\n"},
		{name: "non-2xx status keeps response headers", method: "GET", path: "/teapot", wantStatus: 418, wantBody: "short and stout"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := ts.makeRequest(tt.method, tt.path, nil, nil)
			if err != nil {
				t.Fatalf("%s %s failed: %v", tt.method, tt.path, err)
			}
			body := readResponseBody(t, resp)

			if resp.StatusCode != tt.wantStatus {
				t.Errorf("%s %s status = %d, want %d", tt.method, tt.path, resp.StatusCode, tt.wantStatus)
			}
			if body != tt.wantBody {
				t.Errorf("%s %s body = %q, want %q", tt.method, tt.path, body, tt.wantBody)
			}
		})
	}

	resp, err := ts.makeRequest("GET", "/teapot", nil, nil)
	if err != nil {
		t.Fatalf("GET /teapot failed: %v", err)
	}
	resp.Body.Close()
	if got := resp.Header.Get("X-Kind"); got != "teapot" {
		t.Errorf("GET /teapot X-Kind header = %q, want %q", got, "teapot")
	}
}

func TestServer_Integration_RouteWithoutMethodMatchesAnyMethod(t *testing.T) {
	cfg := createTestConfig([]config.RouteConfig{
		{Path: "/get-only", Method: "GET", Template: "get only"},
		{Path: "/.*/", Template: "catch-all {{ .Request.Method }}"},
	})
	ts := NewTestServer(t, cfg)

	tests := []struct {
		method   string
		path     string
		wantBody string
	}{
		{method: "GET", path: "/get-only", wantBody: "get only"},
		{method: "POST", path: "/get-only", wantBody: "catch-all POST"},
		{method: "DELETE", path: "/anything/else", wantBody: "catch-all DELETE"},
		{method: "PATCH", path: "/", wantBody: "catch-all PATCH"},
	}

	for _, tt := range tests {
		t.Run(tt.method+" "+tt.path, func(t *testing.T) {
			resp, err := ts.makeRequest(tt.method, tt.path, nil, nil)
			if err != nil {
				t.Fatalf("%s %s failed: %v", tt.method, tt.path, err)
			}
			body := readResponseBody(t, resp)

			if resp.StatusCode != http.StatusOK {
				t.Errorf("%s %s status = %d, want 200", tt.method, tt.path, resp.StatusCode)
			}
			if body != tt.wantBody {
				t.Errorf("%s %s body = %q, want %q", tt.method, tt.path, body, tt.wantBody)
			}
		})
	}
}

func TestServer_Integration_Delay(t *testing.T) {
	cfg := createTestConfig([]config.RouteConfig{
		{Path: "/slow", Method: "GET", Delay: 150 * time.Millisecond, Template: "slow"},
		{Path: "/very-slow", Method: "GET", Delay: 10 * time.Second, Template: "never sent"},
	})
	ts := NewTestServer(t, cfg)

	t.Run("response waits for the delay", func(t *testing.T) {
		start := time.Now()
		resp, err := ts.makeRequest("GET", "/slow", nil, nil)
		if err != nil {
			t.Fatalf("GET /slow failed: %v", err)
		}
		body := readResponseBody(t, resp)
		elapsed := time.Since(start)

		if elapsed < 150*time.Millisecond {
			t.Errorf("GET /slow took %s, want at least 150ms", elapsed)
		}
		if resp.StatusCode != http.StatusOK || body != "slow" {
			t.Errorf("GET /slow = %d %q, want 200 %q", resp.StatusCode, body, "slow")
		}
	})

	t.Run("cancelled request stops waiting", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(t.Context(), 50*time.Millisecond)
		defer cancel()
		req := httptest.NewRequest("GET", "/very-slow", nil).WithContext(ctx)
		rec := httptest.NewRecorder()

		start := time.Now()
		ts.Server.ServeHTTP(rec, req)
		elapsed := time.Since(start)

		if rec.Code != http.StatusRequestTimeout {
			t.Errorf("GET /very-slow with 50ms deadline status = %d, want 408", rec.Code)
		}
		if elapsed > 2*time.Second {
			t.Errorf("GET /very-slow with 50ms deadline took %s, want it to stop at the deadline", elapsed)
		}
	})
}

func TestServer_ReloadIsNotBlockedByDelayedRequest(t *testing.T) {
	configFile := filepath.Join(t.TempDir(), "config.yaml")
	writeConfig := func(body string) {
		t.Helper()
		if err := os.WriteFile(configFile, []byte(body), 0o644); err != nil {
			t.Fatalf("failed to write config file: %v", err)
		}
	}
	writeConfig("routes:\n  - path: /slow\n    delay: 3s\n    template: slow\n")

	cfg, err := config.LoadConfig(configFile)
	if err != nil {
		t.Fatalf("LoadConfig(%q) returned unexpected error: %v", configFile, err)
	}
	srv, err := NewServer(cfg, configFile, ":0", slog.New(slog.NewTextHandler(io.Discard, nil)), "test-version")
	if err != nil {
		t.Fatalf("NewServer() returned unexpected error: %v", err)
	}

	ctx, cancel := context.WithCancel(t.Context())
	requestDone := make(chan struct{})
	go func() {
		defer close(requestDone)
		srv.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/slow", nil).WithContext(ctx))
	}()
	t.Cleanup(func() {
		cancel()
		<-requestDone
	})
	time.Sleep(100 * time.Millisecond) // let the request reach its delay

	writeConfig("routes:\n  - path: /fast\n    template: fast\n")
	start := time.Now()
	if err := srv.ReloadConfig(); err != nil {
		t.Fatalf("ReloadConfig() returned unexpected error: %v", err)
	}

	if elapsed := time.Since(start); elapsed > time.Second {
		t.Errorf("ReloadConfig() took %s while a delayed request was in flight, want under 1s", elapsed)
	}
}

func TestServer_Integration_FormFields(t *testing.T) {
	cfg := createTestConfig([]config.RouteConfig{
		{Path: "/login", Method: "POST", Template: `user={{ .Form.Get "user" }} tags={{ index .Form "tag" }}`},
	})
	ts := NewTestServer(t, cfg)

	resp, err := ts.makeRequest("POST", "/login", strings.NewReader("user=alice&tag=a&tag=b"), map[string]string{
		"Content-Type": "application/x-www-form-urlencoded",
	})
	if err != nil {
		t.Fatalf("POST /login failed: %v", err)
	}
	body := readResponseBody(t, resp)

	if want := "user=alice tags=[a b]"; body != want {
		t.Errorf("POST /login body = %q, want %q", body, want)
	}
}

func TestServer_HealthCheckReportsVersion(t *testing.T) {
	ts := NewTestServer(t, createTestConfig([]config.RouteConfig{
		{Path: "/x", Method: "GET", Template: "x"},
	}))

	resp, err := ts.makeRequest("GET", "/health", nil, nil)
	if err != nil {
		t.Fatalf("GET /health failed: %v", err)
	}
	defer resp.Body.Close()

	var health HealthCheckResponse
	if err := json.NewDecoder(resp.Body).Decode(&health); err != nil {
		t.Fatalf("failed to decode health check response: %v", err)
	}
	if health.Version != "test-version" {
		t.Errorf("GET /health version = %q, want %q", health.Version, "test-version")
	}
}
