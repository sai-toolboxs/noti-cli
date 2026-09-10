package errors

import (
	"encoding/json"
	stderrors "errors"
	"testing"
)

func TestNewError(t *testing.T) {
	err := New(CodeUsage, "bad argument")
	if err.Code != CodeUsage {
		t.Errorf("expected code %s, got %s", CodeUsage, err.Code)
	}
	if err.Error() != "bad argument" {
		t.Errorf("expected message 'bad argument', got %q", err.Error())
	}
}

func TestWrapError(t *testing.T) {
	cause := stderrors.New("original")
	err := Wrap(CodeNetwork, "request failed", cause)
	if err.Code != CodeNetwork {
		t.Errorf("expected code %s, got %s", CodeNetwork, err.Code)
	}
	if !stderrors.Is(err, cause) {
		t.Error("expected error to wrap cause")
	}
}

func TestWithDetails(t *testing.T) {
	err := New(CodeValidation, "invalid input")
	err2 := err.WithDetails("field 'name' is required")
	if len(err2.Details) != 1 {
		t.Fatalf("expected 1 detail, got %d", len(err2.Details))
	}
	if err2.Details[0] != "field 'name' is required" {
		t.Errorf("unexpected detail: %s", err2.Details[0])
	}
	// Original should not be modified.
	if len(err.Details) != 0 {
		t.Error("original error should not have details")
	}
}

func TestExitCodeMapping(t *testing.T) {
	tests := []struct {
		code Code
		want int
	}{
		{CodeUsage, ExitUsage},
		{CodeAuthentication, ExitAuthentication},
		{CodeNotFound, ExitNotFound},
		{CodeAmbiguousTarget, ExitAmbiguousTarget},
		{CodeValidation, ExitValidation},
		{CodeConflict, ExitConflict},
		{CodeRateLimit, ExitRateLimited},
		{CodeNetwork, ExitNetworkTimeout},
		{CodeTimeout, ExitNetworkTimeout},
		{CodeNotionAPI, ExitNotionAPIFailure},
		{CodeInternal, ExitGeneral},
	}
	for _, tt := range tests {
		got := ExitCode(tt.code)
		if got != tt.want {
			t.Errorf("ExitCode(%s) = %d, want %d", tt.code, got, tt.want)
		}
	}
}

func TestCodeFromExitCode(t *testing.T) {
	// Round-trip test.
	for _, code := range []Code{
		CodeUsage, CodeAuthentication, CodeNotFound, CodeAmbiguousTarget,
		CodeValidation, CodeConflict, CodeRateLimit, CodeNetwork,
		CodeNotionAPI, CodeInternal,
	} {
		exit := ExitCode(code)
		got := CodeFromExitCode(exit)
		if got != code {
			t.Errorf("round-trip failed: %s -> %d -> %s", code, exit, got)
		}
	}
}

func TestErrorJSON(t *testing.T) {
	err := New(CodeNotFound, "page not found")
	j := err.JSON()
	data, err2 := json.Marshal(j)
	if err2 != nil {
		t.Fatalf("failed to marshal JSON: %v", err2)
	}
	var parsed map[string]interface{}
	if err3 := json.Unmarshal(data, &parsed); err3 != nil {
		t.Fatalf("failed to unmarshal JSON: %v", err3)
	}
	errObj, ok := parsed["error"].(map[string]interface{})
	if !ok {
		t.Fatal("expected 'error' key in JSON")
	}
	if errObj["code"] != "not_found" {
		t.Errorf("expected code 'not_found', got %v", errObj["code"])
	}
}

func TestUnwrap(t *testing.T) {
	inner := stderrors.New("inner error")
	outer := Wrap(CodeInternal, "outer", inner)
	if !stderrors.Is(outer, inner) {
		t.Error("expected outer to unwrap to inner")
	}
}
