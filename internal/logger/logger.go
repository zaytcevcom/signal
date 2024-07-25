package logger

import (
	"io"
	"log/slog"
	"os"
	"strings"
)

type Logger struct {
	logger *slog.Logger
}

func New(level string, writer io.Writer) *Logger {
	if writer == nil {
		writer = os.Stdout
	}

	opts := slog.HandlerOptions{
		Level: getSlogLevel(level),
	}

	loggerJSON := slog.New(
		slog.NewJSONHandler(
			writer,
			&opts,
		),
	)

	return &Logger{loggerJSON}
}

func (l Logger) Debug(msg string) {
	l.logger.Debug(msg)
}

func (l Logger) Info(msg string) {
	l.logger.Info(msg)
}

func (l Logger) Warn(msg string) {
	l.logger.Warn(msg)
}

func (l Logger) Error(msg string) {
	l.logger.Error(msg)
}

func getSlogLevel(level string) slog.Level {
	level = strings.ToUpper(level)

	var slogLevel slog.Level
	switch level {
	case "DEBUG":
		slogLevel = slog.LevelDebug
	case "INFO":
		slogLevel = slog.LevelInfo
	case "WARN":
		slogLevel = slog.LevelWarn
	case "ERROR":
		slogLevel = slog.LevelError
	default:
		slogLevel = slog.LevelInfo
	}

	return slogLevel
}
