package logger

import (
	"bytes"
	"strings"
	"testing"

	"github.com/charmbracelet/log"
	"github.com/stretchr/testify/assert"
)

func TestSetLevel(t *testing.T) {
	tests := []struct {
		name     string
		level    string
		expected log.Level
	}{
		{"Debug", "debug", log.DebugLevel},
		{"Info", "info", log.InfoLevel},
		{"Warn", "warn", log.WarnLevel},
		{"Error", "error", log.ErrorLevel},
		{"Fatal", "fatal", log.FatalLevel},
		{"Panic", "panic", log.FatalLevel}, // Panic uses Fatal level
		{"Default", "invalid", log.InfoLevel},
		{"Empty", "", log.InfoLevel},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a logger with a buffer so we can validate level is set correctly
			var buf bytes.Buffer
			testLogger := log.NewWithOptions(&buf, log.Options{Level: log.DebugLevel})

			// Store original logger
			oldLogger := Log
			Log = testLogger

			// Set the level
			SetLevel(tt.level)

			// Verify the level is set correctly by checking if debug messages appear
			if tt.expected == log.DebugLevel {
				Log.Debug("test debug message")
				assert.Contains(t, buf.String(), "test debug message")
			} else {
				buf.Reset()
				Log.Debug("test debug message")
				assert.NotContains(t, buf.String(), "test debug message")
			}

			// Restore the original logger
			Log = oldLogger
		})
	}
}

func TestLogLevels(t *testing.T) {
	// Create tests for different log levels
	t.Run("DebugLevel", func(t *testing.T) {
		var buf bytes.Buffer
		testLogger := log.NewWithOptions(&buf, log.Options{Level: log.DebugLevel})

		// Store original logger
		oldLogger := Log
		Log = testLogger

		// At debug level, all messages should appear
		Log.Debug("debug test")
		assert.Contains(t, buf.String(), "debug test")

		buf.Reset()
		Log.Info("info test")
		assert.Contains(t, buf.String(), "info test")

		buf.Reset()
		Log.Warn("warn test")
		assert.Contains(t, buf.String(), "warn test")

		buf.Reset()
		Log.Error("error test")
		assert.Contains(t, buf.String(), "error test")

		// Restore original logger
		Log = oldLogger
	})

	t.Run("InfoLevel", func(t *testing.T) {
		var buf bytes.Buffer
		testLogger := log.NewWithOptions(&buf, log.Options{Level: log.InfoLevel})

		// Store original logger
		oldLogger := Log
		Log = testLogger

		// At info level, debug should not appear, others should
		Log.Debug("debug test")
		assert.NotContains(t, buf.String(), "debug test")

		buf.Reset()
		Log.Info("info test")
		assert.Contains(t, buf.String(), "info test")

		buf.Reset()
		Log.Warn("warn test")
		assert.Contains(t, buf.String(), "warn test")

		buf.Reset()
		Log.Error("error test")
		assert.Contains(t, buf.String(), "error test")

		// Restore original logger
		Log = oldLogger
	})

	t.Run("WarnLevel", func(t *testing.T) {
		var buf bytes.Buffer
		testLogger := log.NewWithOptions(&buf, log.Options{Level: log.WarnLevel})

		// Store original logger
		oldLogger := Log
		Log = testLogger

		// At warn level, debug and info should not appear
		Log.Debug("debug test")
		assert.NotContains(t, buf.String(), "debug test")

		buf.Reset()
		Log.Info("info test")
		assert.NotContains(t, buf.String(), "info test")

		buf.Reset()
		Log.Warn("warn test")
		assert.Contains(t, buf.String(), "warn test")

		buf.Reset()
		Log.Error("error test")
		assert.Contains(t, buf.String(), "error test")

		// Restore original logger
		Log = oldLogger
	})

	t.Run("ErrorLevel", func(t *testing.T) {
		var buf bytes.Buffer
		testLogger := log.NewWithOptions(&buf, log.Options{Level: log.ErrorLevel})

		// Store original logger
		oldLogger := Log
		Log = testLogger

		// At error level, only error should appear
		Log.Debug("debug test")
		assert.NotContains(t, buf.String(), "debug test")

		buf.Reset()
		Log.Info("info test")
		assert.NotContains(t, buf.String(), "info test")

		buf.Reset()
		Log.Warn("warn test")
		assert.NotContains(t, buf.String(), "warn test")

		buf.Reset()
		Log.Error("error test")
		assert.Contains(t, buf.String(), "error test")

		// Restore original logger
		Log = oldLogger
	})
}

