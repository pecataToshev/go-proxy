package logger

import (
	"log/slog"
	"os"
)

// Init sets up the application's global logger.
// If debug is true, it sets the log level to Debug, otherwise Info.
// If structured is true, it outputs in JSON format, otherwise in Text format.
func Init(debug bool, structured bool) {
	var level slog.Level
	if debug {
		level = slog.LevelDebug
	} else {
		level = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{
		Level: level,
	}

	var handler slog.Handler
	if structured {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		handler = slog.NewTextHandler(os.Stdout, opts)
	}

	logger := slog.New(handler)
	slog.SetDefault(logger)
}

// Info logs at LevelInfo.
func Info(msg string, args ...any) {
	slog.Info(msg, args...)
}

// Debug logs at LevelDebug.
func Debug(msg string, args ...any) {
	slog.Debug(msg, args...)
}

// Warn logs at LevelWarn.
func Warn(msg string, args ...any) {
	slog.Warn(msg, args...)
}

// Error logs at LevelError.
func Error(msg string, args ...any) {
	slog.Error(msg, args...)
}

// Fatal logs at LevelError and then exits.
func Fatal(msg string, args ...any) {
	slog.Error(msg, args...)
	os.Exit(1)
}
