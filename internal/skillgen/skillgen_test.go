package skillgen

import (
	"testing"
)

func TestNewGenerator(t *testing.T) {
	tests := []struct {
		name   string
		format string
	}{
		{
			name:   "claude format",
			format: FormatClaude,
		},
		{
			name:   "codex format",
			format: FormatCodex,
		},
		{
			name:   "both format",
			format: FormatBoth,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := New(tt.format)
			if g == nil {
				t.Error("New() should return non-nil generator")
			}
		})
	}
}
