package resources

import (
	"testing"

	"k8s.io/apimachinery/pkg/api/resource"
)

func TestParseQuantity(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		defaultValue string
		expected     resource.Quantity
	}{
		{
			name:         "Valid quantity",
			input:        "100Mi",
			defaultValue: "100Mi",
			expected:     resource.MustParse("100Mi"),
		},
		{
			name:         "Invalid quantity, falling back to default",
			input:        "invalid",
			defaultValue: "100Mi",
			expected:     resource.MustParse("100Mi"),
		},
		{
			name:         "Empty input, falling back to default",
			input:        "",
			defaultValue: "100Mi",
			expected:     resource.MustParse("100Mi"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseQuantity(tt.input, tt.defaultValue)
			if !result.Equal(tt.expected) {
				t.Errorf("ParseQuantity(%q, %q) = %v; want %v", tt.input, tt.defaultValue, result, tt.expected)
			}
		})
	}
}
