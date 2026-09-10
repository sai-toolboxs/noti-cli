package auth

import (
	"context"
	"net/http"
	"time"

	nerr "noti-cli/internal/errors"
)

type contextKey string

const tokenKey contextKey = "notion_token"

// FromContext extracts the token from context.
func FromContext(ctx context.Context) (string, bool) {
	token, ok := ctx.Value(tokenKey).(string)
	return token, ok
}

// WithContext returns a new context carrying the token.
func WithContext(ctx context.Context, token string) context.Context {
	return context.WithValue(ctx, tokenKey, token)
}

// ValidateToken checks that a token is non-empty and has the expected prefix.
func ValidateToken(token string) error {
	if token == "" {
		return nerr.New(nerr.CodeAuthentication, "token is empty")
	}
	if len(token) < 10 {
		return nerr.New(nerr.CodeAuthentication, "token appears too short to be valid")
	}
	return nil
}

// RoundTripper injects the Authorization header into requests.
type RoundTripper struct {
	Token      string
	APIVersion string
	Inner      http.RoundTripper
}

// RoundTrip implements http.RoundTripper.
func (rt *RoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	req = req.Clone(req.Context())
	req.Header.Set("Authorization", "Bearer "+rt.Token)
	version := rt.APIVersion
	if version == "" {
		version = "2022-06-28"
	}
	req.Header.Set("Notion-Version", version)
	inner := rt.Inner
	if inner == nil {
		inner = http.DefaultTransport
	}
	return inner.RoundTrip(req)
}

// Client returns an *http.Client configured with the given token, apiVersion, and timeout.
func Client(token, apiVersion string, timeout time.Duration) *http.Client {
	return &http.Client{
		Timeout: timeout,
		Transport: &RoundTripper{
			Token:      token,
			APIVersion: apiVersion,
		},
	}
}

// ClientWithTransport returns an *http.Client with custom transport and token.
func ClientWithTransport(token, apiVersion string, timeout time.Duration, transport http.RoundTripper) *http.Client {
	return &http.Client{
		Timeout: timeout,
		Transport: &RoundTripper{
			Token:      token,
			APIVersion: apiVersion,
			Inner:      transport,
		},
	}
}
