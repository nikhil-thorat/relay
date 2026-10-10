package logging

import (
	"fmt"
	"io"
	"log/slog"
	"os"
)

type Logger struct {
	*slog.Logger
}

func New(level string, format string) (*Logger, error) {

	var output io.Writer = os.Stdout
	var logLevel slog.Level

	switch level {
	case "debug":
		logLevel = slog.LevelDebug
	case "info":
		logLevel = slog.LevelInfo
	case "warn":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	case "off":
		output = io.Discard
		logLevel = slog.LevelError
	default:
		return nil, fmt.Errorf("invalid log level: %s", level)
	}

	if format == "" {
		format = "text"
	}

	opts := &slog.HandlerOptions{
		Level: logLevel,
	}

	var handler slog.Handler
	switch format {
	case "text":
		handler = slog.NewTextHandler(output, opts)
	case "json":
		handler = slog.NewJSONHandler(output, opts)
	default:
		return nil, fmt.Errorf("invalid log format: %s", format)
	}

	logger := &Logger{
		slog.New(handler),
	}

	return logger, nil
}

func (logger *Logger) WithComponent(component string) *Logger {
	child := logger.With(
		"component",
		component,
	)

	return &Logger{
		Logger: child,
	}
}