func TestStyledLogging(t *testing.T) {
	// Create a buffer to capture output
	var buf bytes.Buffer

	// Store original logger
	oldLogger := Log

	// Create a test logger with debug level to capture all logs
	Log = log.NewWithOptions(&buf, log.Options{Level: log.DebugLevel})

	// Test error styled
	ErrorStyled("error styled message")
	assert.Contains(t, buf.String(), "error styled message")

	// Test warn styled
	buf.Reset()
	WarnStyled("warn styled message")
	assert.Contains(t, buf.String(), "warn styled message")

	// Test info styled
	buf.Reset()
	InfoStyled("info styled message")
	assert.Contains(t, buf.String(), "info styled message")

	// Test debug styled
	buf.Reset()
	DebugStyled("debug styled message")
	assert.Contains(t, buf.String(), "debug styled message")

	// Restore original logger
	Log = oldLogger
}

func TestInit(t *testing.T) {
	// This is mostly a smoke test since Init() currently doesn't do much
	// Create a buffer to capture output
	var buf bytes.Buffer

	// Store original logger
	oldLogger := Log

	// Create a test logger with configured level
	Log = log.NewWithOptions(&buf, log.Options{Level: log.InfoLevel})

	// Call Init which should maintain settings
	Init()

	// Verify level is maintained by checking debug visibility
	Log.Debug("debug message")
	assert.NotContains(t, buf.String(), "debug message")

	buf.Reset()
	Log.Info("info message")
	assert.Contains(t, buf.String(), "info message")

	// Restore original logger
	Log = oldLogger
}

func TestLogWithKeyValues(t *testing.T) {
	// Create a buffer to capture output
	var buf bytes.Buffer

	// Store original logger
	oldLogger := Log

	// Create a test logger with debug level to capture all logs
	Log = log.NewWithOptions(&buf, log.Options{Level: log.DebugLevel})

	// Test logging with key-value pairs
	Log.Info("message with context", "key1", "value1", "key2", 42)
	output := buf.String()

	assert.Contains(t, output, "message with context")
	assert.Contains(t, output, "key1")
	assert.Contains(t, output, "value1")
	assert.Contains(t, output, "key2")
	assert.Contains(t, output, "42")

	// Restore original logger
	Log = oldLogger
}

func TestLogFormattedMessage(t *testing.T) {
	// Create a buffer to capture output
	var buf bytes.Buffer

	// Store original logger
	oldLogger := Log

	// Create a test logger with info level
	Log = log.NewWithOptions(&buf, log.Options{Level: log.InfoLevel})

	// Test logging with formatted message using Infof
	Log.Infof("Formatted %s with %d values", "message", 2)

	assert.Contains(t, buf.String(), "Formatted message with 2 values")

	// Restore original logger
	Log = oldLogger
}

func TestChangingLogLevelsMultipleTimes(t *testing.T) {
	// Create a buffer to capture output
	var buf bytes.Buffer

	// Store original logger
	oldLogger := Log

	// Create a test logger with initial debug level
	Log = log.NewWithOptions(&buf, log.Options{Level: log.DebugLevel})

	// Test initial debug level
	Log.Debug("debug message 1")
	assert.Contains(t, buf.String(), "debug message 1")

	// Change to info level
	SetLevel(InfoLevel)
	buf.Reset()

	// Debug should not appear
	Log.Debug("debug message 2")
	assert.NotContains(t, buf.String(), "debug message 2")

	// But info should
	Log.Info("info message 2")
	assert.Contains(t, buf.String(), "info message 2")

	// Change to warn level
	SetLevel(WarnLevel)
	buf.Reset()

	// Info should not appear
	Log.Info("info message 3")
	assert.NotContains(t, buf.String(), "info message 3")

	// But warn should
	Log.Warn("warn message 3")
	assert.Contains(t, buf.String(), "warn message 3")

	// Change back to debug level
	SetLevel(DebugLevel)
	buf.Reset()

	// Debug should appear again
	Log.Debug("debug message 4")
	assert.Contains(t, buf.String(), "debug message 4")

	// Restore original logger
	Log = oldLogger
}

