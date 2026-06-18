package logger

import (
	"io"
	"os"
	"time"

	"github.com/rs/zerolog"
)

var log zerolog.Logger

func init() {
	log = zerolog.New(
		zerolog.ConsoleWriter{
			Out:        os.Stderr,
			TimeFormat: time.RFC3339,
			NoColor:    false,
		},
	).With().Timestamp().Logger()
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
}

// SetLevel adjusts the global log level.
func SetLevel(level zerolog.Level) {
	zerolog.SetGlobalLevel(level)
}

// SetOutput redirects log output.
func SetOutput(w io.Writer) {
	log = log.Output(w)
}

// Debug logs a debug message.
func Debug() *zerolog.Event { return log.Debug() }

// Info logs an informational message.
func Info() *zerolog.Event { return log.Info() }

// Warn logs a warning.
func Warn() *zerolog.Event { return log.Warn() }

// Error logs an error.
func Error() *zerolog.Event { return log.Error() }

// Fatal logs a fatal message and exits.
func Fatal() *zerolog.Event { return log.Fatal() }

// With returns a context logger for adding fields.
func With() zerolog.Context { return log.With() }

// SetQuiet disables all output below error level.
func SetQuiet() {
	zerolog.SetGlobalLevel(zerolog.ErrorLevel)
}

// SetVerbose enables debug output.
func SetVerbose() {
	zerolog.SetGlobalLevel(zerolog.DebugLevel)
}
