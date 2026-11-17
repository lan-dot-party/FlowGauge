// Package duration parses durations like time.ParseDuration, plus a
// leading day count ("7d", "30d", "1d12h") as used in the API and CLI.
package duration

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// maxDays keeps the result well inside time.Duration's range (~292 years).
const maxDays = 36500

// Parse accepts every time.ParseDuration format and additionally an
// optional leading "<n>d" (days of 24h), optionally followed by a normal
// duration: "7d", "30d", "1d12h". Negative values are rejected.
func Parse(s string) (time.Duration, error) {
	s = strings.TrimSpace(s)
	var days time.Duration
	if i := strings.IndexByte(s, 'd'); i > 0 {
		n, err := strconv.Atoi(s[:i])
		if err != nil || n < 0 || n > maxDays {
			return 0, fmt.Errorf("invalid duration %q", s)
		}
		days = time.Duration(n) * 24 * time.Hour
		s = s[i+1:]
		if s == "" {
			return days, nil
		}
	}
	d, err := time.ParseDuration(s)
	if err != nil {
		return 0, fmt.Errorf("invalid duration %q: %w", s, err)
	}
	if d < 0 {
		return 0, fmt.Errorf("invalid duration %q: negative", s)
	}
	return days + d, nil
}
