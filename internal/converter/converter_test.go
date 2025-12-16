package converter

import (
	"testing"
)

func TestNewConverter(t *testing.T) {
	c := New()
	if c == nil {
		t.Error("New() should return non-nil converter")
	}
}

func TestPostProcessMarkdown(t *testing.T) {
	tests := []struct {
		name     string
		markdown string
	}{
		{
			name:     "simple markdown",
			markdown: "# Hello\nWorld",
		},
		{
			name:     "empty markdown",
			markdown: "",
		},
	}

	c := New()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := c.postProcessMarkdown(tt.markdown)
			if result == "" && tt.markdown != "" {
				t.Errorf("postProcessMarkdown() returned empty for non-empty input")
			}
		})
	}
}
