package core_invest

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseNumeric(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected float64
	}{
		{"float64", float64(123.45), 123.45},
		{"float32", float32(45.5), 45.5},
		{"int", int(100), 100.0},
		{"int32", int32(42), 42.0},
		{"int64", int64(9999), 9999.0},
		{"string returns 0", "123.45", 0.0},
		{"nil returns 0", nil, 0.0},
		{"bool returns 0", true, 0.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseNumeric(tt.input)
			assert.Equal(t, tt.expected, got)
		})
	}
}
