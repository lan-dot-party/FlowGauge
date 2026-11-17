package duration

import (
	"testing"
	"time"
)

func TestParse(t *testing.T) {
	ok := map[string]time.Duration{
		"24h":   24 * time.Hour,
		"90m":   90 * time.Minute,
		"7d":    7 * 24 * time.Hour,
		"30d":   30 * 24 * time.Hour,
		"1d12h": 36 * time.Hour,
		" 2d ":  48 * time.Hour,
		"0d":    0,
		"1h30m": 90 * time.Minute,
	}
	for in, want := range ok {
		got, err := Parse(in)
		if err != nil || got != want {
			t.Errorf("Parse(%q) = %v, %v; want %v", in, got, err, want)
		}
	}
	for _, in := range []string{"", "d", "xd", "-1d", "7dd", "1.5d", "-5h", "99999999d", "abc"} {
		if _, err := Parse(in); err == nil {
			t.Errorf("Parse(%q): expected error", in)
		}
	}
}
