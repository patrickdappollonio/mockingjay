package template

import (
	"strings"
	"testing"
	"time"
)

func TestTrimPrefix(t *testing.T) {
	tests := []struct {
		name     string
		prefix   string
		input    string
		expected string
	}{
		{
			name:     "removes prefix",
			prefix:   "hello",
			input:    "hello world",
			expected: " world",
		},
		{
			name:     "no prefix to remove",
			prefix:   "foo",
			input:    "hello world",
			expected: "hello world",
		},
		{
			name:     "empty prefix",
			prefix:   "",
			input:    "hello world",
			expected: "hello world",
		},
		{
			name:     "empty input",
			prefix:   "hello",
			input:    "",
			expected: "",
		},
		{
			name:     "prefix longer than input",
			prefix:   "hello world!",
			input:    "hello",
			expected: "hello",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := trimPrefix(tt.prefix, tt.input)
			if result != tt.expected {
				t.Errorf("trimPrefix(%q, %q) = %q, want %q", tt.prefix, tt.input, result, tt.expected)
			}
		})
	}
}

func TestSleep(t *testing.T) {
	tests := []struct {
		name        string
		duration    interface{}
		expectDelay bool
		minDuration time.Duration
		maxDuration time.Duration
	}{
		{
			name:        "string duration - milliseconds",
			duration:    "50ms",
			expectDelay: true,
			minDuration: 45 * time.Millisecond,
			maxDuration: 100 * time.Millisecond,
		},
		{
			name:        "string duration - seconds",
			duration:    "1s",
			expectDelay: true,
			minDuration: 950 * time.Millisecond,
			maxDuration: 1100 * time.Millisecond,
		},
		{
			name:        "int duration - seconds",
			duration:    1,
			expectDelay: true,
			minDuration: 950 * time.Millisecond,
			maxDuration: 1100 * time.Millisecond,
		},
		{
			name:        "float duration - milliseconds",
			duration:    0.1, // 100ms
			expectDelay: true,
			minDuration: 95 * time.Millisecond,
			maxDuration: 150 * time.Millisecond,
		},
		{
			name:        "invalid string duration",
			duration:    "invalid",
			expectDelay: false,
			minDuration: 0,
			maxDuration: 10 * time.Millisecond,
		},
		{
			name:        "zero duration",
			duration:    0,
			expectDelay: false,
			minDuration: 0,
			maxDuration: 10 * time.Millisecond,
		},
		{
			name:        "negative duration",
			duration:    -1,
			expectDelay: false,
			minDuration: 0,
			maxDuration: 10 * time.Millisecond,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start := time.Now()
			result := sleep(tt.duration)
			elapsed := time.Since(start)

			// Check that the function returns an empty string
			if result != "" {
				t.Errorf("sleep(%v) = %q, want empty string", tt.duration, result)
			}

			// Check timing
			if tt.expectDelay {
				if elapsed < tt.minDuration {
					t.Errorf("sleep(%v) took %v, expected at least %v", tt.duration, elapsed, tt.minDuration)
				}
				if elapsed > tt.maxDuration {
					t.Errorf("sleep(%v) took %v, expected at most %v", tt.duration, elapsed, tt.maxDuration)
				}
			} else {
				if elapsed > tt.maxDuration {
					t.Errorf("sleep(%v) took %v, expected at most %v for non-delay case", tt.duration, elapsed, tt.maxDuration)
				}
			}
		})
	}
}

