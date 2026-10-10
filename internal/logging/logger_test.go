package logging

import "testing"

func TestNewLevels(t *testing.T) {
	levels := []string{"debug", "info", "warn", "error", "off"}

	for _, level := range levels {
		for _, format := range []string{"text", "json"} {
			logger, err := New(level, format)
			if err != nil {
				t.Fatalf("level %s format %s: unexpected error: %v", level, format, err)
			}
			if logger == nil {
				t.Fatalf("level %s: expected logger", level)
			}
		}
	}
}

func TestInvalidLevel(t *testing.T) {
	if _, err := New("verbose", "text"); err == nil {
		t.Fatal("expected error for invalid level")
	}

	if _, err := New("info", "yaml"); err == nil {
		t.Fatal("expected error for invalid format")
	}
}

func TestWithComponent(t *testing.T) {
	logger, err := New("info", "text")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	child := logger.WithComponent("proxy")
	if child == nil {
		t.Fatal("expected child logger")
	}
}
