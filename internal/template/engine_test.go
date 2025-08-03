package template

import (
	"bytes"
	"errors"
	"html/template"
	"net/http"
	"net/url"
	"os"
	"strings"
	"testing"
)

func TestEngine_NewEngine(t *testing.T) {
	engine := NewEngine()
	if engine == nil {
		t.Error("NewEngine() should not return nil")
	}

	funcMap := engine.GetFuncMap()
	if funcMap == nil {
		t.Error("NewEngine() should create engine with non-nil function map")
	}

	// Verify expected custom functions are present
	expectedFuncs := []string{"trimPrefix", "header", "query", "jsonBody"}
	for _, funcName := range expectedFuncs {
		if _, exists := funcMap[funcName]; !exists {
			t.Errorf("NewEngine() function map missing expected function: %s", funcName)
		}
	}

	// Verify sprig functions are present (test a few common ones)
	sprigFuncs := []string{"upper", "lower", "trim", "split"}
	for _, funcName := range sprigFuncs {
		if _, exists := funcMap[funcName]; !exists {
			t.Errorf("NewEngine() function map missing expected sprig function: %s", funcName)
		}
	}
}

func TestEngine_CustomTrimPrefix(t *testing.T) {
	tests := []struct {
		name   string
		prefix string
		input  string
		want   string
	}{
		{
			name:   "simple prefix removal",
			prefix: "test_",
			input:  "test_value",
			want:   "value",
		},
		{
			name:   "no prefix match",
			prefix: "abc_",
			input:  "xyz_value",
			want:   "xyz_value",
		},
		{
			name:   "empty prefix",
			prefix: "",
			input:  "value",
			want:   "value",
		},
		{
			name:   "empty input",
			prefix: "test_",
			input:  "",
			want:   "",
		},
		{
			name:   "prefix longer than input",
			prefix: "verylongprefix_",
			input:  "short",
			want:   "short",
		},
		{
			name:   "exact match",
			prefix: "exact",
			input:  "exact",
			want:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := customTrimPrefix(tt.prefix, tt.input); got != tt.want {
				t.Errorf("customTrimPrefix() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEngine_CustomHeader(t *testing.T) {
	tests := []struct {
		name      string
		headerKey string
		headers   http.Header
		want      string
	}{
		{
			name:      "existing header",
			headerKey: "X-User-Id",
			headers: http.Header{
				"X-User-Id": []string{"12345"},
			},
			want: "12345",
		},
		{
			name:      "case insensitive header",
			headerKey: "content-type",
			headers: http.Header{
				"Content-Type": []string{"application/json"},
			},
			want: "application/json",
		},
		{
			name:      "multiple header values",
			headerKey: "Accept",
			headers: http.Header{
				"Accept": []string{"text/html", "application/xml"},
			},
			want: "text/html",
		},
		{
			name:      "non-existent header",
			headerKey: "X-Missing",
			headers:   http.Header{},
			want:      "",
		},
		{
			name:      "empty header value",
			headerKey: "X-Empty",
			headers: http.Header{
				"X-Empty": []string{""},
			},
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &http.Request{
				Header: tt.headers,
			}

			if got := customHeader(tt.headerKey, req); got != tt.want {
				t.Errorf("customHeader() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEngine_CustomHeader_NilRequest(t *testing.T) {
	result := customHeader("X-Test", nil)
	if result != "" {
		t.Errorf("customHeader() with nil request = %v, want empty string", result)
	}
}

func TestEngine_CustomQuery(t *testing.T) {
	tests := []struct {
		name      string
		queryKey  string
		queryVals url.Values
		want      string
	}{
		{
			name:     "existing query param",
			queryKey: "debug",
			queryVals: url.Values{
				"debug": []string{"true"},
			},
			want: "true",
		},
		{
			name:     "multiple query values",
			queryKey: "tags",
			queryVals: url.Values{
				"tags": []string{"golang", "testing"},
			},
			want: "golang",
		},
		{
			name:      "non-existent query param",
			queryKey:  "missing",
			queryVals: url.Values{},
			want:      "",
		},
		{
			name:     "empty query value",
			queryKey: "empty",
			queryVals: url.Values{
				"empty": []string{""},
			},
			want: "",
		},
		{
			name:     "query param with special characters",
			queryKey: "search",
			queryVals: url.Values{
				"search": []string{"hello world & more"},
			},
			want: "hello world & more",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &http.Request{
				URL: &url.URL{
					RawQuery: tt.queryVals.Encode(),
				},
			}
			req.URL.RawQuery = tt.queryVals.Encode()

			if got := customQuery(tt.queryKey, req); got != tt.want {
				t.Errorf("customQuery() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEngine_CustomQuery_NilRequest(t *testing.T) {
	result := customQuery("test", nil)
	if result != "" {
		t.Errorf("customQuery() with nil request = %v, want empty string", result)
	}
}

func TestEngine_CustomQuery_NilURL(t *testing.T) {
	req := &http.Request{URL: nil}
	result := customQuery("test", req)
	if result != "" {
		t.Errorf("customQuery() with nil URL = %v, want empty string", result)
	}
}

func TestEngine_CustomJsonBody(t *testing.T) {
	// This function is a placeholder that returns nil
	// Real JSON body parsing is handled in the context building
	result := customJsonBody()
	if result != nil {
		t.Errorf("customJsonBody() = %v, want nil", result)
	}
}

func TestEngine_CompileInlineTemplate(t *testing.T) {
	engine := NewEngine()

	tests := []struct {
		name     string
		tmplName string
		content  string
		wantErr  bool
		errMsg   string
	}{
		{
			name:     "simple valid template",
			tmplName: "test1",
			content:  "Hello {{.Name}}",
			wantErr:  false,
		},
		{
			name:     "template with custom functions",
			tmplName: "test2",
			content:  "{{trimPrefix \"prefix_\" \"prefix_value\"}}",
			wantErr:  false,
		},
		{
			name:     "template with sprig functions",
			tmplName: "test3",
			content:  "{{upper \"hello\"}}",
			wantErr:  false,
		},
		{
			name:     "empty template name",
			tmplName: "",
			content:  "Hello World",
			wantErr:  true,
			errMsg:   "template name cannot be empty",
		},
		{
			name:     "whitespace template name",
			tmplName: "   ",
			content:  "Hello World",
			wantErr:  true,
			errMsg:   "template name cannot be empty",
		},
		{
			name:     "empty content",
			tmplName: "test4",
			content:  "",
			wantErr:  true,
			errMsg:   "template content cannot be empty",
		},
		{
			name:     "whitespace content",
			tmplName: "test5",
			content:  "   ",
			wantErr:  true,
			errMsg:   "template content cannot be empty",
		},
		{
			name:     "invalid template syntax",
			tmplName: "test6",
			content:  "{{.InvalidSyntax",
			wantErr:  true,
			errMsg:   "failed to parse template",
		},
		{
			name:     "undefined function",
			tmplName: "test7",
			content:  "{{undefinedFunc}}",
			wantErr:  true,
			errMsg:   "failed to parse template",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpl, err := engine.CompileInlineTemplate(tt.tmplName, tt.content)
			if (err != nil) != tt.wantErr {
				t.Errorf("CompileInlineTemplate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				if tmpl != nil {
					t.Error("CompileInlineTemplate() should return nil template on error")
				}
				var compErr *CompilationError
				if !errors.As(err, &compErr) {
					t.Errorf("CompileInlineTemplate() error should be CompilationError, got %T", err)
				}
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("CompileInlineTemplate() error = %v, want error containing %q", err, tt.errMsg)
				}
			} else {
				if tmpl == nil {
					t.Error("CompileInlineTemplate() should return non-nil template on success")
				}
				if tmpl.Name() != tt.tmplName {
					t.Errorf("CompileInlineTemplate() template name = %v, want %v", tmpl.Name(), tt.tmplName)
				}
			}
		})
	}
}

func TestEngine_CompileFileTemplate(t *testing.T) {
	engine := NewEngine()

	tests := []struct {
		name    string
		content string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "simple valid template file",
			content: "Hello {{.Name}}",
			wantErr: false,
		},
		{
			name:    "template file with custom functions",
			content: "{{trimPrefix \"prefix_\" \"prefix_value\"}}",
			wantErr: false,
		},
		{
			name:    "template file with sprig functions",
			content: "{{upper \"hello\"}}",
			wantErr: false,
		},
		{
			name:    "invalid template syntax in file",
			content: "{{.InvalidSyntax",
			wantErr: true,
			errMsg:  "failed to parse template file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var tmpFile string
			if !tt.wantErr || tt.content != "" {
				tmpFile = createTempFile(t, tt.content)
				defer removeFile(tmpFile)
			}

			tmpl, err := engine.CompileFileTemplate(tmpFile)
			if (err != nil) != tt.wantErr {
				t.Errorf("CompileFileTemplate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				if tmpl != nil {
					t.Error("CompileFileTemplate() should return nil template on error")
				}
				var compErr *CompilationError
				if !errors.As(err, &compErr) {
					t.Errorf("CompileFileTemplate() error should be CompilationError, got %T", err)
				}
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("CompileFileTemplate() error = %v, want error containing %q", err, tt.errMsg)
				}
			} else {
				if tmpl == nil {
					t.Error("CompileFileTemplate() should return non-nil template on success")
				}
			}
		})
	}
}

func TestEngine_CompileFileTemplate_ErrorCases(t *testing.T) {
	engine := NewEngine()

	tests := []struct {
		name     string
		filename string
		errMsg   string
	}{
		{
			name:     "empty filename",
			filename: "",
			errMsg:   "filename cannot be empty",
		},
		{
			name:     "whitespace filename",
			filename: "   ",
			errMsg:   "filename cannot be empty",
		},
		{
			name:     "non-existent file",
			filename: "/nonexistent/file.tmpl",
			errMsg:   "failed to parse template file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpl, err := engine.CompileFileTemplate(tt.filename)
			if err == nil {
				t.Error("CompileFileTemplate() expected error but got none")
				return
			}

			if tmpl != nil {
				t.Error("CompileFileTemplate() should return nil template on error")
			}

			if !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("CompileFileTemplate() error = %v, want error containing %q", err, tt.errMsg)
			}
		})
	}
}

func TestEngine_ExecuteTemplate(t *testing.T) {
	engine := NewEngine()

	// Create a simple template
	tmpl, err := engine.CompileInlineTemplate("test", "Hello {{.Headers.Name}}")
	if err != nil {
		t.Fatalf("Failed to compile template: %v", err)
	}

	tests := []struct {
		name    string
		tmpl    *template.Template
		ctx     *TemplateContext
		wantErr bool
		errMsg  string
		wantOut string
	}{
		{
			name: "successful execution",
			tmpl: tmpl,
			ctx: &TemplateContext{
				Headers: map[string]string{"Name": "World"},
				Query:   make(map[string]string),
				Params:  make(map[string]string),
			},
			wantErr: false,
			wantOut: "Hello World",
		},
		{
			name:    "nil template",
			tmpl:    nil,
			ctx:     &TemplateContext{},
			wantErr: true,
			errMsg:  "template is nil",
		},
		{
			name:    "nil context",
			tmpl:    tmpl,
			ctx:     nil,
			wantErr: true,
			errMsg:  "context is nil",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			err := engine.ExecuteTemplate(tt.tmpl, &buf, tt.ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("ExecuteTemplate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				var execErr *ExecutionError
				if !errors.As(err, &execErr) {
					t.Errorf("ExecuteTemplate() error should be ExecutionError, got %T", err)
				}
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("ExecuteTemplate() error = %v, want error containing %q", err, tt.errMsg)
				}
			} else {
				if tt.wantOut != "" && buf.String() != tt.wantOut {
					t.Errorf("ExecuteTemplate() output = %v, want %v", buf.String(), tt.wantOut)
				}
			}
		})
	}
}

func TestEngine_ExecuteTemplate_NilWriter(t *testing.T) {
	engine := NewEngine()
	tmpl, err := engine.CompileInlineTemplate("test", "Hello World")
	if err != nil {
		t.Fatalf("Failed to compile template: %v", err)
	}

	ctx := &TemplateContext{
		Headers: make(map[string]string),
		Query:   make(map[string]string),
		Params:  make(map[string]string),
	}

	err = engine.ExecuteTemplate(tmpl, nil, ctx)
	if err == nil {
		t.Error("ExecuteTemplate() expected error with nil writer but got none")
		return
	}

	if !strings.Contains(err.Error(), "writer is nil") {
		t.Errorf("ExecuteTemplate() error = %v, want error containing 'writer is nil'", err)
	}
}

func TestEngine_BuildTemplateContext(t *testing.T) {
	engine := NewEngine()

	// Create a mock request
	req, err := http.NewRequest("GET", "/test?debug=true", strings.NewReader(`{"name":"test"}`))
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", "12345")

	params := map[string]string{
		"id":   "123",
		"name": "test",
	}

	ctx, err := engine.BuildTemplateContext(req, params)
	if err != nil {
		t.Errorf("BuildTemplateContext() error = %v, expected no error", err)
		return
	}

	if ctx == nil {
		t.Error("BuildTemplateContext() should return non-nil context")
		return
	}

	// Verify request is set
	if ctx.Request != req {
		t.Error("BuildTemplateContext() context Request should match input request")
	}

	// Verify headers are extracted
	if ctx.Headers["X-User-Id"] != "12345" {
		t.Errorf("BuildTemplateContext() context Headers[X-User-Id] = %q, want 12345", ctx.Headers["X-User-Id"])
	}

	// Verify query params are extracted
	if ctx.Query["debug"] != "true" {
		t.Errorf("BuildTemplateContext() context Query[debug] = %v, want true", ctx.Query["debug"])
	}

	// Verify params are set
	if ctx.Params["id"] != "123" {
		t.Errorf("BuildTemplateContext() context Params[id] = %v, want 123", ctx.Params["id"])
	}
}

func TestEngine_BuildTemplateContext_NilRequest(t *testing.T) {
	engine := NewEngine()

	ctx, err := engine.BuildTemplateContext(nil, nil)
	if err == nil {
		t.Error("BuildTemplateContext() expected error with nil request but got none")
		return
	}

	if ctx != nil {
		t.Error("BuildTemplateContext() should return nil context on error")
	}

	var ctxErr *ContextError
	if !errors.As(err, &ctxErr) {
		t.Errorf("BuildTemplateContext() error should be ContextError, got %T", err)
	}
}

func TestEngine_CompileAndExecute(t *testing.T) {
	engine := NewEngine()

	// Create a mock request
	req, err := http.NewRequest("GET", "/test?name=World", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	params := map[string]string{"greeting": "Hello"}

	tests := []struct {
		name            string
		templateSource  string
		templateContent string
		wantErr         bool
		wantContains    string
	}{
		{
			name:            "inline template execution",
			templateSource:  "greeting_template",
			templateContent: "{{.Params.greeting}} {{.Query.name}}!",
			wantErr:         false,
			wantContains:    "Hello World!",
		},
		{
			name:            "template with custom function",
			templateSource:  "custom_func_template",
			templateContent: `{{trimPrefix "test_" "test_value"}}`,
			wantErr:         false,
			wantContains:    "value",
		},
		{
			name:            "invalid template syntax",
			templateSource:  "invalid_template",
			templateContent: "{{.InvalidSyntax",
			wantErr:         true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			err := engine.CompileAndExecute(tt.templateSource, tt.templateContent, &buf, req, params)
			if (err != nil) != tt.wantErr {
				t.Errorf("CompileAndExecute() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && tt.wantContains != "" {
				if !strings.Contains(buf.String(), tt.wantContains) {
					t.Errorf("CompileAndExecute() output = %v, want containing %v", buf.String(), tt.wantContains)
				}
			}
		})
	}
}

func TestEngine_GetFuncMap(t *testing.T) {
	engine := NewEngine()
	funcMap := engine.GetFuncMap()

	if funcMap == nil {
		t.Error("GetFuncMap() should not return nil")
		return
	}

	// Verify that modifying the returned map doesn't affect the engine
	originalLen := len(funcMap)
	funcMap["testFunc"] = func() {}

	// Get the map again and verify it wasn't modified
	funcMap2 := engine.GetFuncMap()
	if len(funcMap2) != originalLen {
		t.Error("GetFuncMap() should return a copy that can't be modified")
	}

	if _, exists := funcMap2["testFunc"]; exists {
		t.Error("GetFuncMap() should return a copy that protects the original function map")
	}
}

// Benchmark tests for performance
func BenchmarkEngine_CompileInlineTemplate(b *testing.B) {
	engine := NewEngine()
	templateContent := "Hello {{.Name}}, your ID is {{.Params.id}}"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := engine.CompileInlineTemplate("benchmark", templateContent)
		if err != nil {
			b.Fatalf("CompileInlineTemplate() error = %v", err)
		}
	}
}

func BenchmarkEngine_ExecuteTemplate(b *testing.B) {
	engine := NewEngine()
	tmpl, err := engine.CompileInlineTemplate("benchmark", "Hello {{.Query.name}}")
	if err != nil {
		b.Fatalf("Failed to compile template: %v", err)
	}

	ctx := &TemplateContext{
		Headers: make(map[string]string),
		Query:   map[string]string{"name": "World"},
		Params:  make(map[string]string),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var buf bytes.Buffer
		err := engine.ExecuteTemplate(tmpl, &buf, ctx)
		if err != nil {
			b.Fatalf("ExecuteTemplate() error = %v", err)
		}
	}
}

// Helper functions
func createTempFile(t *testing.T, content string) string {
	tmpFile, err := os.CreateTemp("", "template_test_*.tmpl")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer tmpFile.Close()

	if _, err := tmpFile.WriteString(content); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}

	return tmpFile.Name()
}

func removeFile(filename string) {
	os.Remove(filename)
}
