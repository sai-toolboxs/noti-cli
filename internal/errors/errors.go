package errors

import (
	"encoding/json"
	stderrors "errors"
	"fmt"
	"os"
)

// Code represents a machine-readable error category.
type Code string

const (
	CodeUsage           Code = "usage"
	CodeConfiguration   Code = "configuration"
	CodeAuthentication  Code = "authentication"
	CodeNotFound        Code = "not_found"
	CodeAmbiguousTarget Code = "ambiguous_target"
	CodeValidation      Code = "validation"
	CodeConflict        Code = "conflict"
	CodeRateLimit       Code = "rate_limit"
	CodeNotionAPI       Code = "notion_api"
	CodeNetwork         Code = "network"
	CodeTimeout         Code = "timeout"
	CodeInternal        Code = "internal"
)

// Exit codes matching the design specification.
const (
	ExitSuccess          = 0
	ExitGeneral          = 1
	ExitUsage            = 2
	ExitAuthentication   = 3
	ExitNotFound         = 4
	ExitAmbiguousTarget  = 5
	ExitValidation       = 6
	ExitConflict         = 7
	ExitRateLimited      = 8
	ExitNetworkTimeout   = 9
	ExitNotionAPIFailure = 10
)

// Error is a structured error with a machine-readable code.
type Error struct {
	Code    Code     `json:"code"`
	Message string   `json:"message"`
	Details []string `json:"details,omitempty"`
	cause   error
}

func (e *Error) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return string(e.Code)
}

func (e *Error) Unwrap() error {
	return e.cause
}

// JSON returns the error as a JSON-compatible structure.
func (e *Error) JSON() map[string]interface{} {
	return map[string]interface{}{
		"error": map[string]interface{}{
			"code":    e.Code,
			"message": e.Message,
			"details": e.Details,
		},
	}
}

// ExitCode maps an error code to its exit code.
func ExitCode(code Code) int {
	switch code {
	case CodeUsage:
		return ExitUsage
	case CodeConfiguration:
		return ExitGeneral
	case CodeAuthentication:
		return ExitAuthentication
	case CodeNotFound:
		return ExitNotFound
	case CodeAmbiguousTarget:
		return ExitAmbiguousTarget
	case CodeValidation:
		return ExitValidation
	case CodeConflict:
		return ExitConflict
	case CodeRateLimit:
		return ExitRateLimited
	case CodeNetwork, CodeTimeout:
		return ExitNetworkTimeout
	case CodeNotionAPI:
		return ExitNotionAPIFailure
	default:
		return ExitGeneral
	}
}

// New creates a new Error with the given code and message.
func New(code Code, msg string) *Error {
	return &Error{Code: code, Message: msg}
}

// Wrap creates a new Error that wraps a cause.
func Wrap(code Code, msg string, cause error) *Error {
	return &Error{Code: code, Message: msg, cause: cause}
}

// WithDetails returns a copy of the error with added details.
func (e *Error) WithDetails(details ...string) *Error {
	copied := *e
	copied.Details = append(append([]string{}, e.Details...), details...)
	return &copied
}

// As wraps stderrors.As for use with our Error type.
func As(err error, target interface{}) bool {
	return stderrors.As(err, target)
}

// Is wraps stderrors.Is.
func Is(err, target error) bool {
	return stderrors.Is(err, target)
}

// PrintError prints the error to stderr in the appropriate format.
func PrintError(err error, jsonOutput bool) {
	if err == nil {
		return
	}

	var nerr *Error
	if stderrors.As(err, &nerr) {
		if jsonOutput {
			enc := json.NewEncoder(os.Stderr)
			enc.SetIndent("", "  ")
			enc.Encode(nerr.JSON())
		} else {
			fmt.Fprintf(os.Stderr, "error: [%s] %s\n", nerr.Code, nerr.Message)
		}
		return
	}

	if jsonOutput {
		enc := json.NewEncoder(os.Stderr)
		enc.SetIndent("", "  ")
		enc.Encode(map[string]interface{}{
			"error": map[string]interface{}{
				"code":    string(CodeInternal),
				"message": err.Error(),
			},
		})
	} else {
		fmt.Fprintf(os.Stderr, "error: %s\n", err.Error())
	}
}

// CodeFromExitCode maps an exit code back to a Code.
func CodeFromExitCode(code int) Code {
	switch code {
	case ExitUsage:
		return CodeUsage
	case ExitAuthentication:
		return CodeAuthentication
	case ExitNotFound:
		return CodeNotFound
	case ExitAmbiguousTarget:
		return CodeAmbiguousTarget
	case ExitValidation:
		return CodeValidation
	case ExitConflict:
		return CodeConflict
	case ExitRateLimited:
		return CodeRateLimit
	case ExitNetworkTimeout:
		return CodeNetwork
	case ExitNotionAPIFailure:
		return CodeNotionAPI
	default:
		return CodeInternal
	}
}
