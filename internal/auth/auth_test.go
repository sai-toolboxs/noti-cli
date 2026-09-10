package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	nerr "github.com/sai-toolboxs/noti-cli/internal/errors"
)

func TestValidateTokenEmpty(t *testing.T) {
	err := ValidateToken("")
	if err == nil {
		t.Fatal("expected error for empty token")
	}
	var nerr2 *nerr.Error
	if !nerr.As(err, &nerr2) {
		t.Fatal("expected *nerr.Error")
	}
	if nerr2.Code != nerr.CodeAuthentication {
		t.Errorf("expected code %s, got %s", nerr.CodeAuthentication, nerr2.Code)
	}
}

func TestValidateTokenTooShort(t *testing.T) {
	err := ValidateToken("short")
	if err == nil {
		t.Fatal("expected error for short token")
	}
}

func TestValidateTokenValid(t *testing.T) {
	err := ValidateToken("ntn_1234567890abcdef")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestContextRoundTrip(t *testing.T) {
	ctx := context.Background()
	token := "test-token-1234567890"
	ctx = WithContext(ctx, token)

	got, ok := FromContext(ctx)
	if !ok {
		t.Fatal("expected token in context")
	}
	if got != token {
		t.Errorf("expected %q, got %q", token, got)
	}
}

func TestContextMissing(t *testing.T) {
	_, ok := FromContext(context.Background())
	if ok {
		t.Error("expected no token in empty context")
	}
}

func TestRoundTripper(t *testing.T) {
	var gotAuth string
	var gotVersion string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotVersion = r.Header.Get("Notion-Version")
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	rt := &RoundTripper{
		Token: "test-token-1234567890",
	}
	client := &http.Client{Transport: rt}
	req, _ := http.NewRequest("GET", ts.URL, nil)
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	resp.Body.Close()

	if gotAuth != "Bearer test-token-1234567890" {
		t.Errorf("expected Bearer token, got %q", gotAuth)
	}
	if gotVersion != "2022-06-28" {
		t.Errorf("expected Notion-Version 2022-06-28, got %q", gotVersion)
	}
}

func TestRoundTripperCustomVersion(t *testing.T) {
	var gotVersion string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotVersion = r.Header.Get("Notion-Version")
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	rt := &RoundTripper{
		Token:      "test-token-1234567890",
		APIVersion: "2023-08-01",
	}
	client := &http.Client{Transport: rt}
	req, _ := http.NewRequest("GET", ts.URL, nil)
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	resp.Body.Close()

	if gotVersion != "2023-08-01" {
		t.Errorf("expected Notion-Version 2023-08-01, got %q", gotVersion)
	}
}

func TestClientCreation(t *testing.T) {
	client := Client("test-token-1234567890", "2022-06-28", 30*time.Second)
	if client == nil {
		t.Fatal("expected non-nil client")
	}
	if client.Transport == nil {
		t.Fatal("expected non-nil transport")
	}
	if client.Timeout != 30*time.Second {
		t.Errorf("expected timeout 30s, got %v", client.Timeout)
	}
}

func TestClientWithTransport(t *testing.T) {
	inner := &http.Transport{}
	client := ClientWithTransport("test-token-1234567890", "2022-06-28", 30*time.Second, inner)
	if client == nil {
		t.Fatal("expected non-nil client")
	}
	rt, ok := client.Transport.(*RoundTripper)
	if !ok {
		t.Fatal("expected *RoundTripper transport")
	}
	if rt.Inner != inner {
		t.Error("expected inner transport to be set")
	}
	if rt.APIVersion != "2022-06-28" {
		t.Errorf("expected API version 2022-06-28, got %q", rt.APIVersion)
	}
}
