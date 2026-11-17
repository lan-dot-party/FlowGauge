package speedtest

import (
	"strings"
	"testing"
)

func TestSanitizeForTerminal(t *testing.T) {
	in := "a\x1b[31mred\x07\x7f\u009bb\nc\tÄ日"
	want := "a[31mredbcÄ日"
	if got := SanitizeForTerminal(in); got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestTruncateRunes(t *testing.T) {
	if got := TruncateRunes("日本語日本語", 5); got != "日本..." {
		t.Fatalf("got %q", got)
	}
	if got := TruncateRunes("short", 20); got != "short" {
		t.Fatalf("got %q", got)
	}
	if got := TruncateRunes("\x1b[2Jabcdefghij", 5); strings.ContainsRune(got, 0x1b) || got != "[2..." {
		t.Fatalf("got %q", got)
	}
}

func TestResultFormattingStripsControl(t *testing.T) {
	r := Result{ConnectionName: "wan\x1b[1m", Error: "bad\x1b]0;x\x07"}
	for _, s := range []string{r.String(), r.FormatTable()} {
		if strings.ContainsAny(s, "\x1b\x07") {
			t.Fatalf("control character in %q", s)
		}
	}
}
