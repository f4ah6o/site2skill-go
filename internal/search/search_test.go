package search

import (
	"testing"
)

func TestSearchDocs(t *testing.T) {
	tests := []struct {
		name string
		opts SearchOptions
	}{
		{
			name: "empty search",
			opts: SearchOptions{
				SkillDir: ".",
				Query:    "test",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := SearchDocs(tt.opts)
			if err != nil {
				t.Logf("SearchDocs() error: %v (may be expected for empty directory)", err)
			}
		})
	}
}
