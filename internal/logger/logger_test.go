package logger

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLogger(t *testing.T) {
	tests := []struct {
		name            string
		level           string
		loggingFunc     func(l *Logger)
		expectedMessage string
	}{
		{
			name:            "Debug",
			level:           "debug",
			loggingFunc:     func(l *Logger) { l.Debug("debug message") },
			expectedMessage: "debug message",
		},

		{
			name:            "Info",
			level:           "info",
			loggingFunc:     func(l *Logger) { l.Info("info message") },
			expectedMessage: "info message",
		},
		{
			name:            "Warn",
			level:           "warn",
			loggingFunc:     func(l *Logger) { l.Warn("warn message") },
			expectedMessage: "warn message",
		},
		{
			name:            "Error",
			level:           "error",
			loggingFunc:     func(l *Logger) { l.Error("error message") },
			expectedMessage: "error message",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := &bytes.Buffer{}
			logger := New(tt.level, buf)
			tt.loggingFunc(logger)
			assert.Contains(t, buf.String(), tt.expectedMessage)
		})
	}
}
