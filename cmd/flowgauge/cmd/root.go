// Package cmd contains all CLI commands for FlowGauge.
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/lan-dot-party/flowgauge/internal/config"
	"github.com/lan-dot-party/flowgauge/internal/logger"
	"github.com/lan-dot-party/flowgauge/pkg/version"
)

var (
	// Global flags
	cfgFile string
	verbose bool

	// Loaded configuration (available to subcommands)
	cfg *config.Config
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "flowgauge",
	Short: "FlowGauge - Bandwidth monitoring with Multi-WAN support",
	Long: `FlowGauge is a modular bandwidth testing tool with:

  • Multi-WAN Support - Test multiple connections with different source IPs
  • DSCP Tagging - Set QoS flags for realistic testing
  • Scheduled Tests - Automatic testing via cron syntax
  • REST API - JSON API compatible with Grafana
  • Prometheus Metrics - Native monitoring support

Documentation: https://github.com/lan-dot-party/flowgauge`,
	Version: version.GetVersion(),
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Skip config loading for certain commands
		if cmd.Name() == "version" || cmd.Name() == "help" {
			return nil
		}
		if cmd.Parent() != nil && cmd.Parent().Name() == "config" && cmd.Name() == "init" {
			return nil
		}

		// Initialize logger based on verbose flag
		development := logger.IsDevelopment()
		logLevel := "info"
		if verbose {
			logLevel = "debug"
		}
		if err := logger.Init(logLevel, development); err != nil {
			return fmt.Errorf("failed to initialize logger: %w", err)
		}

		// Load configuration (for commands that need it)
		var err error
		cfg, err = config.Load(cfgFile)
		if err != nil {
			// Don't fail if config not found for some commands
			if cmd.Name() == "config" {
				return nil
			}
			return fmt.Errorf("failed to load configuration: %w", err)
		}

		// Reinitialize logger with config settings (verbose flag takes precedence)
		finalLogLevel := cfg.General.LogLevel
		if verbose {
			finalLogLevel = "debug"
		}
		if err := logger.Init(finalLogLevel, development); err != nil {
			return fmt.Errorf("failed to reinitialize logger: %w", err)
		}

		return nil
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	// Global persistent flags
	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", 
		"config file (default: /etc/flowgauge/config.yaml)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, 
		"enable verbose/debug output")

	// Version template
	rootCmd.SetVersionTemplate(`{{printf "FlowGauge %s\n" .Version}}`)
}

// GetConfig returns the loaded configuration.
// Returns nil if config hasn't been loaded yet.
func GetConfig() *config.Config {
	return cfg
}

// SetConfig sets the configuration (useful for testing).
func SetConfig(c *config.Config) {
	cfg = c
}

