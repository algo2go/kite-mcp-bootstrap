package mcp

import (
	"context"

	"github.com/algo2go/kite-mcp-audit"
	"github.com/algo2go/kite-mcp-cqrs"
)

// alertsData fetches active/triggered alerts via the alerts widget use case.
func alertsData(ctx context.Context, manager extAppManagerPort, _ *audit.Store, email string) any {
	if manager.AlertStore() == nil {
		return nil
	}
	result, err := manager.QueryBus().DispatchWithResult(ctx, cqrs.GetAlertsForWidgetQuery{Email: email})
	if err != nil {
		return nil
	}
	return result
}
