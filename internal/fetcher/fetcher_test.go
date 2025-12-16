package fetcher

import (
	"testing"
)

func TestNewFetcher(t *testing.T) {
	baseDir := "/tmp/test_fetcher"
	f := New(baseDir)
	if f == nil {
		t.Error("New() should return non-nil fetcher")
	}
}
