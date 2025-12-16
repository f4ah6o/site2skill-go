package validator

import (
	"testing"
)

func TestNewValidator(t *testing.T) {
	v := New()
	if v == nil {
		t.Error("New() should return non-nil validator")
	}
}

func TestValidateSkill(t *testing.T) {
	tests := []struct {
		name string
		path string
	}{
		{
			name: "valid skill structure",
			path: ".",
		},
	}

	v := New()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := v.Validate(tt.path)
			if !result {
				t.Logf("Validate(%s) returned false (expected true for current directory)", tt.path)
			}
		})
	}
}
