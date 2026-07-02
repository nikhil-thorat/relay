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

func New(level string) (*Logger, error) {

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
		return nil, fmt.Errorf("invalid log level : %s", level)
	}

	opts := &slog.HandlerOptions{
		Level: logLevel,
	}

	handler := slog.NewTextHandler(
		output,
		opts,
	)

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
