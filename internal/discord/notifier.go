package discord

import (
	"context"
	"time"

	"go.uber.org/zap"

	"github.com/lan-dot-party/flowgauge/internal/config"
	"github.com/lan-dot-party/flowgauge/internal/speedtest"
)

// Notifier renames a Discord channel with the total bandwidth of a test run.
//
// All methods are safe to call on a nil Notifier, so callers do not have to
// branch on whether the integration is configured.
type Notifier struct {
	cfg      config.DiscordConfig
	counting map[string]bool
	client   *Client
	logger   *zap.Logger
}

// NewNotifier creates a Notifier from the application config. It returns nil
// when the Discord integration is disabled or no config was given.
func NewNotifier(cfg *config.Config, logger *zap.Logger) *Notifier {
	if cfg == nil || !cfg.Discord.Enabled {
		return nil
	}

	if logger == nil {
		logger = zap.NewNop()
	}

	return &Notifier{
		cfg:      cfg.Discord,
		counting: CountingNames(cfg.GetTotalConnections()),
		client:   NewClient(cfg.Discord.BotToken, cfg.Discord.Timeout),
		logger:   logger,
	}
}

// Enabled reports whether the notifier will actually talk to Discord.
func (n *Notifier) Enabled() bool {
	return n != nil
}

// ChannelID returns the configured channel, or an empty string if disabled.
func (n *Notifier) ChannelID() string {
	if n == nil {
		return ""
	}
	return n.cfg.ChannelID
}

// UpdateFromResults sums up the results of a completed test run and renames the
// configured channel. It returns the name that was set, which is empty when the
// notifier is disabled or the run contained nothing to report.
func (n *Notifier) UpdateFromResults(ctx context.Context, results []speedtest.Result) (string, error) {
	if n == nil {
		return "", nil
	}

	total := Aggregate(results, n.counting)
	if total.Connections == 0 {
		n.logger.Warn("Skipping Discord update: no successful result counts towards the total",
			zap.Int("results", len(results)),
			zap.Int("skipped", total.Skipped),
		)
		return "", nil
	}

	name := RenderName(n.cfg.NameTemplate, total, time.Now())

	if err := n.client.SetChannelName(ctx, n.cfg.ChannelID, name); err != nil {
		// Log the intended name so a failed update can be diagnosed from the
		// log alone, without re-running the speedtest.
		n.logger.Debug("Discord channel name not applied",
			zap.String("channel_id", n.cfg.ChannelID),
			zap.String("intended_name", name),
		)
		return "", err
	}

	n.logger.Info("Discord channel name updated",
		zap.String("channel_id", n.cfg.ChannelID),
		zap.String("name", name),
		zap.Float64("total_download_mbps", total.DownloadMbps),
		zap.Float64("total_upload_mbps", total.UploadMbps),
		zap.Float64("worst_latency_ms", total.LatencyMs),
		zap.Int("connections", total.Connections),
	)

	return name, nil
}
