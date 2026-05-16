package bootstrap

import (
	"context"
	"log/slog"
	"math"
	"os"
	"runtime/debug"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMemoryLimitConstant verifies the runtime memory limit is set to a
// sane value (450 MB). Path C item per audit 6ee6520: prevents OOM-kill
// on the 512 MB Fly.io machine by giving Go's GC a soft target below
// kernel OOM-killer threshold.
//
// We assert two invariants:
//  1. MemoryLimitBytes is the documented 450 MB constant
//  2. debug.SetMemoryLimit retrieved-via-(-1) reflects an applied limit
//     equal to MemoryLimitBytes (proves init() ran)
//
// Migrated 2026-05-16 from kite-mcp-server's main_test.go as part of the
// bootstrap-relocation dispatch.
func TestMemoryLimitConstant(t *testing.T) {
	t.Parallel()

	const expected int64 = 450 * 1024 * 1024
	assert.Equal(t, expected, MemoryLimitBytes,
		"MemoryLimitBytes must be 450 MB to leave 62 MB headroom on 512 MB Fly.io machine")

	current := debug.SetMemoryLimit(-1)
	assert.Less(t, current, int64(math.MaxInt64),
		"runtime memory limit must be lowered from default by init()")
	if os.Getenv("GOMEMLIMIT") == "" {
		assert.Equal(t, MemoryLimitBytes, current,
			"with no GOMEMLIMIT env override, runtime limit must equal MemoryLimitBytes")
	}
}

// TestParseLogLevel covers the pure parser. Migrated 2026-05-16 from
// kite-mcp-server's main_test.go.
func TestParseLogLevel(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		raw  string
		want slog.Level
	}{
		{"empty defaults to info", "", slog.LevelInfo},
		{"info explicit", "info", slog.LevelInfo},
		{"debug", "debug", slog.LevelDebug},
		{"warn", "warn", slog.LevelWarn},
		{"error", "error", slog.LevelError},
		{"garbage defaults to info (fail-open)", "garbage", slog.LevelInfo},
		{"uppercase NOT recognised (fail-open to info)", "DEBUG", slog.LevelInfo},
		{"whitespace NOT trimmed (current contract)", " info", slog.LevelInfo},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, ParseLogLevel(tc.raw))
		})
	}
}

// TestInitLogger_EnvIntegration is the single adapter test that verifies
// InitLogger reads LOG_LEVEL from the environment and feeds it to
// ParseLogLevel. Cannot run in parallel because t.Setenv mutates process-
// wide env.
func TestInitLogger_EnvIntegration(t *testing.T) {
	t.Setenv("LOG_LEVEL", "debug")
	logger, logBuffer := InitLogger()
	require.NotNil(t, logger)
	require.NotNil(t, logBuffer)
	assert.True(t, logger.Enabled(context.Background(), slog.LevelDebug))
}

// TestInitLogger_LogBufferCaptures verifies the LogBuffer side-effect of
// InitLogger (separate from level parsing).
func TestInitLogger_LogBufferCaptures(t *testing.T) {
	t.Parallel()
	logger, logBuffer := InitLogger()
	logger.Error("test-message-for-buffer")
	entries := logBuffer.Recent(10)
	found := false
	for _, e := range entries {
		if strings.Contains(e.Message, "test-message-for-buffer") {
			found = true
			break
		}
	}
	assert.True(t, found, "LogBuffer should capture log entries")
}
