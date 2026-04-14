package logger

import (
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var (
	// Logger is the global structured logger configured by Init.
	Logger zerolog.Logger
)

// Init configures the global logger with the given level and format.
// Invalid levels fall back to info, so startup can continue with sane defaults.
func Init(level string, format string) {
	parsedLevel, err := zerolog.ParseLevel(level)
	if err != nil {
		parsedLevel = zerolog.InfoLevel
	}

	zerolog.SetGlobalLevel(parsedLevel)
	zerolog.TimeFieldFormat = time.RFC3339

	if format == "console" {
		consoleWriter := zerolog.ConsoleWriter{
			Out:        os.Stderr,
			TimeFormat: time.RFC3339,
		}
		Logger = zerolog.New(consoleWriter).
			With().
			Timestamp().
			Logger()
	} else {
		Logger = zerolog.New(os.Stderr).
			With().
			Timestamp().
			Logger()
	}

	if level == "debug" {
		Logger = Logger.With().Caller().Logger()
	}

	log.Logger = Logger
}

// WithComponent returns a logger that always includes the component field.
func WithComponent(component string) zerolog.Logger {
	return Logger.With().Str("component", component).Logger()
}

// WithFields returns a logger that includes each provided structured field.
func WithFields(fields map[string]any) zerolog.Logger {
	ctx := Logger.With()
	for k, v := range fields {
		ctx = ctx.Interface(k, v)
	}
	return ctx.Logger()
}
