// Package bootstrap is the composition root entry point for the
// kite-mcp-server stack. Deploy repos import this package and call
// Main(Options{}) from their thin main.go.
//
// All composition + DI wiring + HTTP serving lives in subpackages:
//   - app/         : Fx wiring + HTTP mux + lifecycle
//   - kc/          : Kite client manager + sessions + credential store
//   - kc/ops/      : Admin + user dashboards + scanner + payoff
//   - mcp/         : MCP tool registrations + middleware
//   - plugins/     : plugin scaffolding sub-module
//   - testutil/    : test fixtures sub-module
package bootstrap

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"runtime/debug"
	"sync/atomic"

	"github.com/algo2go/kite-mcp-bootstrap/app"
	"github.com/algo2go/kite-mcp-kc/ops"
)

// MemoryLimitBytes is the soft GC target for the Go runtime — set via
// runtime/debug.SetMemoryLimit at package init. Path C item per the
// kite-mcp-server audit 6ee6520: prevents OOM-kill on the 512MB Fly.io
// machine.
//
// 450 MB target leaves ~62 MB headroom on a 512 MB machine — the
// empirical industry standard is 85-90% of available RAM, so 450/512 =
// 88% sits in the conservative upper-band.
//
// GOMEMLIMIT env var (read at runtime startup) overrides this — the
// in-code default is intentionally visible for ops audit + debugging.
const MemoryLimitBytes int64 = 450 * 1024 * 1024 // 450 MB

func init() {
	// SetMemoryLimit caps GC target to 450 MB on the 512 MB Fly.io machine.
	// Without this, sustained allocation can outpace GC and trigger kernel
	// OOM-kill (full outage vs smooth back-pressure).
	//
	// Safe in init(): runtime/debug is stdlib, no third-party deps, no
	// goroutines spawned, no heap allocations of consequence. Runs before
	// any consumer's main() because Go runs imported-package inits before
	// the importing package's main.
	_ = debug.SetMemoryLimit(MemoryLimitBytes)
}

// Options are the runtime parameters a deploy repo's main.go injects when
// calling Main. Both fields default to dev placeholders when empty.
type Options struct {
	// Version is the MCP_SERVER_VERSION string (typically set via ldflags
	// at build time: -X main.MCP_SERVER_VERSION=v1.2.3).
	Version string
	// BuildString is the human-readable build identifier (typically set
	// via ldflags at build time: -X 'main.buildString=2026-05-16 abc123').
	BuildString string
}

// ParseLogLevel maps a raw LOG_LEVEL env value to a slog.Level. Empty
// string or unrecognised values default to LevelInfo. Pure function.
//
// Valid input values: "debug", "info", "warn", "error", "". Anything else
// also defaults to info; the "default to INFO if invalid" branch is fail-
// open: a typo'd LOG_LEVEL must not silence logs.
func ParseLogLevel(raw string) slog.Level {
	switch raw {
	case "debug":
		return slog.LevelDebug
	case "info", "":
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// InitLogger constructs the production logger + log buffer pair. LOG_LEVEL
// env var controls the slog level via ParseLogLevel. The returned
// *ops.LogBuffer captures the last 500 log entries for the ops dashboard
// log streaming endpoint.
func InitLogger() (*slog.Logger, *ops.LogBuffer) {
	level := ParseLogLevel(os.Getenv("LOG_LEVEL"))
	opts := &slog.HandlerOptions{Level: level}
	logBuffer := ops.NewLogBuffer(500)
	inner := slog.NewTextHandler(os.Stderr, opts)
	tee := ops.NewTeeHandler(inner, logBuffer)
	return slog.New(tee), logBuffer
}

// Main is the composition-root entry point. Deploy repos call this from
// their thin main.go after parsing --version flags themselves.
//
// Main blocks until shutdown (SIGINT/SIGTERM/SIGUSR2 graceful restart).
// Returns os.Exit code: 0 on clean shutdown, 1 on config-load failure or
// server start failure.
//
// Contract: Main does NOT call os.Exit itself — callers should:
//
//	func main() {
//	    os.Exit(bootstrap.Main(bootstrap.Options{
//	        Version: MCP_SERVER_VERSION,
//	        BuildString: buildString,
//	    }))
//	}
func Main(o Options) int {
	logger, logBuffer := InitLogger()

	application := app.NewApp(logger)
	application.SetLogBuffer(logBuffer)

	if err := application.LoadConfig(); err != nil {
		logger.Error("Failed to load configuration", "error", err)
		return 1
	}
	application.SetVersion(o.Version)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var activeRequests atomic.Int32
	app.StartGracefulRestartListener(ctx,
		app.GracefulRestartConfig{}.WithDefaults(),
		&activeRequests,
		logger,
		func() { application.TriggerShutdown() })

	logger.Info("Starting Kite MCP Server...",
		"version", o.Version,
		"build", o.BuildString)
	if err := application.RunServer(); err != nil {
		logger.Error("Server failed to start", "error", err)
		return 1
	}
	return 0
}

// PrintVersion writes version + build info to stdout — exposed so deploy
// repos can implement their own --version flag in main without importing
// fmt themselves.
func PrintVersion(o Options) {
	fmt.Printf("Kite MCP Server %s\n", o.Version)
	fmt.Printf("Build: %s\n", o.BuildString)
}
