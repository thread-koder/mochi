package logger

import (
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var (
	// Global logger instance
	Logger zerolog.Logger
)

// Initializes the logger with the specified level and format
func Init(level string, format string) {
	// Parse log level
	logLevel, err := zerolog.ParseLevel(level)
	if err != nil {
		logLevel = zerolog.InfoLevel
	}

	// Set global log level
	zerolog.SetGlobalLevel(logLevel)

	// Set time field format
	zerolog.TimeFieldFormat = time.RFC3339

	// Format selection
	if format == "console" {
		// Console output formatter
		output := zerolog.ConsoleWriter{
			Out:        os.Stderr,
			TimeFormat: time.RFC3339,
		}
		Logger = zerolog.New(output).
			With().
			Timestamp().
			Caller().
			Logger()
	} else {
		// JSON output formatter
		Logger = zerolog.New(os.Stderr).
			With().
			Timestamp().
			Caller().
			Logger()
	}

	// Set as global logger
	log.Logger = Logger
}

// Returns a logger instance with a component field
func WithComponent(component string) zerolog.Logger {
	return Logger.With().Str("component", component).Logger()
}

// Returns a logger instance with additional fields
func WithFields(fields map[string]any) zerolog.Logger {
	ctx := Logger.With()
	for k, v := range fields {
		ctx = ctx.Interface(k, v)
	}
	return ctx.Logger()
}
