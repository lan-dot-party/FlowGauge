package discord

import (
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/lan-dot-party/flowgauge/internal/config"
)

func TestRenderNameDefaultTemplate(t *testing.T) {
	total := Total{DownloadMbps: 369.8, UploadMbps: 100.2, LatencyMs: 11.6, Connections: 2}
	now := time.Date(2026, 9, 21, 3, 14, 0, 0, time.UTC)

	got := RenderName(config.DefaultDiscordNameTemplate, total, now)
	want := "📶: ↓ 370 MBit/s | ↑ 100 MBit/s | 🏓 12 ms"

	if got != want {
		t.Errorf("RenderName() = %q, want %q", got, want)
	}
}

func TestRenderNamePlaceholders(t *testing.T) {
	total := Total{DownloadMbps: 369.84, UploadMbps: 100.25, LatencyMs: 11.84, Connections: 3}
	now := time.Date(2026, 9, 21, 3, 14, 0, 0, time.UTC)

	tests := []struct {
		template string
		want     string
	}{
		{"{download}/{upload}", "370/100"},
		{"{download_mbps}/{upload_mbps}", "369.8/100.2"},
		{"{download_gbps}/{upload_gbps}", "0.37/0.10"},
		{"{ping} ms", "12 ms"},
		{"{ping_ms} ms", "11.8 ms"},
		{"{latency}/{latency_ms}", "12/11.8"},
		{"{connections} WANs", "3 WANs"},
		{"as of {date} {time}", "as of 2026-09-21 03:14"},
		{"no placeholders", "no placeholders"},
	}

	for _, tc := range tests {
		if got := RenderName(tc.template, total, now); got != tc.want {
			t.Errorf("RenderName(%q) = %q, want %q", tc.template, got, tc.want)
		}
	}
}

func TestTruncateNameKeepsMultiByteRunesIntact(t *testing.T) {
	// 60 arrows are 180 bytes but only 60 runes, so nothing is cut.
	short := strings.Repeat("↓", 60)
	if got := TruncateName(short); got != short {
		t.Errorf("TruncateName() shortened a %d rune name", utf8.RuneCountInString(short))
	}

	long := strings.Repeat("📶", 150)
	got := TruncateName(long)

	if n := utf8.RuneCountInString(got); n != MaxChannelNameLength {
		t.Errorf("TruncateName() returned %d runes, want %d", n, MaxChannelNameLength)
	}
	if !utf8.ValidString(got) {
		t.Error("TruncateName() cut a multi-byte rune in half")
	}
}

// The shipped default must fit into Discord's limit even at gigabit speeds.
func TestDefaultTemplateFitsChannelLimit(t *testing.T) {
	total := Total{DownloadMbps: 9999.9, UploadMbps: 9999.9, LatencyMs: 999.9, Connections: 99}

	got := RenderName(config.DefaultDiscordNameTemplate, total, time.Now())

	if n := utf8.RuneCountInString(got); n > MaxChannelNameLength {
		t.Errorf("default template rendered %d runes: %q", n, got)
	}
}
