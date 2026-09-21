package discord

import (
	"fmt"
	"strings"
	"time"
)

// MaxChannelNameLength is Discord's limit for a channel name.
const MaxChannelNameLength = 100

// RenderName fills the placeholders of a channel name template.
//
// Supported placeholders:
//
//	{download}       total download, rounded    -> "370"
//	{upload}         total upload, rounded      -> "100"
//	{download_mbps}  total download, 1 decimal  -> "369.8"
//	{upload_mbps}    total upload, 1 decimal    -> "100.2"
//	{download_gbps}  total download in Gbit/s   -> "0.37"
//	{upload_gbps}    total upload in Gbit/s     -> "0.10"
//	{ping}           worst latency, rounded     -> "12"
//	{ping_ms}        worst latency, 1 decimal   -> "11.8"
//	{connections}    connections included       -> "2"
//	{time}           time of the update         -> "03:14"
//	{date}           date of the update         -> "2026-09-21"
//
// {latency} and {latency_ms} are accepted as aliases for {ping} and {ping_ms}.
//
// The result is truncated to MaxChannelNameLength runes so multi-byte
// characters are never cut in half.
func RenderName(template string, total Total, now time.Time) string {
	replacer := strings.NewReplacer(
		"{download}", fmt.Sprintf("%.0f", total.DownloadMbps),
		"{upload}", fmt.Sprintf("%.0f", total.UploadMbps),
		"{download_mbps}", fmt.Sprintf("%.1f", total.DownloadMbps),
		"{upload_mbps}", fmt.Sprintf("%.1f", total.UploadMbps),
		"{download_gbps}", fmt.Sprintf("%.2f", total.DownloadMbps/1000),
		"{upload_gbps}", fmt.Sprintf("%.2f", total.UploadMbps/1000),
		"{ping}", fmt.Sprintf("%.0f", total.LatencyMs),
		"{ping_ms}", fmt.Sprintf("%.1f", total.LatencyMs),
		"{latency}", fmt.Sprintf("%.0f", total.LatencyMs),
		"{latency_ms}", fmt.Sprintf("%.1f", total.LatencyMs),
		"{connections}", fmt.Sprintf("%d", total.Connections),
		"{time}", now.Format("15:04"),
		"{date}", now.Format("2006-01-02"),
	)

	return TruncateName(replacer.Replace(template))
}

// TruncateName shortens a channel name to Discord's limit without splitting
// multi-byte characters.
func TruncateName(name string) string {
	runes := []rune(name)
	if len(runes) <= MaxChannelNameLength {
		return name
	}
	return string(runes[:MaxChannelNameLength])
}
