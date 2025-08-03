package template

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// TemplateContext represents the data available to templates during rendering
type TemplateContext struct {
	// Request provides access to the raw HTTP request
	Request *http.Request `json:"-"`

	// Headers contains all HTTP headers with full access to http.Header methods
	Headers http.Header `json:"headers"`

	// Query contains all query parameters with full access to url.Values methods
	Query url.Values `json:"query"`

	// Body contains the parsed request body (JSON if applicable, string otherwise)
	Body interface{} `json:"body"`

	// Params contains named capture groups from regex route patterns
	Params map[string]string `json:"params"`
}

// NewTemplateContext creates a new TemplateContext from an HTTP request and route parameters
func NewTemplateContext(req *http.Request, params map[string]string) (*TemplateContext, error) {
	ctx := &TemplateContext{
		Request: req,
		Headers: req.Header,
		Query:   req.URL.Query(),
		Params:  params,
	}

	// Parse request body
	body, err := parseRequestBody(req)
	if err != nil {
		// Don't fail the entire context creation for body parsing errors
		// Just set body to the error message
		ctx.Body = err.Error()
	} else {
		ctx.Body = body
	}

	return ctx, nil
}

// parseRequestBody attempts to parse the request body
// Returns parsed JSON if Content-Type indicates JSON, otherwise returns raw string
func parseRequestBody(req *http.Request) (interface{}, error) {
	if req.Body == nil {
		return nil, nil
	}

	// Read the body
	bodyBytes, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, &ContextError{
			Component: "body",
			Message:   "failed to read request body",
			Cause:     err,
		}
	}

	// Check if body is empty
	if len(bodyBytes) == 0 {
		return nil, nil
	}

	// Get content type
	contentType := req.Header.Get("Content-Type")

	// Attempt JSON parsing if content type suggests JSON
	if isJSONContentType(contentType) {
		var jsonBody interface{}
		if err := json.Unmarshal(bodyBytes, &jsonBody); err != nil {
			// If JSON parsing fails, return as string with error info
			return map[string]interface{}{
				"raw":         string(bodyBytes),
				"parse_error": err.Error(),
			}, nil
		}
		return jsonBody, nil
	}

	// Return as string for non-JSON content
	return string(bodyBytes), nil
}

// isJSONContentType checks if the content type indicates JSON
func isJSONContentType(contentType string) bool {
	// Handle common JSON content types
	contentType = strings.ToLower(strings.TrimSpace(contentType))

	return strings.Contains(contentType, "application/json") ||
		strings.Contains(contentType, "text/json") ||
		strings.HasSuffix(contentType, "+json")
}
