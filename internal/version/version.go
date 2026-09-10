package version

import (
	"fmt"
)

// These variables are set at build time via -ldflags.
var (
	Version   = "dev"
	Commit    = "none"
	BuildDate = "unknown"
)

// String returns the version string.
func String() string {
	return fmt.Sprintf("noti-cli %s (commit: %s, built: %s)", Version, Commit, BuildDate)
}

// Short returns just the version number.
func Short() string {
	return Version
}
