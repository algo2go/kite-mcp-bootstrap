package mcp

import (
	"context"

	"github.com/algo2go/kite-mcp-audit"
)

// optionsChainData returns nil because the options chain widget boots into an
// idle state. The user picks the underlying interactively and loads data via
// AppBridge calls to get_option_chain / options_greeks.
func optionsChainData(_ context.Context, manager extAppManagerPort, _ *audit.Store, email string) any {
	return nil
}
