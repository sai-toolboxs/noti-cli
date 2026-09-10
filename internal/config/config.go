package config

import (
	"os"
	"strconv"
	"time"

	nerr "github.com/sai-toolboxs/noti-cli/internal/errors"
)

// Config holds all configuration for the CLI.
type Config struct {
	Token      string
	APIBaseURL string
	APIVersion string
	Timeout    time.Duration
	OutputJSON bool
	Verbose    bool
}

// DefaultAPIBaseURL is the Notion API base URL.
const DefaultAPIBaseURL = "https://api.notion.com"

// DefaultAPIVersion is the Notion API version header value.
const DefaultAPIVersion = "2022-06-28"

// DefaultTimeout is the default HTTP client timeout.
const DefaultTimeout = 30 * time.Second

// Load creates a Config from environment variables and defaults.
// Precedence: environment > defaults.
func Load() (*Config, error) {
	cfg := &Config{
		APIBaseURL: DefaultAPIBaseURL,
		APIVersion: DefaultAPIVersion,
		Timeout:    DefaultTimeout,
	}

	// Canonical env var is NOTION_API_TOKEN.
	// Fall back to NOTION_TOKEN for compatibility.
	if v := os.Getenv("NOTION_API_TOKEN"); v != "" {
		cfg.Token = v
	} else if v := os.Getenv("NOTION_TOKEN"); v != "" {
		cfg.Token = v
	}

	if v := os.Getenv("NOTION_API_BASE_URL"); v != "" {
		cfg.APIBaseURL = v
	}

	if v := os.Getenv("NOTION_VERSION"); v != "" {
		cfg.APIVersion = v
	}

	if v := os.Getenv("NOTION_TIMEOUT"); v != "" {
		seconds, err := strconv.Atoi(v)
		if err != nil {
			return nil, nerr.Wrap(nerr.CodeConfiguration, "invalid NOTION_TIMEOUT value", err)
		}
		if seconds <= 0 {
			return nil, nerr.New(nerr.CodeConfiguration, "NOTION_TIMEOUT must be positive")
		}
		cfg.Timeout = time.Duration(seconds) * time.Second
	}

	return cfg, nil
}

// HasToken reports whether a token is configured.
func (c *Config) HasToken() bool {
	return c.Token != ""
}

// Validate checks that required configuration is present.
func (c *Config) Validate() error {
	if !c.HasToken() {
		return nerr.New(nerr.CodeAuthentication,
			"NOTION_API_TOKEN not set; export it or pass via environment")
	}
	return nil
}
