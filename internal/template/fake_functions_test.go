package template

import (
	"strings"
	"testing"
	"time"
)

func TestFakeFunctions(t *testing.T) {
	tests := []struct {
		name      string
		funcCall  func() any
		validator func(any) bool
	}{
		{
			name:     "fakeName returns non-empty string",
			funcCall: func() any { return fakeName() },
			validator: func(v any) bool {
				s, ok := v.(string)
				return ok && len(s) > 0
			},
		},
		{
			name:     "fakeEmail returns valid email format",
			funcCall: func() any { return fakeEmail() },
			validator: func(v any) bool {
				s, ok := v.(string)
				return ok && strings.Contains(s, "@") && strings.Contains(s, ".")
			},
		},
		{
			name:     "fakePhone returns non-empty string",
			funcCall: func() any { return fakePhone() },
			validator: func(v any) bool {
				s, ok := v.(string)
				return ok && len(s) > 0
			},
		},
		{
			name:     "fakeCompany returns non-empty string",
			funcCall: func() any { return fakeCompany() },
			validator: func(v any) bool {
				s, ok := v.(string)
				return ok && len(s) > 0
			},
		},
		{
			name:     "fakeJobTitle returns non-empty string",
			funcCall: func() any { return fakeJobTitle() },
			validator: func(v any) bool {
				s, ok := v.(string)
				return ok && len(s) > 0
			},
		},
		{
			name:     "fakeCreditCardNumber returns non-empty string",
			funcCall: func() any { return fakeCreditCardNumber() },
			validator: func(v any) bool {
				s, ok := v.(string)
				return ok && len(s) > 0
			},
		},
		{
			name:     "fakeColor returns non-empty string",
			funcCall: func() any { return fakeColor() },
			validator: func(v any) bool {
				s, ok := v.(string)
				return ok && len(s) > 0
			},
		},
		{
			name:     "fakeUUID returns valid UUID format",
			funcCall: func() any { return fakeUUID() },
			validator: func(v any) bool {
				s, ok := v.(string)
				return ok && len(s) == 36 && strings.Count(s, "-") == 4
			},
		},
		{
			name:     "fakeDate returns valid time",
			funcCall: func() any { return fakeDate() },
			validator: func(v any) bool {
				_, ok := v.(time.Time)
				return ok
			},
		},
		{
			name:     "fakeMonth returns valid month number",
			funcCall: func() any { return fakeMonth() },
			validator: func(v any) bool {
				m, ok := v.(int)
				return ok && m >= 1 && m <= 12
			},
		},
		{
			name:     "fakeYear returns reasonable year",
			funcCall: func() any { return fakeYear() },
			validator: func(v any) bool {
				y, ok := v.(int)
				return ok && y >= 1900 && y <= 2100
			},
		},
		{
			name:     "fakeRandomBool returns boolean",
			funcCall: func() any { return fakeRandomBool() },
			validator: func(v any) bool {
				_, ok := v.(bool)
				return ok
			},
		},
		{
			name:     "fakeWords generates requested number of words",
			funcCall: func() any { return fakeWords(3) },
			validator: func(v any) bool {
				s, ok := v.(string)
				return ok && len(strings.Fields(s)) == 3
			},
		},
		{
			name:     "fakePrice generates price in range",
			funcCall: func() any { return fakePrice(10.0, 20.0) },
			validator: func(v any) bool {
				p, ok := v.(float64)
				return ok && p >= 10.0 && p <= 20.0
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.funcCall()
			if !tt.validator(result) {
				t.Errorf("Test %s failed: got %v", tt.name, result)
			}
		})
	}
}

func TestFakeFunctionsInTemplate(t *testing.T) {
	engine := NewEngine()

	tests := []struct {
		name     string
		template string
	}{
		{
			name:     "basic fake data",
			template: `Name: {{ fakeName }}, Email: {{ fakeEmail }}`,
		},
		{
			name:     "fake company data",
			template: `Company: {{ fakeCompany }}, Job: {{ fakeJobTitle }}`,
		},
		{
			name:     "fake financial data",
			template: `Card: {{ fakeCreditCardNumber }}, Currency: {{ fakeCurrency }}`,
		},
		{
			name:     "fake address data",
			template: `City: {{ fakeCity }}, State: {{ fakeState }}`,
		},
		{
			name:     "fake tech data",
			template: `UUID: {{ fakeUUID }}, URL: {{ fakeURL }}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpl, err := engine.CompileInlineTemplate(tt.name, tt.template)
			if err != nil {
				t.Fatalf("Failed to compile template: %v", err)
			}

			var buf strings.Builder
			ctx := &TemplateContext{} // Empty context should work for fake data functions
			err = engine.ExecuteTemplate(tmpl, &buf, ctx)
			if err != nil {
				t.Fatalf("Failed to execute template: %v", err)
			}

			result := buf.String()
			if len(result) == 0 {
				t.Errorf("Template produced empty result")
			}

			// Basic validation that template was processed (no template syntax remains)
			if strings.Contains(result, "{{") || strings.Contains(result, "}}") {
				t.Errorf("Template not fully processed: %s", result)
			}
		})
	}
}
