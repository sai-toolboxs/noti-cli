package config

import (
	"os"
	"testing"
	"time"
)

func TestLoadDefaults(t *testing.T) {
	// Clear env to test defaults.
	os.Unsetenv("NOTION_API_TOKEN")
	os.Unsetenv("NOTION_TOKEN")
	os.Unsetenv("NOTION_API_BASE_URL")
	os.Unsetenv("NOTION_VERSION")
	os.Unsetenv("NOTION_TIMEOUT")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.APIBaseURL != DefaultAPIBaseURL {
		t.Errorf("expected APIBaseURL %s, got %s", DefaultAPIBaseURL, cfg.APIBaseURL)
	}
	if cfg.APIVersion != DefaultAPIVersion {
		t.Errorf("expected APIVersion %s, got %s", DefaultAPIVersion, cfg.APIVersion)
	}
	if cfg.Timeout != DefaultTimeout {
		t.Errorf("expected Timeout %v, got %v", DefaultTimeout, cfg.Timeout)
	}
	if cfg.HasToken() {
		t.Error("expected no token")
	}
}

func TestLoadWithToken(t *testing.T) {
	os.Unsetenv("NOTION_TOKEN")
	os.Setenv("NOTION_API_TOKEN", "test-token-value-for-testing-12345")
	defer os.Unsetenv("NOTION_API_TOKEN")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cfg.HasToken() {
		t.Error("expected token to be set")
	}
	if cfg.Token != "test-token-value-for-testing-12345" {
		t.Errorf("unexpected token value")
	}
}

func TestLoadWithCustomValues(t *testing.T) {
	os.Unsetenv("NOTION_TOKEN")
	os.Setenv("NOTION_API_TOKEN", "test-token-value-for-testing-12345")
	os.Setenv("NOTION_API_BASE_URL", "https://custom.api.example.com")
	os.Setenv("NOTION_VERSION", "2023-01-01")
	os.Setenv("NOTION_TIMEOUT", "60")
	defer func() {
		os.Unsetenv("NOTION_API_TOKEN")
		os.Unsetenv("NOTION_API_BASE_URL")
		os.Unsetenv("NOTION_VERSION")
		os.Unsetenv("NOTION_TIMEOUT")
	}()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.APIBaseURL != "https://custom.api.example.com" {
		t.Errorf("expected custom APIBaseURL, got %s", cfg.APIBaseURL)
	}
	if cfg.APIVersion != "2023-01-01" {
		t.Errorf("expected custom APIVersion, got %s", cfg.APIVersion)
	}
	if cfg.Timeout != 60*time.Second {
		t.Errorf("expected 60s timeout, got %v", cfg.Timeout)
	}
}

func TestLoadInvalidTimeout(t *testing.T) {
	os.Setenv("NOTION_TIMEOUT", "abc")
	defer os.Unsetenv("NOTION_TIMEOUT")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for invalid timeout")
	}
}

func TestLoadNegativeTimeout(t *testing.T) {
	os.Setenv("NOTION_TIMEOUT", "-5")
	defer os.Unsetenv("NOTION_TIMEOUT")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for negative timeout")
	}
}

func TestValidate(t *testing.T) {
	cfg := &Config{Token: ""}
	if err := cfg.Validate(); err == nil {
		t.Error("expected error for empty token")
	}

	cfg.Token = "valid-token-value-for-testing"
	if err := cfg.Validate(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}
