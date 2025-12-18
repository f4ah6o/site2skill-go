package converter

import (
	"os"
	"path/filepath"
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

func TestConvertFileCreatesOutputInCurrentDir(t *testing.T) {
	tmpDir := t.TempDir()
	htmlPath := filepath.Join(tmpDir, "input.html")
	html := `<html><body><main><h1>Title</h1><p>Hello</p></main></body></html>`

	if err := os.WriteFile(htmlPath, []byte(html), 0644); err != nil {
		t.Fatalf("failed to write html fixture: %v", err)
	}

	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(originalWD)
	})

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change working directory: %v", err)
	}

	c := New()
	outputPath := "output.md"

	if err := c.ConvertFile(htmlPath, outputPath, "https://example.com/docs", "2024-01-01T00:00:00Z"); err != nil {
		t.Fatalf("ConvertFile returned error: %v", err)
	}

	if _, err := os.Stat(filepath.Join(tmpDir, outputPath)); err != nil {
		t.Fatalf("expected output file to be created, got error: %v", err)
	}
}
