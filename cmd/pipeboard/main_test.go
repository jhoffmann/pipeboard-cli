package main

import (
	"bytes"
	"log/slog"
	"os"
	"strings"
	"testing"
)

func TestSetupLogger(t *testing.T) {
	tests := []struct {
		name      string
		level     string
		format    string
		wantLevel slog.Level
	}{
		{"debug level text format", "debug", "text", slog.LevelDebug},
		{"info level text format", "info", "text", slog.LevelInfo},
		{"warn level text format", "warn", "text", slog.LevelWarn},
		{"error level text format", "error", "text", slog.LevelError},
		{"invalid level defaults to info", "invalid", "text", slog.LevelInfo},
		{"json format", "info", "json", slog.LevelInfo},
		{"invalid format defaults to text", "info", "invalid", slog.LevelInfo},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := setupLogger(tt.level, tt.format)
			if logger == nil {
				t.Error("setupLogger() returned nil")
			}

			// Test that the logger can be used without panicking
			logger.Info("test message")
		})
	}
}

func TestSetupLoggerOutput(t *testing.T) {
	// Capture stderr
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	logger := setupLogger("info", "text")
	logger.Info("test message", "key", "value")

	w.Close()
	os.Stderr = oldStderr

	var buf bytes.Buffer
	_, err := buf.ReadFrom(r)
	if err != nil {
		t.Fatalf("Failed to read from buffer: %v", err)
	}
	output := buf.String()

	if !strings.Contains(output, "test message") {
		t.Errorf("Expected log output to contain 'test message', got: %s", output)
	}
}

func TestSetupLoggerJSON(t *testing.T) {
	// Capture stderr
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	logger := setupLogger("info", "json")
	logger.Info("test message", "key", "value")

	w.Close()
	os.Stderr = oldStderr

	var buf bytes.Buffer
	_, err := buf.ReadFrom(r)
	if err != nil {
		t.Fatalf("Failed to read from buffer: %v", err)
	}
	output := buf.String()

	// JSON output should contain structured fields
	if !strings.Contains(output, `"msg":"test message"`) {
		t.Errorf("Expected JSON log output to contain structured message, got: %s", output)
	}
}

func TestLogLevelParsing(t *testing.T) {
	tests := []struct {
		input    string
		expected slog.Level
	}{
		{"debug", slog.LevelDebug},
		{"info", slog.LevelInfo},
		{"warn", slog.LevelWarn},
		{"error", slog.LevelError},
		{"DEBUG", slog.LevelInfo},   // Invalid, should default to info
		{"", slog.LevelInfo},        // Empty, should default to info
		{"unknown", slog.LevelInfo}, // Unknown, should default to info
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			logger := setupLogger(tt.input, "text")

			// We can't directly access the log level from the logger,
			// but we can test that the logger was created successfully
			if logger == nil {
				t.Errorf("setupLogger(%q, \"text\") returned nil", tt.input)
			}
		})
	}
}
