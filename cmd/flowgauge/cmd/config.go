package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/lan-dot-party/flowgauge/internal/config"
)

// configCmd represents the config command
var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Configuration management",
	Long:  `Commands for managing FlowGauge configuration.`,
}

// configValidateCmd validates the configuration
var configValidateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate the configuration file",
	Long: `Check the configuration file for errors.

Examples:
  flowgauge config validate
  flowgauge config validate --config /path/to/config.yaml`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := GetConfig()
		if cfg == nil {
			return fmt.Errorf("configuration not loaded")
		}

		fmt.Println("✅ Configuration is valid!")
		fmt.Printf("   Connections: %d configured, %d enabled\n",
			len(cfg.Connections), len(cfg.GetEnabledConnections()))
		fmt.Printf("   Storage: %s\n", cfg.Storage.Type)
		fmt.Printf("   Webserver: %s (enabled: %t)\n", cfg.Webserver.Listen, cfg.Webserver.Enabled)
		fmt.Printf("   Scheduler: %s (enabled: %t)\n", cfg.Scheduler.Schedule, cfg.Scheduler.Enabled)

		return nil
	},
}

// configShowCmd shows the current configuration
var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show the current configuration",
	Long: `Display the current configuration with all defaults applied.

Examples:
  flowgauge config show`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := GetConfig()
		if cfg == nil {
			return fmt.Errorf("configuration not loaded")
		}

		data, err := yaml.Marshal(cfg)
		if err != nil {
			return fmt.Errorf("failed to marshal config: %w", err)
		}

		fmt.Println("# Current FlowGauge Configuration")
		fmt.Println("# (with defaults applied)")
		fmt.Println()
		fmt.Print(string(data))

		return nil
	},
}

// configInitCmd generates an example configuration
var configInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Generate an example configuration",
	Long: `Generate an example configuration file to stdout.

Examples:
  # Print example config to stdout
  flowgauge config init

  # Save example config to file
  flowgauge config init > /etc/flowgauge/config.yaml`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Read the example config file
		examplePath := "configs/flowgauge.example.yaml"
		
		data, err := os.ReadFile(examplePath)
		if err != nil {
			// If example file not found, generate from defaults
			cfg := config.NewDefault()
			cfg.Connections = []config.ConnectionConfig{
				{
					Name:     "WAN1-Primary",
					SourceIP: "",
					DSCP:     0,
					Enabled:  true,
				},
			}
			
			yamlData, err := yaml.Marshal(cfg)
			if err != nil {
				return fmt.Errorf("failed to generate config: %w", err)
			}
			
			fmt.Println("# FlowGauge Configuration")
			fmt.Println("# Generated from defaults")
			fmt.Println()
			fmt.Print(string(yamlData))
			return nil
		}

		fmt.Print(string(data))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(configValidateCmd)
	configCmd.AddCommand(configShowCmd)
	configCmd.AddCommand(configInitCmd)
}

