package mcp

import (
	"context"

	"github.com/algo2go/kite-mcp-audit"
)

// chartData returns nil because the chart widget boots into an idle state.
// The user picks the symbol interactively and loads data via AppBridge calls
// to search_instruments, get_historical_data, and technical_indicators.
func chartData(_ context.Context, _ extAppManagerPort, _ *audit.Store, _ string) any {
	return nil
}
