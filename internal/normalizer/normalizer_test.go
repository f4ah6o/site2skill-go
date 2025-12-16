package normalizer

import (
	"testing"
)

func TestNewNormalizer(t *testing.T) {
	n := New()
	if n == nil {
		t.Error("New() should return non-nil normalizer")
	}
}

func TestNormalizeLinks(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		sourceURL string
	}{
		{
			name:     "simple content",
			content:  "# Title\nSome content",
			sourceURL: "https://example.com/page",
		},
		{
			name:     "content with links",
			content:  "[link](../relative)",
			sourceURL: "https://example.com/docs/",
		},
		{
			name:     "empty content",
			content:  "",
			sourceURL: "https://example.com",
		},
	}

	n := New()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := n.normalizeLinks(tt.content, tt.sourceURL)
			if result == "" && tt.content != "" {
				t.Errorf("normalizeLinks() returned empty for non-empty input")
			}
		})
	}
}
