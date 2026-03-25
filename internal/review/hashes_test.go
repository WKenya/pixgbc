package review

import (
	"testing"

	"github.com/WKenya/pixgbc/internal/core"
)

func TestHashConfigStable(t *testing.T) {
	a, err := HashConfig(core.Config{})
	if err != nil {
		t.Fatalf("HashConfig() error = %v", err)
	}

	b, err := HashConfig(core.DefaultConfig())
	if err != nil {
		t.Fatalf("HashConfig() error = %v", err)
	}

	if a != b {
		t.Fatalf("HashConfig() mismatch: %q != %q", a, b)
	}
}