func TestRandFloat(t *testing.T) {
	tests := []struct {
		name     string
		min      interface{}
		max      interface{}
		testFunc func(t *testing.T, result float64)
	}{
		{
			name: "basic range 0.0 to 1.0",
			min:  0.0,
			max:  1.0,
			testFunc: func(t *testing.T, result float64) {
				if result < 0.0 || result > 1.0 {
					t.Errorf("randFloat(0.0, 1.0) = %f, expected range [0.0, 1.0]", result)
				}
			},
		},
		{
			name: "integer inputs",
			min:  1,
			max:  10,
			testFunc: func(t *testing.T, result float64) {
				if result < 1.0 || result > 10.0 {
					t.Errorf("randFloat(1, 10) = %f, expected range [1.0, 10.0]", result)
				}
			},
		},
		{
			name: "negative range",
			min:  -5.5,
			max:  -1.1,
			testFunc: func(t *testing.T, result float64) {
				if result < -5.5 || result > -1.1 {
					t.Errorf("randFloat(-5.5, -1.1) = %f, expected range [-5.5, -1.1]", result)
				}
			},
		},
		{
			name: "range crossing zero",
			min:  -2.5,
			max:  3.7,
			testFunc: func(t *testing.T, result float64) {
				if result < -2.5 || result > 3.7 {
					t.Errorf("randFloat(-2.5, 3.7) = %f, expected range [-2.5, 3.7]", result)
				}
			},
		},
		{
			name: "reversed parameters (max < min)",
			min:  10.0,
			max:  1.0,
			testFunc: func(t *testing.T, result float64) {
				// Function should swap them, so result should be in [1.0, 10.0]
				if result < 1.0 || result > 10.0 {
					t.Errorf("randFloat(10.0, 1.0) = %f, expected range [1.0, 10.0] (auto-swapped)", result)
				}
			},
		},
		{
			name: "same min and max",
			min:  5.5,
			max:  5.5,
			testFunc: func(t *testing.T, result float64) {
				if result != 5.5 {
					t.Errorf("randFloat(5.5, 5.5) = %f, expected exactly 5.5", result)
				}
			},
		},
		{
			name: "large range",
			min:  0.0,
			max:  1000000.0,
			testFunc: func(t *testing.T, result float64) {
				if result < 0.0 || result > 1000000.0 {
					t.Errorf("randFloat(0.0, 1000000.0) = %f, expected range [0.0, 1000000.0]", result)
				}
			},
		},
		{
			name: "mixed template types",
			min:  2,   // int from template
			max:  8.5, // float64 from template
			testFunc: func(t *testing.T, result float64) {
				if result < 2.0 || result > 8.5 {
					t.Errorf("randFloat(2, 8.5) = %f, expected range [2.0, 8.5]", result)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Run the function multiple times to test randomness
			for i := 0; i < 10; i++ {
				result := randFloat(tt.min, tt.max)
				tt.testFunc(t, result)
			}
		})
	}
}

func TestRandFloatDistribution(t *testing.T) {
	// Test that randFloat produces reasonable distribution
	min, max := 0.0, 10.0
	count := 1000
	results := make([]float64, count)

	for i := 0; i < count; i++ {
		results[i] = randFloat(min, max)
	}

	// Calculate basic statistics
	var sum float64
	for _, v := range results {
		sum += v
	}
	average := sum / float64(count)

	// Expected average should be around the middle of the range (5.0)
	expectedAverage := (min + max) / 2
	tolerance := 0.5 // Allow some variance due to randomness

	if average < expectedAverage-tolerance || average > expectedAverage+tolerance {
		t.Errorf("randFloat distribution average = %f, expected around %f (±%f)", average, expectedAverage, tolerance)
	}

	// Check that we got values across the range
	hasLow := false
	hasHigh := false
	for _, v := range results {
		if v < min+1.0 {
			hasLow = true
		}
		if v > max-1.0 {
			hasHigh = true
		}
	}

	if !hasLow {
		t.Error("randFloat should produce values in the lower part of the range")
	}
	if !hasHigh {
		t.Error("randFloat should produce values in the upper part of the range")
	}
}

func TestToFloat64(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected float64
	}{
		// Template-compatible types (what Go templates actually parse)
		{"float64", float64(5.5), 5.5},
		{"float64 from literal", 3.14159, 3.14159},
		{"int", int(42), 42.0},
		{"int from literal", 10, 10.0},

		// Unsupported types should return 0
		{"string", "invalid", 0.0},
		{"nil", nil, 0.0},
		{"bool", true, 0.0},
		{"slice", []int{1, 2, 3}, 0.0},
		{"float32", float32(3.3), 0.0}, // Not from templates
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := toFloat64(tt.input)
			// Use tolerance for floating-point comparisons
			tolerance := 1e-9
			if result < tt.expected-tolerance || result > tt.expected+tolerance {
				t.Errorf("toFloat64(%v) = %f, want %f", tt.input, result, tt.expected)
			}
		})
	}
}