func TestComplexStructuredLogging(t *testing.T) {
	// Create a buffer to capture output
	var buf bytes.Buffer

	// Store original logger
	oldLogger := Log

	// Create a test logger with debug level
	Log = log.NewWithOptions(&buf, log.Options{Level: log.DebugLevel})

	// Create complex nested structures for logging
	complexMap := map[string]interface{}{
		"nested": map[string]interface{}{
			"array": []int{1, 2, 3},
			"map":   map[string]string{"a": "b", "c": "d"},
		},
		"simple": "value",
	}

	// Log with complex structure
	Log.Info("complex log", "data", complexMap)
	output := buf.String()

	// Check that the complex data is properly logged
	assert.Contains(t, output, "complex log")
	assert.Contains(t, output, "data=")
	assert.Contains(t, output, "nested")
	assert.Contains(t, output, "simple")

	// Restore original logger
	Log = oldLogger
}

func TestStyledLoggingWithSpecialCharacters(t *testing.T) {
	// Create a buffer to capture output
	var buf bytes.Buffer

	// Store original logger
	oldLogger := Log

	// Create a test logger with debug level
	Log = log.NewWithOptions(&buf, log.Options{Level: log.DebugLevel})

	// Test styled logging with special characters
	specialChars := "Special chars: !@#$%^&*()_+-=[]{}|;':\",./<>?\n\t"

	// Error styled with special chars
	ErrorStyled(specialChars)
	assert.Contains(t, buf.String(), "Special chars")

	// Warn styled with special chars
	buf.Reset()
	WarnStyled(specialChars)
	assert.Contains(t, buf.String(), "Special chars")

	// Info styled with special chars
	buf.Reset()
	InfoStyled(specialChars)
	assert.Contains(t, buf.String(), "Special chars")

	// Debug styled with special chars
	buf.Reset()
	DebugStyled(specialChars)
	assert.Contains(t, buf.String(), "Special chars")

	// Restore original logger
	Log = oldLogger
}

func TestLongMessages(t *testing.T) {
	// Create a buffer to capture output
	var buf bytes.Buffer

	// Store original logger
	oldLogger := Log

	// Create a test logger with debug level
	Log = log.NewWithOptions(&buf, log.Options{Level: log.DebugLevel})

	// Create a very long message (over 1000 characters)
	longMessage := strings.Repeat("This is a long message that will be repeated. ", 25)

	// Test logging with a very long message
	Log.Info(longMessage)

	// Verify the long message was logged correctly
	assert.Contains(t, buf.String(), "This is a long message")
	assert.True(t, len(buf.String()) > 1000, "Expected a long log message")

	// Restore original logger
	Log = oldLogger
}

func TestEmptyMessages(t *testing.T) {
	// Create a buffer to capture output
	var buf bytes.Buffer

	// Store original logger
	oldLogger := Log

	// Create a test logger with debug level
	Log = log.NewWithOptions(&buf, log.Options{Level: log.DebugLevel})

	// Test logging with empty messages
	Log.Info("")
	assert.NotEmpty(t, buf.String(), "Empty message should still produce log output")

	// Test styled logging with empty messages
	buf.Reset()
	ErrorStyled("")
	assert.NotEmpty(t, buf.String(), "Empty styled message should still produce log output")

	buf.Reset()
	WarnStyled("")
	assert.NotEmpty(t, buf.String(), "Empty styled message should still produce log output")

	buf.Reset()
	InfoStyled("")
	assert.NotEmpty(t, buf.String(), "Empty styled message should still produce log output")

	buf.Reset()
	DebugStyled("")
	assert.NotEmpty(t, buf.String(), "Empty styled message should still produce log output")

	// Restore original logger
	Log = oldLogger
}
