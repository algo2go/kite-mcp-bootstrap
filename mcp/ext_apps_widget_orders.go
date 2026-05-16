package mcp

import (
	"context"

	"github.com/algo2go/kite-mcp-audit"
	"github.com/algo2go/kite-mcp-cqrs"
)

// ordersData fetches recent order entries via the orders widget use case.
func ordersData(ctx context.Context, manager extAppManagerPort, auditStore *audit.Store, email string) any {
	if auditStore == nil {
		return nil
	}
	ctx = cqrs.WithWidgetAuditStore(ctx, auditStore)
	result, err := manager.QueryBus().DispatchWithResult(ctx, cqrs.GetOrdersForWidgetQuery{Email: email})
	if err != nil {
		return nil
	}
	return result
}