func TestRandChoice(t *testing.T) {
	tests := []struct {
		name     string
		choices  []interface{}
		testFunc func(t *testing.T, result interface{})
	}{
		{
			name:    "no arguments",
			choices: []interface{}{},
			testFunc: func(t *testing.T, result interface{}) {
				if result != nil {
					t.Errorf("randChoice() = %v, expected nil", result)
				}
			},
		},
		{
			name:    "single string argument",
			choices: []interface{}{"only"},
			testFunc: func(t *testing.T, result interface{}) {
				if result != "only" {
					t.Errorf("randChoice(\"only\") = %v, expected \"only\"", result)
				}
			},
		},
		{
			name:    "string choices",
			choices: []interface{}{"red", "blue"},
			testFunc: func(t *testing.T, result interface{}) {
				if result != "red" && result != "blue" {
					t.Errorf("randChoice(\"red\", \"blue\") = %v, expected \"red\" or \"blue\"", result)
				}
			},
		},
		{
			name:    "integer choices",
			choices: []interface{}{1, 2, 3, 4, 5},
			testFunc: func(t *testing.T, result interface{}) {
				validChoices := map[int]bool{1: true, 2: true, 3: true, 4: true, 5: true}
				if intResult, ok := result.(int); !ok || !validChoices[intResult] {
					t.Errorf("randChoice(1, 2, 3, 4, 5) = %v, expected one of the provided integer choices", result)
				}
			},
		},
		{
			name:    "float choices",
			choices: []interface{}{1.1, 2.2, 3.3},
			testFunc: func(t *testing.T, result interface{}) {
				validChoices := map[float64]bool{1.1: true, 2.2: true, 3.3: true}
				if floatResult, ok := result.(float64); !ok || !validChoices[floatResult] {
					t.Errorf("randChoice(1.1, 2.2, 3.3) = %v, expected one of the provided float choices", result)
				}
			},
		},
		{
			name:    "boolean choices",
			choices: []interface{}{true, false},
			testFunc: func(t *testing.T, result interface{}) {
				if result != true && result != false {
					t.Errorf("randChoice(true, false) = %v, expected true or false", result)
				}
			},
		},
		{
			name:    "mixed type choices",
			choices: []interface{}{"text", 42, 3.14, true, false},
			testFunc: func(t *testing.T, result interface{}) {
				validChoices := map[interface{}]bool{
					"text": true,
					42:     true,
					3.14:   true,
					true:   true,
					false:  true,
				}
				if !validChoices[result] {
					t.Errorf("randChoice with mixed types = %v, expected one of the provided choices", result)
				}
			},
		},
		{
			name:    "string choices with empty strings",
			choices: []interface{}{"", "non-empty", ""},
			testFunc: func(t *testing.T, result interface{}) {
				if result != "" && result != "non-empty" {
					t.Errorf("randChoice(\"\", \"non-empty\", \"\") = %v, expected \"\" or \"non-empty\"", result)
				}
			},
		},
		{
			name:    "choices with special characters",
			choices: []interface{}{"hello world", "foo-bar", "test_123", "special!@#"},
			testFunc: func(t *testing.T, result interface{}) {
				validChoices := map[string]bool{
					"hello world": true,
					"foo-bar":     true,
					"test_123":    true,
					"special!@#":  true,
				}
				if strResult, ok := result.(string); !ok || !validChoices[strResult] {
					t.Errorf("randChoice with special characters = %v, expected one of the provided choices", result)
				}
			},
		},
		{
			name:    "nil values in choices",
			choices: []interface{}{nil, "valid", nil},
			testFunc: func(t *testing.T, result interface{}) {
				if result != nil && result != "valid" {
					t.Errorf("randChoice(nil, \"valid\", nil) = %v, expected nil or \"valid\"", result)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Run the function multiple times to test consistency (except for randomness)
			iterations := 1
			if len(tt.choices) > 1 {
				iterations = 5 // Test multiple times for random functions
			}

			for i := 0; i < iterations; i++ {
				result := randChoice(tt.choices...)
				tt.testFunc(t, result)
			}
		})
	}
}

func TestRandChoiceDistribution(t *testing.T) {
	// Test that randChoice produces reasonable distribution with strings
	choices := []interface{}{"A", "B", "C"}
	count := 300
	results := make(map[interface{}]int)

	for i := 0; i < count; i++ {
		result := randChoice(choices...)
		results[result]++
	}

	// Check that all choices were selected at least once
	for _, choice := range choices {
		if results[choice] == 0 {
			t.Errorf("randChoice never selected %v in %d iterations", choice, count)
		}
	}

	// Check that no invalid choices were returned
	for result := range results {
		found := false
		for _, choice := range choices {
			if result == choice {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("randChoice returned invalid result %v", result)
		}
	}

	// Basic distribution check - each choice should get some selections
	// With 300 iterations and 3 choices, we expect roughly 100 each
	// Allow for significant variance due to randomness
	minExpected := count / len(choices) / 3 // At least 33 per choice

	for choice, choiceCount := range results {
		if choiceCount < minExpected {
			t.Logf("Warning: Choice %v only selected %d times (expected at least %d)", choice, choiceCount, minExpected)
			// Don't fail the test for distribution issues, just log a warning
		}
	}

	t.Logf("Distribution results: %v", results)
}

func TestRandChoiceTypedDistribution(t *testing.T) {
	// Test distribution with different types
	t.Run("integer distribution", func(t *testing.T) {
		choices := []interface{}{1, 2, 3}
		count := 150
		results := make(map[interface{}]int)

		for i := 0; i < count; i++ {
			result := randChoice(choices...)
			results[result]++
		}

		// Check that all integer choices were selected
		for _, choice := range choices {
			if results[choice] == 0 {
				t.Errorf("randChoice never selected integer %v in %d iterations", choice, count)
			}
		}

		t.Logf("Integer distribution: %v", results)
	})

	t.Run("mixed type distribution", func(t *testing.T) {
		choices := []interface{}{"text", 42, 3.14, true}
		count := 200
		results := make(map[interface{}]int)

		for i := 0; i < count; i++ {
			result := randChoice(choices...)
			results[result]++
		}

		// Check that all mixed choices were selected
		for _, choice := range choices {
			if results[choice] == 0 {
				t.Errorf("randChoice never selected mixed type %v in %d iterations", choice, count)
			}
		}

		t.Logf("Mixed type distribution: %v", results)
	})
}

func TestToJsonPretty(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected string
	}{
		{
			name:  "simple object",
			input: map[string]string{"name": "John", "city": "NYC"},
			expected: `{
  "city": "NYC",
  "name": "John"
}`,
		},
		{
			name:  "nested object",
			input: map[string]interface{}{"user": map[string]string{"name": "Jane"}, "active": true},
			expected: `{
  "active": true,
  "user": {
    "name": "Jane"
  }
}`,
		},
		{
			name:     "array",
			input:    []string{"apple", "banana", "cherry"},
			expected: `[\n  "apple",\n  "banana",\n  "cherry"\n]`,
		},
		{
			name:     "empty object",
			input:    map[string]string{},
			expected: "{}",
		},
		{
			name:     "empty array",
			input:    []string{},
			expected: "[]",
		},
		{
			name:     "nil input",
			input:    nil,
			expected: "null",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := toJsonPretty(tt.input)

			// Normalize line endings and whitespace for comparison
			expectedNormalized := strings.ReplaceAll(tt.expected, "\\n", "\n")

			if result != expectedNormalized {
				t.Errorf("toJsonPretty() = %q, want %q", result, expectedNormalized)
			}

			// Verify it contains proper indentation
			if strings.Contains(expectedNormalized, "{") && !strings.Contains(result, "  ") && len(result) > 10 {
				t.Errorf("toJsonPretty() should contain proper indentation, got %q", result)
			}
		})
	}
}

func TestToJsonPrettyError(t *testing.T) {
	// Test with a value that can't be marshaled (channels can't be marshaled to JSON)
	ch := make(chan int)
	result := toJsonPretty(ch)
	if result != "{}" {
		t.Errorf("toJsonPretty() with unmarshalable input = %q, want %q", result, "{}")
	}
}
