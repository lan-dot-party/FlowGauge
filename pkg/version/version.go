// Package version provides build version information.
package version

import "fmt"

// Build-time variables - set via ldflags
var (
	// Version is the semantic version of the application
	Version = "dev"
	// Commit is the git commit hash
	Commit = "none"
	// BuildDate is the date when the binary was built
	BuildDate = "unknown"
)

// GetVersion returns the full version string including commit and build date.
func GetVersion() string {
	return fmt.Sprintf("%s (commit: %s, built: %s)", Version, Commit, BuildDate)
}

// GetShortVersion returns only the semantic version.
func GetShortVersion() string {
	return Version
}

