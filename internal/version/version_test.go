package version

import (
	"strings"
	"testing"
)

func TestString(t *testing.T) {
	s := String()
	if !strings.Contains(s, "noti-cli") {
		t.Errorf("expected 'noti-cli' in version string, got %q", s)
	}
}

func TestShort(t *testing.T) {
	s := Short()
	if s == "" {
		t.Error("expected non-empty version")
	}
}

func TestDefaultValues(t *testing.T) {
	// At build time these are set via ldflags, but in tests they have defaults.
	if Version == "" {
		t.Error("version should not be empty")
	}
}
