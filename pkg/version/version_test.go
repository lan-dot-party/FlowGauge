package version

import "testing"

func TestGetDisplayVersion(t *testing.T) {
	orig := Version
	defer func() { Version = orig }()
	for in, want := range map[string]string{
		"1.1.0":              "v1.1.0",
		"v1.1.0":             "v1.1.0",
		"v1.1.0-21-gcd3d3a2": "v1.1.0-21-gcd3d3a2",
		"dev":                "dev",
		"cd3d3a2":            "cd3d3a2",
		"7c5bc01-dirty":      "7c5bc01-dirty",
		"":                   "",
	} {
		Version = in
		if got := GetDisplayVersion(); got != want {
			t.Errorf("Version %q: got %q, want %q", in, got, want)
		}
	}
}
