package config

import (
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestIncludeInTotalDefaultsToTrue(t *testing.T) {
	tests := []struct {
		name string
		conn ConnectionConfig
		want bool
	}{
		{"unset", ConnectionConfig{Name: "WAN1"}, true},
		{"explicit true", ConnectionConfig{Name: "WAN1", CountsToTotal: boolPtr(true)}, true},
		{"explicit false", ConnectionConfig{Name: "WAN1", CountsToTotal: boolPtr(false)}, false},
	}

	for _, tc := range tests {
		if got := tc.conn.IncludeInTotal(); got != tc.want {
			t.Errorf("%s: IncludeInTotal() = %v, want %v", tc.name, got, tc.want)
		}
	}
}

// An existing config without counts_to_total must keep counting every
// connection, so upgrading FlowGauge does not silently report 0 MBit/s.
func TestCountsToTotalParsesFromYAML(t *testing.T) {
	data := []byte(`
connections:
  - name: WAN1
    enabled: true
  - name: VoIP
    enabled: true
    counts_to_total: false
`)

	cfg := &Config{}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		t.Fatalf("failed to parse config: %v", err)
	}

	if cfg.Connections[0].CountsToTotal != nil {
		t.Error("WAN1: counts_to_total should stay unset when absent from the YAML")
	}
	if !cfg.Connections[0].IncludeInTotal() {
		t.Error("WAN1: should count towards the total")
	}
	if cfg.Connections[1].IncludeInTotal() {
		t.Error("VoIP: counts_to_total: false should be honoured")
	}
}

func TestGetTotalConnections(t *testing.T) {
	cfg := &Config{
		Connections: []ConnectionConfig{
			{Name: "WAN1", Enabled: true},
			{Name: "VoIP", Enabled: true, CountsToTotal: boolPtr(false)},
			{Name: "Old", Enabled: false},
		},
	}

	total := cfg.GetTotalConnections()

	if len(total) != 1 || total[0].Name != "WAN1" {
		t.Errorf("GetTotalConnections() = %+v, want only WAN1", total)
	}
}

func TestValidateDiscord(t *testing.T) {
	newConfig := func(discord DiscordConfig) *Config {
		cfg := NewDefault()
		cfg.Connections = []ConnectionConfig{{Name: "WAN1", Enabled: true}}
		cfg.Discord = discord
		ApplyDefaults(cfg)
		return cfg
	}

	valid := DiscordConfig{Enabled: true, BotToken: "token", ChannelID: "123456789012345678"}

	tests := []struct {
		name       string
		discord    DiscordConfig
		wantSubstr string // empty means: must validate
	}{
		{"disabled needs nothing", DiscordConfig{Enabled: false}, ""},
		{"valid", valid, ""},
		{"missing token", DiscordConfig{Enabled: true, ChannelID: "123"}, "bot_token"},
		{"missing channel", DiscordConfig{Enabled: true, BotToken: "token"}, "channel_id"},
		{"channel name instead of ID", DiscordConfig{Enabled: true, BotToken: "token", ChannelID: "#speed"}, "numeric channel ID"},
		{
			"template too long",
			DiscordConfig{Enabled: true, BotToken: "token", ChannelID: "123", NameTemplate: strings.Repeat("a", 101)},
			"name_template",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := Validate(newConfig(tc.discord))

			if tc.wantSubstr == "" {
				if err != nil {
					t.Fatalf("Validate() returned %v, want no error", err)
				}
				return
			}

			if err == nil {
				t.Fatalf("Validate() returned no error, want one mentioning %q", tc.wantSubstr)
			}
			if !strings.Contains(err.Error(), tc.wantSubstr) {
				t.Errorf("error %q does not mention %q", err, tc.wantSubstr)
			}
		})
	}
}

// Enabling Discord without a single counting connection would always report
// 0 MBit/s, so it must be rejected instead of failing silently at runtime.
func TestValidateDiscordRejectsNoCountingConnections(t *testing.T) {
	cfg := NewDefault()
	cfg.Connections = []ConnectionConfig{{Name: "VoIP", Enabled: true, CountsToTotal: boolPtr(false)}}
	cfg.Discord = DiscordConfig{Enabled: true, BotToken: "token", ChannelID: "123"}
	ApplyDefaults(cfg)

	err := Validate(cfg)
	if err == nil || !strings.Contains(err.Error(), "counts_to_total") {
		t.Errorf("Validate() = %v, want an error about counts_to_total", err)
	}
}

func TestDiscordConfigRedactsTokenOnMarshal(t *testing.T) {
	cfg := NewDefault()
	cfg.Discord.BotToken = "super-secret-token"

	data, err := yaml.Marshal(cfg)
	if err != nil {
		t.Fatalf("failed to marshal config: %v", err)
	}

	if strings.Contains(string(data), "super-secret-token") {
		t.Error("marshalled config leaks the Discord bot token")
	}
	if !strings.Contains(string(data), "***") {
		t.Error("marshalled config does not show the redacted token")
	}
}
