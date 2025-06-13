package template_test

import (
	"testing"

	"github.com/networkteam/shry/template"
)

func TestVariablePattern(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "simple variable",
			input:    "{{variable}}",
			expected: []string{"variable"},
		},
		{
			name:     "variable with spaces",
			input:    "{{ variable }}",
			expected: []string{"variable"},
		},
		{
			name:     "variable with tabs",
			input:    "{{\tvariable\t}}",
			expected: []string{"variable"},
		},
		{
			name:     "variable with mixed whitespace",
			input:    "{{ \t variable \t }}",
			expected: []string{"variable"},
		},
		{
			name:     "multiple variables",
			input:    "{{var1}} and {{ var2 }}",
			expected: []string{"var1", "var2"},
		},
		{
			name:     "variable with hyphens and underscores",
			input:    "{{my-variable_123}}",
			expected: []string{"my-variable_123"},
		},
		{
			name:     "no variables",
			input:    "no variables here",
			expected: nil,
		},
		{
			name:     "invalid variable with newline",
			input:    "{{var\niable}}",
			expected: nil,
		},
		{
			name:     "invalid variable with newline",
			input:    "{{\nvariable}}",
			expected: nil,
		},
		{
			name:     "invalid variable with special chars",
			input:    "{{var.iable}}",
			expected: nil,
		},
		{
			name:     "invalid variable with spaces in name",
			input:    "{{var iable}}",
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := template.FindVariables(tt.input)
			if len(got) != len(tt.expected) {
				t.Errorf("FindVariables() = %v, want %v", got, tt.expected)
				return
			}
			for i, v := range got {
				if v != tt.expected[i] {
					t.Errorf("FindVariables()[%d] = %v, want %v", i, v, tt.expected[i])
				}
			}
		})
	}
}

func TestResolve(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		variables   map[string]any
		expected    string
		expectError bool
	}{
		{
			name:  "simple variable",
			input: "Hello {{name}}!",
			variables: map[string]any{
				"name": "World",
			},
			expected: "Hello World!",
		},
		{
			name:  "multiple variables",
			input: "{{greeting}} {{name}}!",
			variables: map[string]any{
				"greeting": "Hello",
				"name":     "World",
			},
			expected: "Hello World!",
		},
		{
			name:  "variable with whitespace",
			input: "Hello {{ name }}!",
			variables: map[string]any{
				"name": "World",
			},
			expected: "Hello World!",
		},
		{
			name:        "undefined variable",
			input:       "Hello {{name}}!",
			variables:   map[string]any{},
			expectError: true,
		},
		{
			name:      "no variables",
			input:     "Hello World!",
			variables: map[string]any{},
			expected:  "Hello World!",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := template.Resolve(tt.input, tt.variables)
			if tt.expectError {
				if err == nil {
					t.Error("Resolve() expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("Resolve() unexpected error: %v", err)
				return
			}
			if got != tt.expected {
				t.Errorf("Resolve() = %v, want %v", got, tt.expected)
			}
		})
	}
}
