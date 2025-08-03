package router

import (
	"net/http"
	"net/url"
	"regexp"
	"testing"
)

func TestRoute_MatchRequest_LiteralPaths(t *testing.T) {
	tests := []struct {
		name       string
		pattern    string
		verb       string
		reqMethod  string
		reqPath    string
		wantMatch  bool
		wantParams map[string]string
	}{
		{
			name:       "exact path match",
			pattern:    "/health",
			verb:       "GET",
			reqMethod:  "GET",
			reqPath:    "/health",
			wantMatch:  true,
			wantParams: map[string]string{},
		},
		{
			name:       "root path match",
			pattern:    "/",
			verb:       "GET",
			reqMethod:  "GET",
			reqPath:    "/",
			wantMatch:  true,
			wantParams: map[string]string{},
		},
		{
			name:       "nested path match",
			pattern:    "/api/v1/users",
			verb:       "GET",
			reqMethod:  "GET",
			reqPath:    "/api/v1/users",
			wantMatch:  true,
			wantParams: map[string]string{},
		},
		{
			name:       "path mismatch",
			pattern:    "/health",
			verb:       "GET",
			reqMethod:  "GET",
			reqPath:    "/status",
			wantMatch:  false,
			wantParams: nil,
		},
		{
			name:       "partial path mismatch",
			pattern:    "/api",
			verb:       "GET",
			reqMethod:  "GET",
			reqPath:    "/api/v1",
			wantMatch:  false,
			wantParams: nil,
		},
		{
			name:       "verb mismatch",
			pattern:    "/health",
			verb:       "GET",
			reqMethod:  "POST",
			reqPath:    "/health",
			wantMatch:  false,
			wantParams: nil,
		},
		{
			name:       "case insensitive verb match",
			pattern:    "/health",
			verb:       "GET",
			reqMethod:  "get",
			reqPath:    "/health",
			wantMatch:  true,
			wantParams: map[string]string{},
		},
		{
			name:       "mixed case verb match",
			pattern:    "/health",
			verb:       "POST",
			reqMethod:  "Post",
			reqPath:    "/health",
			wantMatch:  true,
			wantParams: map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			route := &Route{
				Pattern:  tt.pattern,
				Method:   tt.verb,
				IsRegexp: false,
				Regex:    nil,
			}

			req, err := http.NewRequest(tt.reqMethod, tt.reqPath, nil)
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}

			match, gotMatch := route.MatchRequest(req)
			if gotMatch != tt.wantMatch {
				t.Errorf("MatchRequest() match = %v, want %v", gotMatch, tt.wantMatch)
				return
			}

			if tt.wantMatch {
				if match == nil {
					t.Error("MatchRequest() should return non-nil match for successful match")
					return
				}

				if match.Route != route {
					t.Error("MatchRequest() match.Route should point to the original route")
				}

				if len(match.Params) != len(tt.wantParams) {
					t.Errorf("MatchRequest() params length = %v, want %v", len(match.Params), len(tt.wantParams))
				}

				for key, expectedValue := range tt.wantParams {
					if actualValue, exists := match.Params[key]; !exists || actualValue != expectedValue {
						t.Errorf("MatchRequest() params[%s] = %v, want %v", key, actualValue, expectedValue)
					}
				}
			} else {
				if match != nil {
					t.Error("MatchRequest() should return nil match for failed match")
				}
			}
		})
	}
}

