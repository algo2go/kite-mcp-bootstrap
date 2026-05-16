package mcp

import (
	"context"

	"github.com/algo2go/kite-mcp-audit"
)

// orderFormData returns paper-mode status for the order form widget.
// Margins are fetched dynamically via callServerTool('order_risk_report')
// rather than pre-injected, since the form needs fresh data at submission time.
func orderFormData(_ context.Context, manager extAppManagerPort, _ *audit.Store, email string) any {
	paperMode := false
	if engine := manager.PaperEngine(); engine != nil {
		paperMode = engine.IsEnabled(email)
	}
	return map[string]any{
		"paper_mode": paperMode,
	}
}
