package packager

import (
	"testing"
)

func TestNewPackager(t *testing.T) {
	p := New()
	if p == nil {
		t.Error("New() should return non-nil packager")
	}
}
