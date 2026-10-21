package reporting

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSanitizeCSVField(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "formula starting with equal",
			input:    "=1+2",
			expected: "'=1+2",
		},
		{
			name:     "formula starting with plus",
			input:    "+123456",
			expected: "'+123456",
		},
		{
			name:     "formula starting with minus",
			input:    "-100.00",
			expected: "'-100.00",
		},
		{
			name:     "formula starting with at sign",
			input:    "@SUM(A1:A10)",
			expected: "'@SUM(A1:A10)",
		},
		{
			name:     "formula starting with tab character",
			input:    "\tcmd|' /C calc'!A0",
			expected: "'\tcmd|' /C calc'!A0",
		},
		{
			name:     "formula starting with carriage return character",
			input:    "\r=1+2",
			expected: "'\r=1+2",
		},
		{
			name:     "single character equal trigger",
			input:    "=",
			expected: "'=",
		},
		{
			name:     "single character plus trigger",
			input:    "+",
			expected: "'+",
		},
		{
			name:     "single character minus trigger",
			input:    "-",
			expected: "'-",
		},
		{
			name:     "single character at trigger",
			input:    "@",
			expected: "'@",
		},
		{
			name:     "single character tab trigger",
			input:    "\t",
			expected: "'\t",
		},
		{
			name:     "single character carriage return trigger",
			input:    "\r",
			expected: "'\r",
		},
		{
			name:     "normal alphanumeric string",
			input:    "Orange Cat Investments",
			expected: "Orange Cat Investments",
		},
		{
			name:     "numeric string",
			input:    "123456.78",
			expected: "123456.78",
		},
		{
			name:     "formula character inside string (equal in middle)",
			input:    "a=b",
			expected: "a=b",
		},
		{
			name:     "formula character inside string (plus in middle)",
			input:    "1+2",
			expected: "1+2",
		},
		{
			name:     "formula character inside string (minus in middle)",
			input:    "test-run",
			expected: "test-run",
		},
		{
			name:     "formula character inside string (at in middle)",
			input:    "user@domain.com",
			expected: "user@domain.com",
		},
		{
			name:     "string starting with leading space",
			input:    " hello",
			expected: " hello",
		},
		{
			name:     "string starting with newline",
			input:    "\nhello",
			expected: "\nhello",
		},
		{
			name:     "string starting with unicode character",
			input:    "🐱 Orange Cat",
			expected: "🐱 Orange Cat",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sanitizeCSVField(tt.input)
			assert.Equal(t, tt.expected, got)
		})
	}
}
