// Package logger provides a configured zap logger for FlowGauge.
package logger

import (
	"os"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	// Log is the global logger instance
	Log *zap.Logger
	// Sugar is the sugared logger for printf-style logging
	Sugar *zap.SugaredLogger
)

// Init initializes the global logger with the specified log level.
// Valid levels: debug, info, warn, error
// Set development=true for console-friendly output, false for JSON.
func Init(level string, development bool) error {
	zapLevel := parseLevel(level)

	var config zap.Config
	if development {
		config = zap.NewDevelopmentConfig()
		config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		config.EncoderConfig.EncodeTime = zapcore.TimeEncoderOfLayout("15:04:05")
	} else {
		config = zap.NewProductionConfig()
		config.EncoderConfig.TimeKey = "timestamp"
		config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	}

	config.Level = zap.NewAtomicLevelAt(zapLevel)

	logger, err := config.Build(
		zap.AddCallerSkip(1),
		zap.AddStacktrace(zapcore.ErrorLevel),
	)
	if err != nil {
		return err
	}

	Log = logger
	Sugar = logger.Sugar()

	return nil
}

// InitDefault initializes the logger with default settings (info level, development mode).
func InitDefault() {
	if err := Init("info", true); err != nil {
		// Fallback to a basic logger if config fails
		Log = zap.NewExample()
		Sugar = Log.Sugar()
	}
}

// parseLevel converts a string log level to zapcore.Level
func parseLevel(level string) zapcore.Level {
	switch strings.ToLower(level) {
	case "debug":
		return zapcore.DebugLevel
	case "info":
		return zapcore.InfoLevel
	case "warn", "warning":
		return zapcore.WarnLevel
	case "error":
		return zapcore.ErrorLevel
	default:
		return zapcore.InfoLevel
	}
}

// Sync flushes any buffered log entries.
// Should be called before the application exits.
func Sync() {
	if Log != nil {
		_ = Log.Sync()
	}
}

// IsDevelopment returns true if running in a terminal (interactive mode)
// or if FLOWGAUGE_LOG_FORMAT=console is set (useful for systemd)
func IsDevelopment() bool {
	// Check environment variable for explicit console format
	if os.Getenv("FLOWGAUGE_LOG_FORMAT") == "console" {
		return true
	}
	// Auto-detect terminal
	fileInfo, _ := os.Stdout.Stat()
	return (fileInfo.Mode() & os.ModeCharDevice) != 0
}

// Helper functions for common logging patterns

// Debug logs a debug message
func Debug(msg string, fields ...zap.Field) {
	if Log != nil {
		Log.Debug(msg, fields...)
	}
}

// Info logs an info message
func Info(msg string, fields ...zap.Field) {
	if Log != nil {
		Log.Info(msg, fields...)
	}
}

// Warn logs a warning message
func Warn(msg string, fields ...zap.Field) {
	if Log != nil {
		Log.Warn(msg, fields...)
	}
}

// Error logs an error message
func Error(msg string, fields ...zap.Field) {
	if Log != nil {
		Log.Error(msg, fields...)
	}
}

// Fatal logs a fatal message and exits
func Fatal(msg string, fields ...zap.Field) {
	if Log != nil {
		Log.Fatal(msg, fields...)
	}
	os.Exit(1)
}

// With creates a child logger with additional fields
func With(fields ...zap.Field) *zap.Logger {
	if Log != nil {
		return Log.With(fields...)
	}
	return zap.NewNop()
}

// Named creates a named child logger
func Named(name string) *zap.Logger {
	if Log != nil {
		return Log.Named(name)
	}
	return zap.NewNop()
}

