package output

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestNewDefault(t *testing.T) {
	w := New(FormatText)
	if w.Format() != FormatText {
		t.Errorf("expected FormatText, got %s", w.Format())
	}
	if w.IsJSON() {
		t.Error("expected non-JSON mode")
	}
}

func TestNewJSON(t *testing.T) {
	w := New(FormatJSON)
	if !w.IsJSON() {
		t.Error("expected JSON mode")
	}
}

func TestPrintText(t *testing.T) {
	var stdout, stderr bytes.Buffer
	w := NewWithWriters(FormatText, &stdout, &stderr)
	w.Print("hello")
	if !strings.Contains(stdout.String(), "hello") {
		t.Errorf("expected 'hello' in output, got %q", stdout.String())
	}
}

func TestPrintJSONSuppressed(t *testing.T) {
	var stdout, stderr bytes.Buffer
	w := NewWithWriters(FormatJSON, &stdout, &stderr)
	w.Print("this should not appear in stdout")
	if stdout.Len() != 0 {
		t.Errorf("expected empty stdout in JSON mode, got %q", stdout.String())
	}
}

func TestPrintJSON(t *testing.T) {
	var stdout, stderr bytes.Buffer
	w := NewWithWriters(FormatJSON, &stdout, &stderr)
	w.PrintJSON(map[string]string{"key": "value"})
	var parsed map[string]string
	if err := json.Unmarshal(stdout.Bytes(), &parsed); err != nil {
		t.Fatalf("invalid JSON output: %v", err)
	}
	if parsed["key"] != "value" {
		t.Errorf("expected key=value, got %v", parsed)
	}
}

func TestDiag(t *testing.T) {
	var stdout, stderr bytes.Buffer
	w := NewWithWriters(FormatText, &stdout, &stderr)
	w.Diag("debug message")
	if !strings.Contains(stderr.String(), "debug message") {
		t.Errorf("expected 'debug message' in stderr, got %q", stderr.String())
	}
}

func TestDiagf(t *testing.T) {
	var stdout, stderr bytes.Buffer
	w := NewWithWriters(FormatText, &stdout, &stderr)
	w.Diagf("count: %d", 42)
	if !strings.Contains(stderr.String(), "count: 42") {
		t.Errorf("expected 'count: 42' in stderr, got %q", stderr.String())
	}
}

func TestPrintf(t *testing.T) {
	var stdout, stderr bytes.Buffer
	w := NewWithWriters(FormatText, &stdout, &stderr)
	w.Printf("hello %s", "world")
	if !strings.Contains(stdout.String(), "hello world") {
		t.Errorf("expected 'hello world', got %q", stdout.String())
	}
}
