package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
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
	server, err := NewServer(cfg, "test-config.yaml", ":0", logger)
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
			Verb:     "GET",
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
			Path: "/^/user/(?P<name>[^/]+)$/",
			Verb: "GET",
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
	if !strings.Contains(body, "User-Agent: test-client/1.0") {
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
			Verb:     "POST",
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
			Path: "/context-test",
			Verb: "GET",
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
		"Query param 'debug': true",
		"Header 'X-Custom': test-value",
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
			Path: "/api/secure",
			Verb: "POST",
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
			Verb:     "GET",
			Template: "Response data",
			ResponseHeaders: map[string]string{
				"X-Request-ID":   "{{ index .Headers \"X-Request-Id\" }}",
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
			Verb:     "GET",
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
			Verb:     "GET",
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
	if !strings.Contains(body, "Template execution failed") {
		t.Errorf("Expected template error message, got %q", body)
	}
}

func TestServer_Integration_InvalidRequestHandling(t *testing.T) {
	// Test handling of invalid request data
	cfg := createTestConfig([]config.RouteConfig{
		{
			Path:     "/json-endpoint",
			Verb:     "POST",
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
			Verb:     "GET",
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
			Path: "/empty-test",
			Verb: "POST",
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
	// Test multiple routes with same pattern but different verbs
	cfg := createTestConfig([]config.RouteConfig{
		{
			Path:     "/api/resource",
			Verb:     "GET",
			Template: "GET response",
		},
		{
			Path:     "/api/resource",
			Verb:     "POST",
			Template: "POST response",
		},
		{
			Path:     "/api/resource",
			Verb:     "PUT",
			Template: "PUT response",
		},
	})

	ts := NewTestServer(t, cfg)

	// Test each verb
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

	// Test unsupported verb
	resp, err := ts.makeRequest("DELETE", "/api/resource", nil, nil)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("Expected status 404 for unsupported verb, got %d", resp.StatusCode)
	}
}

func TestServer_Integration_HeaderTemplateExecutionErrors(t *testing.T) {
	// Test header template execution errors
	cfg := createTestConfig([]config.RouteConfig{
		{
			Path:     "/bad-header-template",
			Verb:     "GET",
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