func TestRoute_MatchRequest_RegexPaths(t *testing.T) {
	tests := []struct {
		name        string
		pattern     string
		verb        string
		reqMethod   string
		reqPath     string
		wantMatch   bool
		wantParams  map[string]string
		description string
	}{
		{
			name:        "simple regex match",
			pattern:     "^/user/[0-9]+$",
			verb:        "GET",
			reqMethod:   "GET",
			reqPath:     "/user/123",
			wantMatch:   true,
			wantParams:  map[string]string{},
			description: "Simple regex without named groups",
		},
		{
			name:        "regex with single named group",
			pattern:     "^/user/(?P<id>[0-9]+)$",
			verb:        "GET",
			reqMethod:   "GET",
			reqPath:     "/user/123",
			wantMatch:   true,
			wantParams:  map[string]string{"id": "123"},
			description: "Regex with single named capture group",
		},
		{
			name:        "regex with multiple named groups",
			pattern:     "^/user/(?P<id>[0-9]+)/posts/(?P<postId>[0-9]+)$",
			verb:        "GET",
			reqMethod:   "GET",
			reqPath:     "/user/123/posts/456",
			wantMatch:   true,
			wantParams:  map[string]string{"id": "123", "postId": "456"},
			description: "Regex with multiple named capture groups",
		},
		{
			name:        "regex with optional groups",
			pattern:     "^/api/v(?P<version>[12])(/(?P<endpoint>.*))?$",
			verb:        "GET",
			reqMethod:   "GET",
			reqPath:     "/api/v1/users",
			wantMatch:   true,
			wantParams:  map[string]string{"version": "1", "endpoint": "users"},
			description: "Regex with optional named groups",
		},
		{
			name:        "regex with optional groups - partial match",
			pattern:     "^/api/v(?P<version>[12])(/(?P<endpoint>.*))?$",
			verb:        "GET",
			reqMethod:   "GET",
			reqPath:     "/api/v1",
			wantMatch:   true,
			wantParams:  map[string]string{"version": "1", "endpoint": ""},
			description: "Regex where optional group doesn't match",
		},
		{
			name:        "regex mismatch - wrong pattern",
			pattern:     "^/user/(?P<id>[0-9]+)$",
			verb:        "GET",
			reqMethod:   "GET",
			reqPath:     "/user/abc",
			wantMatch:   false,
			wantParams:  nil,
			description: "Path doesn't match regex pattern",
		},
		{
			name:        "regex mismatch - wrong verb",
			pattern:     "^/user/(?P<id>[0-9]+)$",
			verb:        "GET",
			reqMethod:   "POST",
			reqPath:     "/user/123",
			wantMatch:   false,
			wantParams:  nil,
			description: "Regex matches but verb doesn't",
		},
		{
			name:        "regex with word boundaries",
			pattern:     "^/search/(?P<term>\\w+)$",
			verb:        "GET",
			reqMethod:   "GET",
			reqPath:     "/search/golang",
			wantMatch:   true,
			wantParams:  map[string]string{"term": "golang"},
			description: "Regex with word character class",
		},
		{
			name:        "regex case sensitive match",
			pattern:     "^/user/(?P<name>[A-Z][a-z]+)$",
			verb:        "GET",
			reqMethod:   "GET",
			reqPath:     "/user/John",
			wantMatch:   true,
			wantParams:  map[string]string{"name": "John"},
			description: "Case sensitive regex pattern",
		},
		{
			name:        "regex case sensitive mismatch",
			pattern:     "^/user/(?P<name>[A-Z][a-z]+)$",
			verb:        "GET",
			reqMethod:   "GET",
			reqPath:     "/user/john",
			wantMatch:   false,
			wantParams:  nil,
			description: "Case sensitive regex doesn't match lowercase",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			route := createRegexRoute(t, tt.pattern, tt.verb)

			req, err := http.NewRequest(tt.reqMethod, tt.reqPath, nil)
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}

			match, gotMatch := route.MatchRequest(req)
			if gotMatch != tt.wantMatch {
				t.Errorf("MatchRequest() match = %v, want %v for %s", gotMatch, tt.wantMatch, tt.description)
				return
			}

			if tt.wantMatch {
				if match == nil {
					t.Error("MatchRequest() should return non-nil match for successful match")
					return
				}

				if match.Route != route {
					t.Error("MatchRequest() match.Route should point to the original route")
				}

				if len(match.Params) != len(tt.wantParams) {
					t.Errorf("MatchRequest() params length = %v, want %v", len(match.Params), len(tt.wantParams))
				}

				for key, expectedValue := range tt.wantParams {
					if actualValue, exists := match.Params[key]; !exists || actualValue != expectedValue {
						t.Errorf("MatchRequest() params[%s] = %v, want %v", key, actualValue, expectedValue)
					}
				}
			} else {
				if match != nil {
					t.Error("MatchRequest() should return nil match for failed match")
				}
			}
		})
	}
}

