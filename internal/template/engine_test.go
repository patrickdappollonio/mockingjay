package template

import (
	"bytes"
	"errors"
	"net/http"
	"net/url"
	"os"
	"strings"
	"testing"
	"text/template"
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
	expectedFuncs := []string{"trimPrefix", "sleep", "randFloat", "randChoice"}
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
	tmpl, err := engine.CompileInlineTemplate("test", "Hello {{.Headers.Get \"Name\"}}")
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
				Headers: http.Header{"Name": []string{"World"}},
				Query:   make(url.Values),
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
		Headers: make(http.Header),
		Query:   make(url.Values),
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
	if ctx.Headers.Get("X-User-Id") != "12345" {
		t.Errorf("BuildTemplateContext() context Headers.Get(X-User-Id) = %q, want 12345", ctx.Headers.Get("X-User-Id"))
	}

	// Verify query params are extracted
	if ctx.Query.Get("debug") != "true" {
		t.Errorf("BuildTemplateContext() context Query.Get(debug) = %v, want true", ctx.Query.Get("debug"))
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
			templateContent: "{{.Params.greeting}} {{.Query.Get \"name\"}}!",
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
	tmpl, err := engine.CompileInlineTemplate("benchmark", "Hello {{.Query.Get \"name\"}}")
	if err != nil {
		b.Fatalf("Failed to compile template: %v", err)
	}

	ctx := &TemplateContext{
		Headers: make(http.Header),
		Query:   url.Values{"name": []string{"World"}},
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

func TestEngine_NewEngineWithDelimiters(t *testing.T) {
	tests := []struct {
		name       string
		leftDelim  string
		rightDelim string
		wantLeft   string
		wantRight  string
	}{
		{
			name:       "custom delimiters",
			leftDelim:  "<%",
			rightDelim: "%>",
			wantLeft:   "<%",
			wantRight:  "%>",
		},
		{
			name:       "double percent delimiters",
			leftDelim:  "<%%",
			rightDelim: "%%>",
			wantLeft:   "<%%",
			wantRight:  "%%>",
		},
		{
			name:       "different style delimiters",
			leftDelim:  "[[[",
			rightDelim: "]]]",
			wantLeft:   "[[[",
			wantRight:  "]]]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine := NewEngineWithDelimiters(tt.leftDelim, tt.rightDelim)
			if engine == nil {
				t.Error("NewEngineWithDelimiters() should not return nil")
			}

			// Check that delimiters are stored correctly
			if engine.leftDelimiter != tt.wantLeft {
				t.Errorf("NewEngineWithDelimiters() leftDelimiter = %q, want %q", engine.leftDelimiter, tt.wantLeft)
			}
			if engine.rightDelimiter != tt.wantRight {
				t.Errorf("NewEngineWithDelimiters() rightDelimiter = %q, want %q", engine.rightDelimiter, tt.wantRight)
			}

			// Test that function map is still available
			funcMap := engine.GetFuncMap()
			if funcMap == nil {
				t.Error("NewEngineWithDelimiters() should create engine with non-nil function map")
			}
		})
	}
}

func TestEngine_CompileInlineTemplateWithCustomDelimiters(t *testing.T) {
	tests := []struct {
		name       string
		leftDelim  string
		rightDelim string
		content    string
		data       interface{}
		wantOutput string
		wantErr    bool
	}{
		{
			name:       "erb-style delimiters",
			leftDelim:  "<%",
			rightDelim: "%>",
			content:    "Hello <% .Name %>!",
			data:       map[string]string{"Name": "World"},
			wantOutput: "Hello World!",
			wantErr:    false,
		},
		{
			name:       "double percent delimiters",
			leftDelim:  "<%%",
			rightDelim: "%%>",
			content:    "Value: <%%  .Value  %%>",
			data:       map[string]int{"Value": 42},
			wantOutput: "Value: 42",
			wantErr:    false,
		},
		{
			name:       "bracket delimiters with function",
			leftDelim:  "[[[",
			rightDelim: "]]]",
			content:    "Uppercase: [[[  .Text | upper  ]]]",
			data:       map[string]string{"Text": "hello"},
			wantOutput: "Uppercase: HELLO",
			wantErr:    false,
		},
		{
			name:       "mixed delimiters should not interfere",
			leftDelim:  "<%",
			rightDelim: "%>",
			content:    "{{ not parsed }} but <% .Value %> is parsed",
			data:       map[string]string{"Value": "success"},
			wantOutput: "{{ not parsed }} but success is parsed",
			wantErr:    false,
		},
		{
			name:       "custom delimiters with conditional",
			leftDelim:  "<<",
			rightDelim: ">>",
			content:    "<< if .Show >>Visible<< end >>",
			data:       map[string]bool{"Show": true},
			wantOutput: "Visible",
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine := NewEngineWithDelimiters(tt.leftDelim, tt.rightDelim)

			tmpl, err := engine.CompileInlineTemplate("test", tt.content)
			if (err != nil) != tt.wantErr {
				t.Errorf("CompileInlineTemplate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				var buf bytes.Buffer
				err = tmpl.Execute(&buf, tt.data)
				if err != nil {
					t.Errorf("Template execution error: %v", err)
					return
				}

				if buf.String() != tt.wantOutput {
					t.Errorf("Template output = %q, want %q", buf.String(), tt.wantOutput)
				}
			}
		})
	}
}

func TestEngine_CompileFileTemplateWithCustomDelimiters(t *testing.T) {
	// Create a temporary template file with custom delimiters
	content := "Hello <% .Name %>! Your value is <% .Value %>."
	tempFile := createTempFile(t, content)
	defer removeFile(tempFile)

	engine := NewEngineWithDelimiters("<%", "%>")

	tmpl, err := engine.CompileFileTemplate(tempFile)
	if err != nil {
		t.Errorf("CompileFileTemplate() error = %v", err)
		return
	}

	// Test execution with data
	data := map[string]interface{}{
		"Name":  "Alice",
		"Value": 123,
	}

	var buf bytes.Buffer
	// For file templates, we need to execute the template by its filename (basename)
	fileName := tempFile[strings.LastIndex(tempFile, "/")+1:]
	err = tmpl.ExecuteTemplate(&buf, fileName, data)
	if err != nil {
		t.Errorf("Template execution error: %v", err)
		return
	}

	expected := "Hello Alice! Your value is 123."
	if buf.String() != expected {
		t.Errorf("Template output = %q, want %q", buf.String(), expected)
	}
}
