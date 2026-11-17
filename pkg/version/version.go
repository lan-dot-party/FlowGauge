// Package version provides build version information.
package version

import (
	"fmt"
	"regexp"
	"strings"
)

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

// semverPrefix matches "1.2" at the start, so commit hashes such as
// "7c5bc01" are not mistaken for versions.
var semverPrefix = regexp.MustCompile(`^[0-9]+\.[0-9]+`)

// GetDisplayVersion returns the version for display with exactly one
// leading "v" for numeric versions ("1.1.0" and "v1.1.0" both become
// "v1.1.0"); other values such as "dev" are returned unchanged.
func GetDisplayVersion() string {
	if v := strings.TrimPrefix(Version, "v"); semverPrefix.MatchString(v) {
		return "v" + v
	}
	return Version
}