func TestRoute_MatchRequest_HTTPMethods(t *testing.T) {
	methods := []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS", "CONNECT", "TRACE"}

	for _, method := range methods {
		t.Run("method_"+method, func(t *testing.T) {
			route := &Route{
				Pattern:  "/test",
				Method:   method,
				IsRegexp: false,
				Regex:    nil,
			}

			// Test matching method
			req, err := http.NewRequest(method, "/test", nil)
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}

			match, gotMatch := route.MatchRequest(req)
			if !gotMatch {
				t.Errorf("MatchRequest() should match method %s", method)
				return
			}

			if match == nil {
				t.Error("MatchRequest() should return non-nil match")
				return
			}

			// Test non-matching method (use different method)
			otherMethod := "POST"
			if method == "POST" {
				otherMethod = "GET"
			}

			req2, err := http.NewRequest(otherMethod, "/test", nil)
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}

			match2, gotMatch2 := route.MatchRequest(req2)
			if gotMatch2 {
				t.Errorf("MatchRequest() should not match different method %s when route expects %s", otherMethod, method)
			}

			if match2 != nil {
				t.Error("MatchRequest() should return nil match for different method")
			}
		})
	}
}

func TestRoute_MatchRequest_EdgeCases(t *testing.T) {
	tests := []struct {
		name      string
		route     *Route
		reqMethod string
		reqPath   string
		wantMatch bool
		wantPanic bool
	}{
		{
			name: "nil regex in regex route",
			route: &Route{
				Pattern:  "^/test$",
				Method:   "GET",
				IsRegexp: true,
				Regex:    nil,
			},
			reqMethod: "GET",
			reqPath:   "/test",
			wantMatch: false,
			wantPanic: false,
		},
		{
			name: "empty pattern literal route",
			route: &Route{
				Pattern:  "",
				Method:   "GET",
				IsRegexp: false,
				Regex:    nil,
			},
			reqMethod: "GET",
			reqPath:   "",
			wantMatch: true,
			wantPanic: false,
		},
		{
			name: "empty pattern regex route",
			route: &Route{
				Pattern:  "",
				Method:   "GET",
				IsRegexp: true,
				Regex:    nil,
			},
			reqMethod: "GET",
			reqPath:   "",
			wantMatch: false,
			wantPanic: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil && !tt.wantPanic {
					t.Errorf("MatchRequest() unexpected panic: %v", r)
				}
			}()

			req, err := http.NewRequest(tt.reqMethod, tt.reqPath, nil)
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}

			_, gotMatch := tt.route.MatchRequest(req)
			if gotMatch != tt.wantMatch {
				t.Errorf("MatchRequest() match = %v, want %v", gotMatch, tt.wantMatch)
			}
		})
	}
}

