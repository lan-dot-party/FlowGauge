package discord

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/lan-dot-party/flowgauge/pkg/version"
)

// apiBaseURL is the Discord REST API base. It is a variable so tests can point
// the client at a local server.
var apiBaseURL = "https://discord.com/api/v10"

// Client talks to the Discord REST API with a bot token.
//
// Only the single call FlowGauge needs is implemented, so the project stays
// free of a Discord library dependency.
type Client struct {
	token      string
	httpClient *http.Client
}

// NewClient creates a Discord API client for the given bot token.
func NewClient(token string, timeout time.Duration) *Client {
	if timeout <= 0 {
		timeout = 15 * time.Second
	}

	return &Client{
		token:      token,
		httpClient: &http.Client{Timeout: timeout},
	}
}

// SetChannelName renames a channel via PATCH /channels/{id}.
//
// Discord rate-limits channel edits to 2 per 10 minutes per channel. On 429 the
// call waits out the retry_after once and then gives up, so a caller can never
// block a test run for minutes.
func (c *Client) SetChannelName(ctx context.Context, channelID, name string) error {
	name = TruncateName(name)

	resp, err := c.patchChannel(ctx, channelID, name)
	if err != nil {
		return err
	}

	if resp.statusCode == http.StatusTooManyRequests && resp.retryAfter > 0 {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(resp.retryAfter):
		}

		resp, err = c.patchChannel(ctx, channelID, name)
		if err != nil {
			return err
		}
	}

	return apiError(resp, channelID)
}

// patchResponse is the part of a Discord response the client acts on.
type patchResponse struct {
	statusCode int
	body       string
	retryAfter time.Duration
}

func (c *Client) patchChannel(ctx context.Context, channelID, name string) (*patchResponse, error) {
	payload, err := json.Marshal(map[string]string{"name": name})
	if err != nil {
		return nil, fmt.Errorf("failed to encode channel payload: %w", err)
	}

	url := fmt.Sprintf("%s/channels/%s", apiBaseURL, channelID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPatch, url, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("failed to create discord request: %w", err)
	}

	req.Header.Set("Authorization", "Bot "+c.token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "DiscordBot (https://github.com/lan-dot-party/flowgauge, "+version.GetShortVersion()+")")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		// The error may contain the URL but never the Authorization header.
		return nil, fmt.Errorf("discord API request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Responses are small; cap the read anyway so a broken proxy cannot flood us.
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<10))

	out := &patchResponse{
		statusCode: resp.StatusCode,
		body:       string(body),
	}

	if resp.StatusCode == http.StatusTooManyRequests {
		out.retryAfter = parseRetryAfter(body)
	}

	return out, nil
}

// parseRetryAfter reads the retry_after field (seconds, may be fractional) from
// a Discord rate limit response. It caps the wait so a long rate limit does not
// stall the caller.
func parseRetryAfter(body []byte) time.Duration {
	var rateLimit struct {
		RetryAfter float64 `json:"retry_after"`
	}
	if err := json.Unmarshal(body, &rateLimit); err != nil || rateLimit.RetryAfter <= 0 {
		return 0
	}

	retryAfter := time.Duration(rateLimit.RetryAfter * float64(time.Second))
	if retryAfter > 30*time.Second {
		return 0
	}

	return retryAfter
}

// apiError turns a non-2xx response into an actionable error message.
func apiError(resp *patchResponse, channelID string) error {
	if resp.statusCode >= 200 && resp.statusCode < 300 {
		return nil
	}

	switch resp.statusCode {
	case http.StatusUnauthorized:
		return errors.New("discord rejected the bot token (401) - check discord.bot_token")
	case http.StatusForbidden:
		return fmt.Errorf("discord denied the channel update (403) - the bot needs the MANAGE_CHANNELS permission on channel %s", channelID)
	case http.StatusNotFound:
		return fmt.Errorf("discord channel %s not found (404) - check discord.channel_id and that the bot is a member of that server", channelID)
	case http.StatusTooManyRequests:
		return fmt.Errorf("discord rate limit hit for channel %s (429) - a channel name can only be changed twice per 10 minutes", channelID)
	default:
		return fmt.Errorf("discord API returned %d: %s", resp.statusCode, resp.body)
	}
}
