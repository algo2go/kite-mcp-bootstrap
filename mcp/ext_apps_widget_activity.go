package mcp

import (
	"context"

	"github.com/algo2go/kite-mcp-audit"
	"github.com/algo2go/kite-mcp-cqrs"
)

// activityData fetches recent audit trail entries via the activity widget use case.
//
// Closes the last production CQRS escape: previously a "defensive fallback"
// branch for nil manager existed, but no test or production caller ever hit
// it (TestActivityData_NoAuditStore short-circuits at the nil-auditStore
// guard above). Removing the fallback makes every widget-activity read go
// through the QueryBus uniformly.
func activityData(ctx context.Context, manager extAppManagerPort, auditStore *audit.Store, email string) any {
	if auditStore == nil || manager == nil {
		return nil
	}
	ctx = cqrs.WithWidgetAuditStore(ctx, auditStore)
	result, err := manager.QueryBus().DispatchWithResult(ctx, cqrs.GetActivityForWidgetQuery{Email: email})
	if err != nil {
		return nil
	}
	return result
}