func TestRoute_String(t *testing.T) {
	tests := []struct {
		name     string
		route    *Route
		expected string
	}{
		{
			name: "literal route",
			route: &Route{
				Pattern:        "/health",
				Method:         "GET",
				IsRegexp:       false,
				TemplateSource: "inline",
			},
			expected: "GET /health (literal) template=inline",
		},
		{
			name: "regex route",
			route: &Route{
				Pattern:        "^/user/(?P<id>[0-9]+)$",
				Method:         "POST",
				IsRegexp:       true,
				TemplateSource: "user.tmpl",
			},
			expected: "POST ^/user/(?P<id>[0-9]+)$ (regex) template=user.tmpl",
		},
		{
			name: "route with empty template source",
			route: &Route{
				Pattern:        "/test",
				Method:         "PUT",
				IsRegexp:       false,
				TemplateSource: "",
			},
			expected: "PUT /test (literal) template=",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.route.String(); got != tt.expected {
				t.Errorf("Route.String() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestRoute_matchesMethod(t *testing.T) {
	tests := []struct {
		name        string
		routeMethod string
		reqMethod   string
		want        bool
	}{
		{
			name:        "exact match uppercase",
			routeMethod: "GET",
			reqMethod:   "GET",
			want:        true,
		},
		{
			name:        "case insensitive match",
			routeMethod: "GET",
			reqMethod:   "get",
			want:        true,
		},
		{
			name:        "mixed case match",
			routeMethod: "POST",
			reqMethod:   "Post",
			want:        true,
		},
		{
			name:        "no match",
			routeMethod: "GET",
			reqMethod:   "POST",
			want:        false,
		},
		{
			name:        "empty route method",
			routeMethod: "",
			reqMethod:   "GET",
			want:        false,
		},
		{
			name:        "empty request method",
			routeMethod: "GET",
			reqMethod:   "",
			want:        false,
		},
		{
			name:        "both empty",
			routeMethod: "",
			reqMethod:   "",
			want:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			route := &Route{Method: tt.routeMethod}
			if got := route.matchesMethod(tt.reqMethod); got != tt.want {
				t.Errorf("matchesMethod() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRoute_matchLiteralPattern(t *testing.T) {
	tests := []struct {
		name       string
		pattern    string
		path       string
		wantMatch  bool
		wantParams map[string]string
	}{
		{
			name:       "exact match",
			pattern:    "/health",
			path:       "/health",
			wantMatch:  true,
			wantParams: map[string]string{},
		},
		{
			name:       "no match",
			pattern:    "/health",
			path:       "/status",
			wantMatch:  false,
			wantParams: nil,
		},
		{
			name:       "root path match",
			pattern:    "/",
			path:       "/",
			wantMatch:  true,
			wantParams: map[string]string{},
		},
		{
			name:       "empty pattern and path",
			pattern:    "",
			path:       "",
			wantMatch:  true,
			wantParams: map[string]string{},
		},
		{
			name:       "partial match fails",
			pattern:    "/api",
			path:       "/api/v1",
			wantMatch:  false,
			wantParams: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			route := &Route{Pattern: tt.pattern}
			match, gotMatch := route.matchLiteralPattern(tt.path)

			if gotMatch != tt.wantMatch {
				t.Errorf("matchLiteralPattern() match = %v, want %v", gotMatch, tt.wantMatch)
				return
			}

			if tt.wantMatch {
				if match == nil {
					t.Error("matchLiteralPattern() should return non-nil match for successful match")
					return
				}

				if match.Route != route {
					t.Error("matchLiteralPattern() match.Route should point to the original route")
				}

				if len(match.Params) != len(tt.wantParams) {
					t.Errorf("matchLiteralPattern() params length = %v, want %v", len(match.Params), len(tt.wantParams))
				}
			} else {
				if match != nil {
					t.Error("matchLiteralPattern() should return nil match for failed match")
				}
			}
		})
	}
}

func TestRoute_matchRegexPattern(t *testing.T) {
	tests := []struct {
		name       string
		pattern    string
		path       string
		wantMatch  bool
		wantParams map[string]string
	}{
		{
			name:       "simple regex match",
			pattern:    "^/user/[0-9]+$",
			path:       "/user/123",
			wantMatch:  true,
			wantParams: map[string]string{},
		},
		{
			name:       "regex with named group",
			pattern:    "^/user/(?P<id>[0-9]+)$",
			path:       "/user/123",
			wantMatch:  true,
			wantParams: map[string]string{"id": "123"},
		},
		{
			name:       "regex with multiple named groups",
			pattern:    "^/user/(?P<id>[0-9]+)/posts/(?P<postId>[0-9]+)$",
			path:       "/user/123/posts/456",
			wantMatch:  true,
			wantParams: map[string]string{"id": "123", "postId": "456"},
		},
		{
			name:       "regex no match",
			pattern:    "^/user/[0-9]+$",
			path:       "/user/abc",
			wantMatch:  false,
			wantParams: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			route := createRegexRoute(t, tt.pattern, "GET")
			match, gotMatch := route.matchRegexPattern(tt.path)

			if gotMatch != tt.wantMatch {
				t.Errorf("matchRegexPattern() match = %v, want %v", gotMatch, tt.wantMatch)
				return
			}

			if tt.wantMatch {
				if match == nil {
					t.Error("matchRegexPattern() should return non-nil match for successful match")
					return
				}

				if match.Route != route {
					t.Error("matchRegexPattern() match.Route should point to the original route")
				}

				if len(match.Params) != len(tt.wantParams) {
					t.Errorf("matchRegexPattern() params length = %v, want %v", len(match.Params), len(tt.wantParams))
				}

				for key, expectedValue := range tt.wantParams {
					if actualValue, exists := match.Params[key]; !exists || actualValue != expectedValue {
						t.Errorf("matchRegexPattern() params[%s] = %v, want %v", key, actualValue, expectedValue)
					}
				}
			} else {
				if match != nil {
					t.Error("matchRegexPattern() should return nil match for failed match")
				}
			}
		})
	}
}

func TestRoute_matchRegexPattern_NilRegex(t *testing.T) {
	route := &Route{
		Pattern:  "^/test$",
		Method:   "GET",
		IsRegexp: true,
		Regex:    nil,
	}

	match, gotMatch := route.matchRegexPattern("/test")
	if gotMatch {
		t.Error("matchRegexPattern() should not match when regex is nil")
	}

	if match != nil {
		t.Error("matchRegexPattern() should return nil match when regex is nil")
	}
}

// Performance benchmarks
func BenchmarkRoute_MatchRequest_Literal(b *testing.B) {
	route := &Route{
		Pattern:  "/api/v1/users",
		Method:   "GET",
		IsRegexp: false,
		Regex:    nil,
	}

	req, err := http.NewRequest("GET", "/api/v1/users", nil)
	if err != nil {
		b.Fatalf("Failed to create request: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = route.MatchRequest(req)
	}
}

func BenchmarkRoute_MatchRequest_Regex(b *testing.B) {
	route := createRegexRoute(nil, "^/user/(?P<id>[0-9]+)/posts/(?P<postId>[0-9]+)$", "GET")

	req, err := http.NewRequest("GET", "/user/123/posts/456", nil)
	if err != nil {
		b.Fatalf("Failed to create request: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = route.MatchRequest(req)
	}
}

func BenchmarkRoute_MatchRequest_RegexNoMatch(b *testing.B) {
	route := createRegexRoute(nil, "^/user/(?P<id>[0-9]+)$", "GET")

	req, err := http.NewRequest("GET", "/different/path", nil)
	if err != nil {
		b.Fatalf("Failed to create request: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = route.MatchRequest(req)
	}
}

// Helper functions
func createRegexRoute(t *testing.T, pattern, verb string) *Route {
	route := &Route{
		Pattern:  pattern,
		Method:   verb,
		IsRegexp: true,
	}

	// Compile the regex
	regex, err := regexp.Compile(pattern)
	if err != nil && t != nil {
		t.Fatalf("Failed to compile regex %q: %v", pattern, err)
	}
	route.Regex = regex

	return route
}

func createRequestWithURL(method, path string) *http.Request {
	u, _ := url.Parse(path)
	return &http.Request{
		Method: method,
		URL:    u,
		Header: make(http.Header),
	}
}

func TestRoute_MatchRequest_WithHeaders(t *testing.T) {
	tests := []struct {
		name         string
		pattern      string
		verb         string
		matchHeaders map[string]*HeaderMatcher
		reqMethod    string
		reqPath      string
		reqHeaders   map[string]string
		wantMatch    bool
	}{
		{
			name:         "no header requirements - should match",
			pattern:      "/api/test",
			verb:         "GET",
			matchHeaders: nil,
			reqMethod:    "GET",
			reqPath:      "/api/test",
			reqHeaders:   map[string]string{},
			wantMatch:    true,
		},
		{
			name:         "empty header requirements - should match",
			pattern:      "/api/test",
			verb:         "GET",
			matchHeaders: map[string]*HeaderMatcher{},
			reqMethod:    "GET",
			reqPath:      "/api/test",
			reqHeaders:   map[string]string{},
			wantMatch:    true,
		},
		{
			name:    "literal header match - exact match",
			pattern: "/api/test",
			verb:    "GET",
			matchHeaders: map[string]*HeaderMatcher{
				"content-type": {
					IsRegex: false,
					Literal: "application/json",
				},
			},
			reqMethod: "GET",
			reqPath:   "/api/test",
			reqHeaders: map[string]string{
				"Content-Type": "application/json",
			},
			wantMatch: true,
		},
		{
			name:    "literal header match - case insensitive header name",
			pattern: "/api/test",
			verb:    "GET",
			matchHeaders: map[string]*HeaderMatcher{
				"content-type": {
					IsRegex: false,
					Literal: "application/json",
				},
			},
			reqMethod: "GET",
			reqPath:   "/api/test",
			reqHeaders: map[string]string{
				"CONTENT-TYPE": "application/json",
			},
			wantMatch: true,
		},
		{
			name:    "literal header match - value mismatch",
			pattern: "/api/test",
			verb:    "GET",
			matchHeaders: map[string]*HeaderMatcher{
				"content-type": {
					IsRegex: false,
					Literal: "application/json",
				},
			},
			reqMethod: "GET",
			reqPath:   "/api/test",
			reqHeaders: map[string]string{
				"Content-Type": "text/html",
			},
			wantMatch: false,
		},
		{
			name:    "missing required header",
			pattern: "/api/test",
			verb:    "GET",
			matchHeaders: map[string]*HeaderMatcher{
				"authorization": {
					IsRegex: false,
					Literal: "Bearer secret",
				},
			},
			reqMethod:  "GET",
			reqPath:    "/api/test",
			reqHeaders: map[string]string{},
			wantMatch:  false,
		},
		{
			name:    "regex header match - valid pattern",
			pattern: "/api/test",
			verb:    "GET",
			matchHeaders: map[string]*HeaderMatcher{
				"authorization": {
					IsRegex: true,
					Regex:   regexp.MustCompile("Bearer .+"),
				},
			},
			reqMethod: "GET",
			reqPath:   "/api/test",
			reqHeaders: map[string]string{
				"Authorization": "Bearer abc123",
			},
			wantMatch: true,
		},
		{
			name:    "regex header match - pattern doesn't match",
			pattern: "/api/test",
			verb:    "GET",
			matchHeaders: map[string]*HeaderMatcher{
				"authorization": {
					IsRegex: true,
					Regex:   regexp.MustCompile("Bearer .+"),
				},
			},
			reqMethod: "GET",
			reqPath:   "/api/test",
			reqHeaders: map[string]string{
				"Authorization": "Basic abc123",
			},
			wantMatch: false,
		},
		{
			name:    "multiple headers - all match",
			pattern: "/api/test",
			verb:    "POST",
			matchHeaders: map[string]*HeaderMatcher{
				"content-type": {
					IsRegex: false,
					Literal: "application/json",
				},
				"authorization": {
					IsRegex: true,
					Regex:   regexp.MustCompile("Bearer .+"),
				},
			},
			reqMethod: "POST",
			reqPath:   "/api/test",
			reqHeaders: map[string]string{
				"Content-Type":  "application/json",
				"Authorization": "Bearer token123",
			},
			wantMatch: true,
		},
		{
			name:    "multiple headers - one doesn't match",
			pattern: "/api/test",
			verb:    "POST",
			matchHeaders: map[string]*HeaderMatcher{
				"content-type": {
					IsRegex: false,
					Literal: "application/json",
				},
				"authorization": {
					IsRegex: true,
					Regex:   regexp.MustCompile("Bearer .+"),
				},
			},
			reqMethod: "POST",
			reqPath:   "/api/test",
			reqHeaders: map[string]string{
				"Content-Type":  "text/plain",
				"Authorization": "Bearer token123",
			},
			wantMatch: false,
		},
		{
			name:    "path matches but headers don't",
			pattern: "/api/test",
			verb:    "GET",
			matchHeaders: map[string]*HeaderMatcher{
				"x-api-key": {
					IsRegex: false,
					Literal: "secret",
				},
			},
			reqMethod:  "GET",
			reqPath:    "/api/test",
			reqHeaders: map[string]string{},
			wantMatch:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			route := &Route{
				Pattern:      tt.pattern,
				Method:       tt.verb,
				IsRegexp:     false,
				MatchHeaders: tt.matchHeaders,
			}

			req := createRequestWithURL(tt.reqMethod, tt.reqPath)

			// Add headers to request
			for name, value := range tt.reqHeaders {
				req.Header.Set(name, value)
			}

			match, ok := route.MatchRequest(req)

			if ok != tt.wantMatch {
				t.Errorf("Route.MatchRequest() = %v, want %v", ok, tt.wantMatch)
			}

			if tt.wantMatch && match == nil {
				t.Error("Route.MatchRequest() returned true but match is nil")
			}

			if !tt.wantMatch && match != nil {
				t.Error("Route.MatchRequest() returned false but match is not nil")
			}
		})
	}
}

func TestHeaderMatcher_MatchHeaderValue(t *testing.T) {
	tests := []struct {
		name    string
		matcher *HeaderMatcher
		value   string
		want    bool
	}{
		{
			name: "literal match - exact",
			matcher: &HeaderMatcher{
				IsRegex: false,
				Literal: "application/json",
			},
			value: "application/json",
			want:  true,
		},
		{
			name: "literal match - case sensitive",
			matcher: &HeaderMatcher{
				IsRegex: false,
				Literal: "application/json",
			},
			value: "Application/JSON",
			want:  false,
		},
		{
			name: "literal match - different value",
			matcher: &HeaderMatcher{
				IsRegex: false,
				Literal: "application/json",
			},
			value: "text/html",
			want:  false,
		},
		{
			name: "regex match - matches",
			matcher: &HeaderMatcher{
				IsRegex: true,
				Regex:   regexp.MustCompile("Bearer .+"),
			},
			value: "Bearer abc123",
			want:  true,
		},
		{
			name: "regex match - doesn't match",
			matcher: &HeaderMatcher{
				IsRegex: true,
				Regex:   regexp.MustCompile("Bearer .+"),
			},
			value: "Basic abc123",
			want:  false,
		},
		{
			name: "regex match - case sensitive",
			matcher: &HeaderMatcher{
				IsRegex: true,
				Regex:   regexp.MustCompile("bearer .+"),
			},
			value: "Bearer abc123",
			want:  false,
		},
		{
			name: "regex match - case insensitive regex",
			matcher: &HeaderMatcher{
				IsRegex: true,
				Regex:   regexp.MustCompile("(?i)bearer .+"),
			},
			value: "Bearer abc123",
			want:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			route := &Route{} // We just need a route instance for the method
			got := route.matchHeaderValue(tt.value, tt.matcher)
			if got != tt.want {
				t.Errorf("Route.matchHeaderValue() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetHeaderIgnoreCase(t *testing.T) {
	tests := []struct {
		name       string
		headers    map[string]string
		searchName string
		want       string
	}{
		{
			name:       "exact case match",
			headers:    map[string]string{"Content-Type": "application/json"},
			searchName: "Content-Type",
			want:       "application/json",
		},
		{
			name:       "different case match",
			headers:    map[string]string{"Content-Type": "application/json"},
			searchName: "content-type",
			want:       "application/json",
		},
		{
			name:       "uppercase search",
			headers:    map[string]string{"content-type": "application/json"},
			searchName: "CONTENT-TYPE",
			want:       "application/json",
		},
		{
			name:       "mixed case",
			headers:    map[string]string{"CoNtEnT-tYpE": "application/json"},
			searchName: "content-TYPE",
			want:       "application/json",
		},
		{
			name:       "header not found",
			headers:    map[string]string{"Content-Type": "application/json"},
			searchName: "Authorization",
			want:       "",
		},
		{
			name:       "empty headers",
			headers:    map[string]string{},
			searchName: "Content-Type",
			want:       "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &http.Request{Header: make(http.Header)}

			// Set headers on request
			for name, value := range tt.headers {
				req.Header.Set(name, value)
			}

			got := getHeaderIgnoreCase(req, tt.searchName)
			if got != tt.want {
				t.Errorf("getHeaderIgnoreCase() = %v, want %v", got, tt.want)
			}
		})
	}
}
