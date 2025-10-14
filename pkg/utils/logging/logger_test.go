package logging_test

import (
	"bytes"
	"context"
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/m-mizutani/leveret/pkg/utils/logging"
)

func TestNew(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := logging.New("info", buf)
	gt.V(t, logger).NotNil()

	logger.Info("test message")
	gt.S(t, buf.String()).Contains("test message")
}

func TestNewWithDifferentLevels(t *testing.T) {
	testCases := []struct {
		level       string
		expectDebug bool
		expectInfo  bool
		expectWarn  bool
		expectError bool
	}{
		{"debug", true, true, true, true},
		{"info", false, true, true, true},
		{"warn", false, false, true, true},
		{"warning", false, false, true, true},
		{"error", false, false, false, true},
		{"DEBUG", true, true, true, true}, // Case-insensitive
		{"invalid", false, true, true, true}, // Defaults to info
	}

	for _, tc := range testCases {
		t.Run(tc.level, func(t *testing.T) {
			buf := &bytes.Buffer{}
			logger := logging.New(tc.level, buf)
			gt.V(t, logger).NotNil()

			logger.Debug("debug message")
			logger.Info("info message")
			logger.Warn("warn message")
			logger.Error("error message")

			output := buf.String()
			if tc.expectDebug {
				gt.S(t, output).Contains("debug message")
			} else {
				gt.S(t, output).NotContains("debug message")
			}
			if tc.expectInfo {
				gt.S(t, output).Contains("info message")
			} else {
				gt.S(t, output).NotContains("info message")
			}
			if tc.expectWarn {
				gt.S(t, output).Contains("warn message")
			} else {
				gt.S(t, output).NotContains("warn message")
			}
			if tc.expectError {
				gt.S(t, output).Contains("error message")
			} else {
				gt.S(t, output).NotContains("error message")
			}
		})
	}
}

func TestWithAndFrom(t *testing.T) {
	ctx := context.Background()
	buf := &bytes.Buffer{}
	logger := logging.New("debug", buf)

	// Test With
	ctx = logging.With(ctx, logger)

	// Test From - should return the same logger
	retrieved := logging.From(ctx)
	gt.V(t, retrieved).NotNil()
	gt.Equal(t, retrieved, logger)

	// Verify logging works
	retrieved.Info("context message")
	gt.S(t, buf.String()).Contains("context message")
}

func TestFromWithoutLogger(t *testing.T) {
	ctx := context.Background()

	// Test From without a logger - should return default logger
	logger := logging.From(ctx)
	gt.V(t, logger).NotNil()
}

func TestFromWithCustomLogger(t *testing.T) {
	ctx := context.Background()
	buf := &bytes.Buffer{}
	customLogger := logging.New("info", buf).With("component", "test")

	ctx = logging.With(ctx, customLogger)
	retrieved := logging.From(ctx)

	gt.V(t, retrieved).NotNil()
	gt.Equal(t, retrieved, customLogger)

	// Verify component is included in output
	retrieved.Info("custom message")
	output := buf.String()
	gt.S(t, output).Contains("custom message")
	gt.S(t, output).Contains("component")
	gt.S(t, output).Contains("test")
}

func TestDefault(t *testing.T) {
	logger := logging.Default()
	gt.V(t, logger).NotNil()
}

func TestSetDefault(t *testing.T) {
	// Get original default
	original := logging.Default()

	// Create and set new default
	buf := &bytes.Buffer{}
	newLogger := logging.New("debug", buf)
	logging.SetDefault(newLogger)

	// Verify new default
	retrieved := logging.Default()
	gt.Equal(t, retrieved, newLogger)

	// Verify logging works with new default
	retrieved.Info("default message")
	gt.S(t, buf.String()).Contains("default message")

	// Restore original
	logging.SetDefault(original)
}

func TestFromUsesDefault(t *testing.T) {
	ctx := context.Background()

	// Get original default
	original := logging.Default()

	// Set custom default
	buf := &bytes.Buffer{}
	customDefault := logging.New("warn", buf)
	logging.SetDefault(customDefault)

	// From should return the custom default when no logger in context
	retrieved := logging.From(ctx)
	gt.Equal(t, retrieved, customDefault)

	// Verify it's the custom default by logging
	retrieved.Warn("warning from default")
	gt.S(t, buf.String()).Contains("warning from default")

	// Restore original default
	logging.SetDefault(original)
}
