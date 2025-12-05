// Package main is the entry point for the FlowGauge CLI application.
package main

import (
	"os"

	"github.com/lan-dot-party/flowgauge/cmd/flowgauge/cmd"
	"github.com/lan-dot-party/flowgauge/internal/logger"
)

func main() {
	// Initialize default logger (will be reconfigured after config is loaded)
	logger.InitDefault()
	defer logger.Sync()

	// Execute the root command
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
